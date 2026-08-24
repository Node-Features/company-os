package application

import (
	"context"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/Node-Features/company-os/apps/companyd/internal/observability"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// idempotencyReservationTTL bounds how long a reservation may sit
// IN_PROGRESS before another request racing the same key is allowed to
// treat it as abandoned (the original request crashed between winning the
// reservation and calling finalize) and reclaim it. Long enough to cover a
// real request's governance+persistence round trips; short enough that a
// genuine crash doesn't wedge a key for long.
const idempotencyReservationTTL = 30 * time.Second

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
	// Knowledge persists KnowledgeItem versions (ROADMAP.md Phase 5 Slice 1,
	// docs/architecture/knowledge.md's ingestion/versioning flow).
	Knowledge ports.KnowledgeRepository
	// Metrics is an optional operational-metrics sink
	// (docs/architecture/observability.md). A nil Metrics is never an
	// error — every emission call site checks it first.
	Metrics ports.MetricsRecorder
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

// reserveOrReplay implements application.md's idempotent-replay guard:
// retrying the same logical request must reuse its idempotency identity
// and cannot create a second legal transition — including when two
// requests for the same key race each other concurrently. It atomically
// reserves the key (see ports.AuthoritativeStateRepository.IdempotencyReserve
// for why this must be one round trip, not a lookup-then-write). A true
// (Result, true) means the caller must return Result immediately without
// running the use case, whether that Result is a real replay of a terminal
// outcome or Indeterminate (another request is still in flight — the
// caller is expected to retry with the same key). A false ok means this
// call won the reservation and must eventually call finalize. Takes orgID/
// requestID/key directly rather than the use case's command.WorkflowCommandEnvelope
// since every call site invokes this before that envelope is built (it
// needs Workflow state this guard must run ahead of loading).
func (a *Application) reserveOrReplay(ctx context.Context, orgID, requestID uuid.UUID, key string) (Result, bool) {
	won, existingOutcome, err := a.Repo.IdempotencyReserve(ctx, orgID, requestID, key, idempotencyReservationTTL)
	if err != nil {
		// Fail open to "not reserved, proceed" — matches this guard's
		// pre-existing posture on a transient lookup error: a false
		// negative here means unnecessary re-execution, never a lost
		// legal-transition guarantee, since every domain write downstream
		// still goes through its own CAS.
		return Result{}, false
	}
	if won {
		return Result{}, false
	}
	if existingOutcome == ports.IdempotencyInProgress {
		return Result{Outcome: Indeterminate, Reasons: []string{"request_in_progress_retry_with_same_idempotency_key"}}, true
	}
	return Result{Outcome: Outcome(existingOutcome)}, true
}

// finalize overwrites the reservation reserveOrReplay won with the use
// case's real outcome. A finalize failure is logged, not swallowed — but
// the use case still returns res, its already-committed real outcome,
// since failing the response here would misrepresent state that already
// changed. A stuck IN_PROGRESS row left by this failure is only recovered
// after idempotencyReservationTTL, by whichever request next races this
// key (see IdempotencyReserve) — a narrowed, not eliminated, crash window,
// consistent with this codebase's other documented residual gaps.
func (a *Application) finalize(ctx context.Context, cmd command.WorkflowCommandEnvelope, res Result) Result {
	enriched := observability.WithExecutionContext(ctx, observability.ExecutionContext{
		CorrelationID:  cmd.CorrelationID,
		CommandID:      cmd.CommandID,
		OrganizationID: cmd.OrganizationID,
		WorkflowID:     cmd.WorkflowID,
		PrincipalID:    cmd.RequestingPrincipalID,
	})
	if err := a.Repo.IdempotencyFinalize(ctx, cmd.OrganizationID, cmd.IdempotencyKey, string(res.Outcome)); err != nil {
		observability.Logger(enriched).Error("idempotency finalize failed (result already committed; ledger entry may be stale until reclaimed)",
			"idempotency_key", cmd.IdempotencyKey, "error", err.Error())
	}
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
