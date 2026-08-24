package application

import (
	"context"
	"sync"
	"testing"

	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/google/uuid"
)

// newTestApp wires an Application against the in-memory fakes
// (fake_repo_test.go). Per docs/testing/strategy.md's sanctioned-fakes
// decision, this file exists only for tests that need deterministic
// control over concurrency or commit/notify ordering — properties a real
// database and network round-trip make flaky to assert on directly.
// State-transition correctness (the happy path, idempotent replay,
// rejecting an already-terminal command) is proved exclusively against
// real persistence in integration_test.go; a fake is never the sole
// evidence for that claim, so those cases were retired from here rather
// than duplicated.
func newTestApp() (*Application, chan uuid.UUID) {
	notify := make(chan uuid.UUID, 8)
	app := &Application{
		Repo:     newFakeRepo(),
		Pending:  fakePending{},
		Exec:     newFakeExec(),
		Fixtures: fixtures.NewRegistry(),
		Notify:   notify,
	}
	return app, notify
}

func TestStartWorkflow_ConcurrentSameVersion_OneAcceptedOneConflict(t *testing.T) {
	app, _ := newTestApp()
	repo := app.Repo.(*fakeRepo)

	created := app.CreateWorkflow(context.Background(), CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String(), RequestingPrincipalID: uuid.New()})
	if created.Outcome != Accepted {
		t.Fatalf("setup: create outcome = %s", created.Outcome)
	}
	workflowID := uuid.MustParse(created.Workflow.WorkflowID)

	// StartWorkflow's pipeline is Load -> Kernel-validate -> Governance ->
	// CommitTransition, with no atomicity between Load and Commit (that gap
	// is exactly what CommitTransition's own compare-and-swap exists to
	// police). Two concurrent callers racing for the same ExpectedVersion
	// can interleave two different, individually-correct ways:
	//
	//  1. Both Load before either Commits: both reach CommitTransition with
	//     ExpectedVersion == the version they both saw; the CAS lets exactly
	//     one through and the other gets ports.ErrConflict -> Outcome=Conflict.
	//  2. One caller's entire Load-Validate-Commit completes before the
	//     other even calls Load: the second caller's Load observes the
	//     already-bumped version, so Kernel-level staleness validation
	//     (ValidateStartProposal) rejects it *before* CommitTransition is
	//     ever reached -> Outcome=Rejected, reasons
	//     [ILLEGAL_STATE VERSION_MISMATCH], not Conflict.
	//
	// Both are correct optimistic-concurrency outcomes — no state is ever
	// corrupted or double-committed in either case — but they're different
	// Outcome values, and which one happens depends on real goroutine
	// scheduling. Left to the scheduler, that made this assertion flaky
	// (measured ~10% of runs failing under `go test -count=500`). The gate
	// below forces every run into interleaving 1, which is what this test
	// specifically names and exercises: the storage layer's conflict path.
	repo.loadGate = newRendezvous(2)

	var wg sync.WaitGroup
	results := make([]Result, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = app.StartWorkflow(context.Background(), StartWorkflowRequest{
				RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
				WorkflowID: workflowID, ExpectedVersion: created.Workflow.Version, RequestingPrincipalID: uuid.New(),
			})
		}(i)
	}
	wg.Wait()

	accepted, conflict := 0, 0
	for _, r := range results {
		switch r.Outcome {
		case Accepted:
			accepted++
		case Conflict:
			conflict++
		default:
			t.Errorf("unexpected outcome: %s (reasons: %v)", r.Outcome, r.Reasons)
		}
	}
	if accepted != 1 || conflict != 1 {
		t.Fatalf("got accepted=%d conflict=%d, want exactly one of each", accepted, conflict)
	}
}

func TestStartWorkflow_NotifiesOnlyAfterCommit(t *testing.T) {
	app, notify := newTestApp()

	created := app.CreateWorkflow(context.Background(), CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String(), RequestingPrincipalID: uuid.New()})
	workflowID := uuid.MustParse(created.Workflow.WorkflowID)

	// CREATE_WORKFLOW causes no ExecutionIntent (only START_WORKFLOW does),
	// so nothing should be on the channel yet.
	select {
	case id := <-notify:
		t.Fatalf("unexpected notify after CreateWorkflow: %s", id)
	default:
	}

	res := app.StartWorkflow(context.Background(), StartWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: created.Workflow.Version, RequestingPrincipalID: uuid.New(),
	})
	if res.Outcome != Accepted {
		t.Fatalf("start outcome = %s, want ACCEPTED (reasons: %v)", res.Outcome, res.Reasons)
	}

	select {
	case <-notify:
	default:
		t.Fatal("expected a notify after StartWorkflow committed its ExecutionIntent")
	}
}
