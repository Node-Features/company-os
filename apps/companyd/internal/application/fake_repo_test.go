package application

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/execution"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// rendezvous blocks each of n Arrive callers until all n have arrived, then
// releases all of them simultaneously — a one-shot barrier. Used to force a
// specific goroutine interleaving deterministically in a concurrency test,
// rather than relying on real Go-scheduler timing (which only makes a race
// window *likely* to be hit, not certain — see loadGate's use in
// pipeline_test.go for why that distinction matters here).
type rendezvous struct {
	n       int32
	arrived int32
	release chan struct{}
}

func newRendezvous(n int) *rendezvous {
	return &rendezvous{n: int32(n), release: make(chan struct{})}
}

func (r *rendezvous) Arrive() {
	if atomic.AddInt32(&r.arrived, 1) >= r.n {
		close(r.release)
	}
	<-r.release
}

// fakeRepo is an in-memory ports.AuthoritativeStateRepository. Sanctioned
// under docs/testing/strategy.md for pipeline_test.go's concurrency- and
// ordering-specific tests only — it is never the sole evidence for a
// state-transition correctness claim; integration_test.go's real-database
// tests own that.
// idempotencyRecord mirrors the two columns workflow_repo.go's real
// IdempotencyReserve/IdempotencyFinalize actually need: the outcome (or
// ports.IdempotencyInProgress while a reservation is live) and when that
// value was set, for the same staleness-based reclaim the real
// implementation does.
type idempotencyRecord struct {
	outcome   string
	createdAt time.Time
}

type fakeRepo struct {
	mu                  sync.Mutex
	workflows           map[string]*workflow.Workflow
	idempotency         map[string]idempotencyRecord
	results             map[uuid.UUID]*result.Result
	governanceDecisions map[string]bool // keyed by governanceDecisionKey(orgID, action, resourceType, resourceID)

	// loadGate, when non-nil, synchronizes concurrent LoadWorkflow callers
	// via rendezvous — see pipeline_test.go's use of it. nil (the default
	// for every test but the one that sets it) makes LoadWorkflow behave
	// exactly as before.
	loadGate *rendezvous
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		workflows:           map[string]*workflow.Workflow{},
		idempotency:         map[string]idempotencyRecord{},
		results:             map[uuid.UUID]*result.Result{},
		governanceDecisions: map[string]bool{},
	}
}

func governanceDecisionKey(orgID uuid.UUID, action, resourceType, resourceID string) string {
	return orgID.String() + ":" + action + ":" + resourceType + ":" + resourceID
}

func key(org, id uuid.UUID) string { return org.String() + ":" + id.String() }

func (f *fakeRepo) LoadWorkflow(_ context.Context, orgID, workflowID uuid.UUID) (*workflow.Workflow, error) {
	f.mu.Lock()
	w, ok := f.workflows[key(orgID, workflowID)]
	var cp workflow.Workflow
	if ok {
		cp = *w
	}
	f.mu.Unlock()
	if !ok {
		return nil, ports.ErrNotFound
	}
	// loadGate.Arrive() (if armed) blocks here, AFTER the locked read but
	// BEFORE returning to the caller — so a racing caller cannot proceed to
	// CommitTransition until every gated LoadWorkflow call has completed its
	// own read. Arriving before the lock (an earlier version of this code)
	// only synchronized *starting* the load, not the load itself: goroutine
	// A could still lock, read, unlock, and race all the way through
	// Kernel-validate/Governance/CommitTransition before goroutine B ever
	// got scheduled to acquire the lock for its own read — reproduced via
	// debug tracing during development of this fix. Blocking the return
	// instead closes that gap for real.
	if f.loadGate != nil {
		f.loadGate.Arrive()
	}
	return &cp, nil
}

func (f *fakeRepo) CreateWorkflow(_ context.Context, w *workflow.Workflow, _ []event.DomainEvent, _ uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := key(w.OrganizationID, w.WorkflowID)
	if _, ok := f.workflows[k]; ok {
		return ports.ErrConflict
	}
	cp := *w
	f.workflows[k] = &cp
	return nil
}

func (f *fakeRepo) CommitTransition(_ context.Context, w *workflow.Workflow, expectedVersion int64, _ []event.DomainEvent, _ uuid.UUID, _ *workflow.ExecutionIntent, _ *uuid.UUID, _ *uuid.UUID, decideResult *ports.ResultDecision, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := key(w.OrganizationID, w.WorkflowID)
	cur, ok := f.workflows[k]
	if !ok || cur.Version != expectedVersion {
		return ports.ErrConflict
	}
	cp := *w
	f.workflows[k] = &cp
	if decideResult != nil {
		if r, ok := f.results[decideResult.ResultID]; ok {
			accepted := decideResult.Accepted
			r.Accepted = &accepted
		}
	}
	return nil
}

