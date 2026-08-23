package application

import (
	"context"
	"sync"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/execution"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// fakeRepo is an in-memory ports.AuthoritativeStateRepository. Sanctioned
// under docs/testing/strategy.md for pipeline_test.go's concurrency- and
// ordering-specific tests only — it is never the sole evidence for a
// state-transition correctness claim; integration_test.go's real-database
// tests own that.
type fakeRepo struct {
	mu                  sync.Mutex
	workflows           map[string]*workflow.Workflow
	idempotency         map[string]string
	results             map[uuid.UUID]*result.Result
	governanceDecisions map[string]bool // keyed by governanceDecisionKey(orgID, action, resourceType, resourceID)
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		workflows:           map[string]*workflow.Workflow{},
		idempotency:         map[string]string{},
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
	defer f.mu.Unlock()
	w, ok := f.workflows[key(orgID, workflowID)]
	if !ok {
		return nil, ports.ErrNotFound
	}
	cp := *w
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

func (f *fakeRepo) IdempotencyLookup(_ context.Context, orgID uuid.UUID, k string) (bool, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.idempotency[orgID.String()+":"+k]
	return ok, v, nil
}

func (f *fakeRepo) IdempotencyStore(_ context.Context, orgID, _ uuid.UUID, k, outcome string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	ik := orgID.String() + ":" + k
	if _, ok := f.idempotency[ik]; !ok {
		f.idempotency[ik] = outcome
	}
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

func (fakePending) ResolveApproval(context.Context, uuid.UUID, uuid.UUID, bool, *string) (*command.PendingCommand, *approval.Approval, error) {
	return nil, nil, ports.ErrConflict
}

func (fakePending) ListPendingApprovals(context.Context, uuid.UUID) ([]approval.Approval, error) {
	return nil, nil
}
