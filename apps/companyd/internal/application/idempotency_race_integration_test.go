package application

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// verificationPool opens an independent connection to the same database
// requireRealApp already validated is reachable, purely to assert ground
// truth (real rows committed to Postgres) rather than trusting only the
// in-memory Results a race returns — a bug in the reservation scheme could
// in principle report a correct-looking Result while still having written
// a duplicate row, and only a direct query rules that out.
func verificationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("verification pool connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func countWorkflowsForOrg(t *testing.T, orgID uuid.UUID) int {
	t.Helper()
	var count int
	if err := verificationPool(t).QueryRow(context.Background(),
		`SELECT count(*) FROM workflows WHERE organization_id=$1`, orgID).Scan(&count); err != nil {
		t.Fatalf("count workflows: %v", err)
	}
	return count
}

func countApprovalsForOrg(t *testing.T, orgID uuid.UUID) int {
	t.Helper()
	var count int
	if err := verificationPool(t).QueryRow(context.Background(),
		`SELECT count(*) FROM approvals WHERE organization_id=$1`, orgID).Scan(&count); err != nil {
		t.Fatalf("count approvals: %v", err)
	}
	return count
}

// TestIntegration_CreateWorkflow_ConcurrentSameIdempotencyKey_OneWorkflow is
// the load-bearing regression test for the idempotency-reservation fix
// (docs/audit/gap-approval-flow-durability.md's idempotency-key finding):
// before the fix, application.go's replay/store guard was a
// lookup-then-write-then-store sequence with no atomicity between the
// lookup and the domain write, so N concurrent callers with the same key
// could all miss the lookup and all execute CreateWorkflow — each minting
// its own uuid.New() WorkflowID (workflow_create.go) — leaving N Workflow
// rows for one logical request. IdempotencyReserve's single atomic upsert
// (workflow_repo.go) closes that: only one caller can ever win the
// reservation and actually create a Workflow; every other racer either
// sees Indeterminate (reservation still in flight when it raced in) or
// replays the winner's own terminal ACCEPTED outcome (if it raced in after
// finalize already ran) — never a second, independent creation.
func TestIntegration_CreateWorkflow_ConcurrentSameIdempotencyKey_OneWorkflow(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	key := uuid.New().String()

	const racers = 8
	var wg sync.WaitGroup
	results := make([]Result, racers)
	wg.Add(racers)
	for i := 0; i < racers; i++ {
		go func(i int) {
			defer wg.Done()
			results[i] = app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: key, RequestingPrincipalID: app.Fixtures.TriggerPrincipal().PrincipalID})
		}(i)
	}
	wg.Wait()

	// Exactly one racer can have actually won the reservation and created a
	// real Workflow (Result.Workflow != nil, since only the winning path
	// populates it — replay.md's documented "fast path stays cheap"
	// tradeoff, see TestIntegration_CreateWorkflow_IdempotentReplayReturnsSameOutcome).
	// Every other racer must be either Indeterminate (raced in while the
	// reservation was still IN_PROGRESS) or a replay of the same terminal
	// ACCEPTED outcome with no Workflow attached (raced in after finalize)
	// — never a second real creation, and never any other outcome.
	wonAccepted, replayedAccepted, indeterminate := 0, 0, 0
	var wonWorkflowID string
	for i, res := range results {
		switch res.Outcome {
		case Accepted:
			if res.Workflow != nil {
				wonAccepted++
				wonWorkflowID = res.Workflow.WorkflowID
			} else {
				replayedAccepted++
			}
		case Indeterminate:
			indeterminate++
		default:
			t.Fatalf("racer %d: unexpected outcome %s (reasons: %v) among concurrent same-key CreateWorkflow racers", i, res.Outcome, res.Reasons)
		}
	}
	if wonAccepted != 1 {
		t.Fatalf("racers that actually created a Workflow = %d, want exactly 1 (wonWorkflowID=%q)", wonAccepted, wonWorkflowID)
	}
	if wonAccepted+replayedAccepted+indeterminate != racers {
		t.Fatalf("wonAccepted(%d)+replayedAccepted(%d)+indeterminate(%d) != racers(%d)", wonAccepted, replayedAccepted, indeterminate, racers)
	}

	if got := countWorkflowsForOrg(t, app.Fixtures.Organization().OrganizationID); got != 1 {
		t.Fatalf("workflows rows for this org (ground truth, not just in-memory Results) = %d, want exactly 1", got)
	}
}

