package application

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// Application is the orchestration boundary described in
// docs/architecture/application.md: it coordinates Kernel, Governance, and
// Persistence for every state-changing use case without owning their
// rules. Notify is the in-process Runtime wake-up hint (first-slice plan
// decision #8) — a non-blocking send, never required for correctness since
// Runtime's polling sweep is the durable fallback.
type Application struct {
	Repo     ports.AuthoritativeStateRepository
	Pending  ports.PendingCommandRepository
	Exec     ports.ExecutionRepository
	Fixtures fixtures.Registry
	Notify   chan<- uuid.UUID
	// Research persists the Research department's core contracts
	// (ROADMAP.md Phase 4 Slice 1, docs/workflows/research-loop.md).
	Research ports.ResearchRepository
	// MonitoringEvaluation persists M&E's core contracts (ROADMAP.md Phase
	// 4 Slice 2, docs/departments/monitoring-evaluation.md).
	MonitoringEvaluation ports.MonitoringEvaluationRepository
	// Finance persists Finance's core contracts (ROADMAP.md Phase 4 Slice
	// 3, docs/departments/finance.md).
	Finance ports.FinanceRepository
	// Objective persists proposed Objectives (ROADMAP.md Phase 4 Slice 4,
	// docs/architecture/departments.md's Objective creation gate).
	Objective ports.ObjectiveRepository
}

func viewOf(w *workflow.Workflow) *WorkflowView {
	return &WorkflowView{
		WorkflowID:     w.WorkflowID.String(),
		Version:        w.Version,
		State:          string(w.State),
		WaitReason:     w.WaitReason,
		TerminalReason: w.TerminalReason,
	}
}

// replay implements application.md's idempotent-replay guard: retrying the
// same logical request must reuse its idempotency identity and cannot
// create a second legal transition.
func (a *Application) replay(ctx context.Context, orgID uuid.UUID, key string) (Result, bool) {
	found, outcome, err := a.Repo.IdempotencyLookup(ctx, orgID, key)
	if err != nil || !found {
		return Result{}, false
	}
	return Result{Outcome: Outcome(outcome)}, true
}

func (a *Application) store(ctx context.Context, cmd command.WorkflowCommandEnvelope, res Result) Result {
	_ = a.Repo.IdempotencyStore(ctx, cmd.OrganizationID, cmd.RequestID, cmd.IdempotencyKey, string(res.Outcome))
	return res
}

// notify sends a non-blocking wake-up hint; a full channel or nil Notify
// is never an error — the polling sweep remains the durable path.
func (a *Application) notify(intentID uuid.UUID) {
	if a.Notify == nil {
		return
	}
	select {
	case a.Notify <- intentID:
	default:
	}
}