func (f *fakeRepo) SaveGovernanceDecision(_ context.Context, _ uuid.UUID, orgID uuid.UUID, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID,
	action, resourceType, resourceID string, _ string, _ string, _ string, _ string, _ string, _ *string, _ *string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.governanceDecisions[governanceDecisionKey(orgID, action, resourceType, resourceID)] = true
	return nil
}

func (f *fakeRepo) GovernanceDecisionExists(_ context.Context, orgID uuid.UUID, action, resourceType, resourceID string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.governanceDecisions[governanceDecisionKey(orgID, action, resourceType, resourceID)], nil
}

// IdempotencyReserve mirrors workflow_repo.go's real atomic upsert: a
// missing key or a stale (older than ttl) IN_PROGRESS reservation is won
// by this caller; anything else returns won=false with the existing
// outcome (which may itself be ports.IdempotencyInProgress, meaning
// genuinely still in flight). f.mu makes the whole read-decide-write
// sequence atomic with respect to every other fakeRepo caller, matching
// what the real single-round-trip SQL statement guarantees against real
// concurrent Postgres sessions.
func (f *fakeRepo) IdempotencyReserve(_ context.Context, orgID, _ uuid.UUID, k string, ttl time.Duration) (bool, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ik := orgID.String() + ":" + k
	rec, ok := f.idempotency[ik]
	if !ok || (rec.outcome == ports.IdempotencyInProgress && time.Since(rec.createdAt) >= ttl) {
		f.idempotency[ik] = idempotencyRecord{outcome: ports.IdempotencyInProgress, createdAt: time.Now()}
		return true, "", nil
	}
	return false, rec.outcome, nil
}

func (f *fakeRepo) IdempotencyFinalize(_ context.Context, orgID uuid.UUID, k, outcome string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	ik := orgID.String() + ":" + k
	f.idempotency[ik] = idempotencyRecord{outcome: outcome, createdAt: f.idempotency[ik].createdAt}
	return nil
}

// fakeExec is a minimal in-memory ports.ExecutionRepository — this
// package's tests exercise the state-changing pipeline, not Runtime
// mechanics, so most methods are unused no-ops.
type fakeExec struct {
	mu      sync.Mutex
	results map[uuid.UUID]*result.Result
}

func newFakeExec() *fakeExec { return &fakeExec{results: map[uuid.UUID]*result.Result{}} }

func (f *fakeExec) ClaimDueIntents(context.Context, uuid.UUID, int, time.Duration, string) ([]execution.ClaimedExecution, error) {
	return nil, nil
}
func (f *fakeExec) RecordDispatched(context.Context, uuid.UUID, int64, string) error { return nil }
func (f *fakeExec) RecordTerminal(context.Context, uuid.UUID, int64, execution.AttemptStatus, *uuid.UUID) error {
	return nil
}
func (f *fakeExec) ScheduleRetry(context.Context, uuid.UUID, uuid.UUID, time.Time) error { return nil }
func (f *fakeExec) ReclaimExpiredLeases(context.Context, uuid.UUID, int) ([]execution.ClaimedExecution, error) {
	return nil, nil
}
func (f *fakeExec) SaveResult(_ context.Context, r *result.Result) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.results[r.ResultID] = r
	return nil
}
func (f *fakeExec) GetResult(_ context.Context, _ uuid.UUID, resultID uuid.UUID) (*result.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.results[resultID]
	if !ok {
		return nil, ports.ErrNotFound
	}
	return r, nil
}
func (f *fakeExec) GetLatestResult(context.Context, uuid.UUID, uuid.UUID) (*result.Result, error) {
	return nil, ports.ErrNotFound
}
func (f *fakeExec) ListExecutionUnits(context.Context, uuid.UUID, uuid.UUID) ([]execution.ExecutionUnit, error) {
	return nil, nil
}

// fakePending is a no-op ports.PendingCommandRepository — pipeline_test.go's
// concurrency/ordering tests never reach REQUIRE_APPROVAL; the full
// resolve/resume round trip is covered against the real database in
// integration_test.go instead (docs/testing/strategy.md's fake-repository
// decision — state-transition correctness belongs to the real-DB suite).
type fakePending struct{}

func (fakePending) CreatePendingApproval(context.Context, *command.PendingCommand, uuid.UUID, *approval.Approval) error {
	return nil
}

func (fakePending) ResolveApproval(context.Context, uuid.UUID, principal.Principal, bool, *string) (*command.PendingCommand, *approval.Approval, error) {
	return nil, nil, ports.ErrConflict
}