// TestIntegration_CreateWorkflow_DifferentIdempotencyKeys_TwoWorkflows
// proves the flip side is correct BY DESIGN, not a bug: two distinct
// idempotency keys are two distinct logical commands, so — unlike the same-
// key case above — both are expected to independently create their own
// Workflow. The idempotency-reservation scheme this pass adds only
// deduplicates retries of the identical key; it deliberately does not
// attempt content-based deduplication of "the same logical action sent
// twice under two different keys" — that would require the client to have
// reused its key on retry in the first place, which is the actual contract
// (see docs/testing/concurrency-guarantees.md).
func TestIntegration_CreateWorkflow_DifferentIdempotencyKeys_TwoWorkflows(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()

	first := app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String(), RequestingPrincipalID: app.Fixtures.TriggerPrincipal().PrincipalID})
	second := app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String(), RequestingPrincipalID: app.Fixtures.TriggerPrincipal().PrincipalID})

	if first.Outcome != Accepted || first.Workflow == nil {
		t.Fatalf("first call outcome = %s (reasons: %v), want ACCEPTED with a Workflow", first.Outcome, first.Reasons)
	}
	if second.Outcome != Accepted || second.Workflow == nil {
		t.Fatalf("second call outcome = %s (reasons: %v), want ACCEPTED with a Workflow", second.Outcome, second.Reasons)
	}
	if first.Workflow.WorkflowID == second.Workflow.WorkflowID {
		t.Fatalf("two different idempotency keys produced the same WorkflowID %s — they must be independent commands", first.Workflow.WorkflowID)
	}
	if got := countWorkflowsForOrg(t, app.Fixtures.Organization().OrganizationID); got != 2 {
		t.Fatalf("workflows rows for this org = %d, want exactly 2 (one per independent key)", got)
	}
}

// TestIntegration_CancelWorkflow_ApprovalResolutionRacesCommandRetry covers
// the user-facing shape of scenario 5: a client that received
// APPROVAL_REQUIRED retries its original CancelWorkflow call (same
// idempotency key, a fresh RequestID — exactly what a real retry after a
// network timeout looks like) at the same moment a human is resolving the
// Approval that call produced. By the time the race starts, the retry's
// own idempotency reservation is already terminal (APPROVAL_REQUIRED, set
// by the original synchronous call before this test spawns any goroutine)
// — so the retry's own outcome is deterministic by design, not a coin
// flip; what genuinely races is that deterministic replay happening
// concurrently, against real Postgres, while ResolveApproval's own
// transaction (SELECT ... FOR UPDATE, UPDATE, commit) is in flight against
// a different row. The invariant under test: that overlap must never
// create a second PendingCommand/Approval for the same logical cancel
// (docs/audit/gap-approval-flow-durability.md's row 4) and must never
// perturb the resolution itself.
func TestIntegration_CancelWorkflow_ApprovalResolutionRacesCommandRetry(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	workflowID, version := startedReadyWorkflow(t, app)
	idemKey := uuid.New().String()

	pending := app.CancelWorkflow(ctx, CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: idemKey,
		WorkflowID: workflowID, ExpectedVersion: version, RequestingPrincipalID: app.Fixtures.TriggerPrincipal().PrincipalID,
	})
	if pending.Outcome != ApprovalRequired || pending.ApprovalID == nil {
		t.Fatalf("setup: CancelWorkflow outcome = %s (reasons: %v), want APPROVAL_REQUIRED", pending.Outcome, pending.Reasons)
	}

	var wg sync.WaitGroup
	var resolveRes, retryRes Result
	wg.Add(2)
	go func() {
		defer wg.Done()
		resolveRes = app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *pending.ApprovalID, Approve: true, DecidingPrincipal: app.Fixtures.ApproverPrincipal()})
	}()
	go func() {
		defer wg.Done()
		retryRes = app.CancelWorkflow(ctx, CancelWorkflowRequest{
			RequestID: uuid.New(), IdempotencyKey: idemKey,
			WorkflowID: workflowID, ExpectedVersion: version, RequestingPrincipalID: app.Fixtures.TriggerPrincipal().PrincipalID,
		})
	}()
	wg.Wait()

	if resolveRes.Outcome != Accepted {
		t.Fatalf("ResolveApproval outcome = %s (reasons: %v), want ACCEPTED", resolveRes.Outcome, resolveRes.Reasons)
	}
	if retryRes.Outcome != ApprovalRequired {
		t.Fatalf("retried CancelWorkflow (same idempotency key) outcome = %s (reasons: %v), want the original APPROVAL_REQUIRED replayed", retryRes.Outcome, retryRes.Reasons)
	}

	if got := countApprovalsForOrg(t, app.Fixtures.Organization().OrganizationID); got != 1 {
		t.Fatalf("approvals rows for this org = %d, want exactly 1 (the retry must never fabricate a second PendingCommand/Approval)", got)
	}
}
