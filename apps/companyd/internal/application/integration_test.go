package application

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/persistence/supabase"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/execution"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// requireRealApp builds an Application against the real Postgres-backed
// repositories (internal/adapters/persistence/supabase), skipped unless
// DATABASE_URL is set — the fake_repo_test.go fakes prove pipeline
// sequencing in isolation; this proves the same Application code actually
// works wired to the real schema supabase/migrations/ applies.
func requireRealApp(t *testing.T) *Application {
	t.Helper()
	_ = godotenv.Load("../../.env")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping application integration test")
	}
	pool, err := supabase.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	return &Application{
		Repo:     supabase.NewWorkflowRepository(pool),
		Pending:  supabase.NewPendingCommandRepository(pool),
		Exec:     supabase.NewExecutionRepository(pool),
		Fixtures: fixtures.NewRegistry(),
		Notify:   make(chan uuid.UUID, 4),
	}
}

// submitFakeResult stands in for Runtime: claims the due ExecutionIntent
// StartWorkflow committed (exactly what Runtime's Sweep does), saves a
// Result bound to that real claimed attempt, and submits it — proving
// SubmitResult's proposal/Governance/Kernel/commit pipeline against the
// real DB without making a live provider call.
func submitFakeResult(t *testing.T, app *Application, outcome result.Outcome) Result {
	t.Helper()
	ctx := context.Background()

	// Poll rather than assert on the first call: due_at is set from this
	// process's clock but compared against the DB server's now() — real
	// Runtime never assumes synchronous availability either, it polls
	// (internal/runtime.Runtime.Start), so a bounded retry here matches
	// production behavior instead of asserting an unrealistic guarantee.
	var claims []execution.ClaimedExecution
	var err error
	for attempt := 0; attempt < 30; attempt++ {
		claims, err = app.Exec.ClaimDueIntents(ctx, fixtures.OrganizationID, 10, time.Minute, "integration-test-worker")
		if err != nil {
			t.Fatalf("ClaimDueIntents: %v", err)
		}
		if len(claims) > 0 {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if len(claims) != 1 {
		t.Fatalf("claimed %d intents after polling, want exactly 1", len(claims))
	}
	claim := claims[0]

	now := time.Now().UTC()
	res := &result.Result{
		ResultID:             uuid.New(),
		OrganizationID:       claim.Intent.OrganizationID,
		ResultType:           "INTELLIGENCE_TEXT_GENERATION",
		WorkflowID:           claim.Intent.WorkflowID,
		ObjectiveID:          app.Fixtures.Objective().ObjectiveID,
		IntentID:             claim.Intent.IntentID,
		AttemptID:            claim.Attempt.AttemptID,
		CapabilityRequestID:  claim.Attempt.CapabilityRequestID,
		IdempotencyKey:       claim.Intent.IdempotencyKey + ":integration-test",
		ProducingPrincipalID: app.Fixtures.TriggerPrincipal().PrincipalID,
		ProviderAdapter:      "integration-test",
		ModelID:              "integration-test",
		Outcome:              outcome,
		Output:                map[string]any{"text": "integration test output"},
		StartedAt:             now,
		ObservedAt:            now,
		ReportedAt:            now,
	}
	if err := app.Exec.SaveResult(ctx, res); err != nil {
		t.Fatalf("SaveResult: %v", err)
	}

	return app.SubmitResult(ctx, SubmitResultRequest{
		RequestID:       uuid.New(),
		IdempotencyKey:  uuid.New().String(),
		ResultID:        res.ResultID,
		ExpectedVersion: claim.Intent.WorkflowVersion,
	})
}

func TestIntegration_CreateStartAccept_FullPipelineToCompleted(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()

	created := app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String()})
	if created.Outcome != Accepted {
		t.Fatalf("CreateWorkflow outcome = %s (reasons: %v)", created.Outcome, created.Reasons)
	}
	if created.Workflow.State != "PLANNED" {
		t.Fatalf("state after create = %s, want PLANNED", created.Workflow.State)
	}
	workflowID := uuid.MustParse(created.Workflow.WorkflowID)

	started := app.StartWorkflow(ctx, StartWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: created.Workflow.Version,
	})
	if started.Outcome != Accepted {
		t.Fatalf("StartWorkflow outcome = %s (reasons: %v)", started.Outcome, started.Reasons)
	}
	if started.Workflow.State != "READY" {
		t.Fatalf("state after start = %s, want READY", started.Workflow.State)
	}

	accepted := submitFakeResult(t, app, result.OutcomeSucceeded)
	if accepted.Outcome != Accepted {
		t.Fatalf("SubmitResult outcome = %s (reasons: %v)", accepted.Outcome, accepted.Reasons)
	}
	if accepted.Workflow.State != "COMPLETED" {
		t.Fatalf("state after accept = %s, want COMPLETED", accepted.Workflow.State)
	}

	status, err := app.GetWorkflowStatus(ctx, workflowID)
	if err != nil {
		t.Fatalf("GetWorkflowStatus: %v", err)
	}
	if status.Workflow.State != "COMPLETED" || status.LatestResult == nil || status.LatestResult.Outcome != "SUCCEEDED" {
		t.Fatalf("status = %+v, want COMPLETED with a SUCCEEDED latestResult", status)
	}
}

func TestIntegration_CreateStartReject_FullPipelineToFailed(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()

	created := app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String()})
	if created.Outcome != Accepted {
		t.Fatalf("CreateWorkflow outcome = %s (reasons: %v)", created.Outcome, created.Reasons)
	}
	workflowID := uuid.MustParse(created.Workflow.WorkflowID)

	started := app.StartWorkflow(ctx, StartWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: created.Workflow.Version,
	})
	if started.Outcome != Accepted {
		t.Fatalf("StartWorkflow outcome = %s (reasons: %v)", started.Outcome, started.Reasons)
	}

	rejected := submitFakeResult(t, app, result.OutcomeFailed)
	if rejected.Outcome != Accepted {
		t.Fatalf("SubmitResult outcome = %s (reasons: %v)", rejected.Outcome, rejected.Reasons)
	}
	if rejected.Workflow.State != "FAILED" {
		t.Fatalf("state after reject = %s, want FAILED", rejected.Workflow.State)
	}
}

func TestIntegration_StartWorkflow_ConflictOnStaleVersion(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()

	created := app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String()})
	if created.Outcome != Accepted {
		t.Fatalf("CreateWorkflow outcome = %s (reasons: %v)", created.Outcome, created.Reasons)
	}
	workflowID := uuid.MustParse(created.Workflow.WorkflowID)

	staleVersion := created.Workflow.Version + 99
	res := app.StartWorkflow(ctx, StartWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: staleVersion,
	})
	if res.Outcome != Rejected {
		t.Fatalf("StartWorkflow with stale version outcome = %s, want REJECTED (Kernel proposal validation catches it before any write)", res.Outcome)
	}
}

func TestIntegration_CancelWorkflow_PlannedToCancelled(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()

	created := app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String()})
	if created.Outcome != Accepted {
		t.Fatalf("CreateWorkflow outcome = %s (reasons: %v)", created.Outcome, created.Reasons)
	}
	workflowID := uuid.MustParse(created.Workflow.WorkflowID)

	cancelled := app.CancelWorkflow(ctx, CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: created.Workflow.Version,
	})
	if cancelled.Outcome != Accepted {
		t.Fatalf("CancelWorkflow outcome = %s (reasons: %v)", cancelled.Outcome, cancelled.Reasons)
	}
	if cancelled.Workflow.State != "CANCELLED" {
		t.Fatalf("state after cancel = %s, want CANCELLED", cancelled.Workflow.State)
	}

	status, err := app.GetWorkflowStatus(ctx, workflowID)
	if err != nil {
		t.Fatalf("GetWorkflowStatus: %v", err)
	}
	if status.Workflow.State != "CANCELLED" || status.Workflow.TerminalReason == nil {
		t.Fatalf("status = %+v, want CANCELLED with a TerminalReason", status)
	}
}

func TestIntegration_CancelWorkflow_ReadyToCancelled_ClosesOutstandingIntent(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()

	created := app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String()})
	if created.Outcome != Accepted {
		t.Fatalf("CreateWorkflow outcome = %s (reasons: %v)", created.Outcome, created.Reasons)
	}
	workflowID := uuid.MustParse(created.Workflow.WorkflowID)

	started := app.StartWorkflow(ctx, StartWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: created.Workflow.Version,
	})
	if started.Outcome != Accepted {
		t.Fatalf("StartWorkflow outcome = %s (reasons: %v)", started.Outcome, started.Reasons)
	}

	cancelled := app.CancelWorkflow(ctx, CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: started.Workflow.Version,
	})
	if cancelled.Outcome != Accepted {
		t.Fatalf("CancelWorkflow outcome = %s (reasons: %v)", cancelled.Outcome, cancelled.Reasons)
	}
	if cancelled.Workflow.State != "CANCELLED" {
		t.Fatalf("state after cancel = %s, want CANCELLED", cancelled.Workflow.State)
	}

	// The outstanding ExecutionIntent START_WORKFLOW produced must now be
	// durably closed — Runtime's claim query (status = 'PENDING') must
	// never pick it up, per docs/domain/execution.md's invariants.
	claims, err := app.Exec.ClaimDueIntents(ctx, fixtures.OrganizationID, 10, time.Minute, "integration-test-worker")
	if err != nil {
		t.Fatalf("ClaimDueIntents: %v", err)
	}
	for _, c := range claims {
		if c.Intent.WorkflowID == workflowID {
			t.Fatalf("cancelled Workflow's ExecutionIntent %s was still claimable after CANCEL_WORKFLOW", c.Intent.IntentID)
		}
	}
}

// TestIntegration_CreateWorkflow_IdempotentReplayReturnsSameOutcome proves
// the idempotency-replay guard (application.md) against real persistence —
// docs/testing/strategy.md requires a state-transition correctness claim to
// hold against the real database, not only fake_repo_test.go's in-memory
// fakeRepo.
func TestIntegration_CreateWorkflow_IdempotentReplayReturnsSameOutcome(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	key := uuid.New().String()

	first := app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: key})
	if first.Outcome != Accepted {
		t.Fatalf("first call outcome = %s (reasons: %v)", first.Outcome, first.Reasons)
	}

	// The replay path (application.go's replay()) intentionally returns
	// only Outcome, not a re-fetched Workflow view, keeping the fast path
	// cheap — so idempotency is verified by confirming no second
	// transition was applied, not by comparing a Workflow field replay
	// never populates.
	second := app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: key})
	if second.Outcome != first.Outcome {
		t.Fatalf("replayed outcome = %s, want %s (same as first call)", second.Outcome, first.Outcome)
	}

	workflowID := uuid.MustParse(first.Workflow.WorkflowID)
	status, err := app.GetWorkflowStatus(ctx, workflowID)
	if err != nil {
		t.Fatalf("GetWorkflowStatus: %v", err)
	}
	if status.Workflow.Version != 1 {
		t.Fatalf("version after replay = %d, want 1 (idempotency must not create a second transition)", status.Workflow.Version)
	}
}

// TestIntegration_CancelWorkflow_AlreadyTerminalRejected proves cancelling
// an already-terminal Workflow is REJECTED against real persistence, per
// docs/testing/strategy.md's same real-database requirement.
func TestIntegration_CancelWorkflow_AlreadyTerminalRejected(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()

	created := app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String()})
	if created.Outcome != Accepted {
		t.Fatalf("CreateWorkflow outcome = %s (reasons: %v)", created.Outcome, created.Reasons)
	}
	workflowID := uuid.MustParse(created.Workflow.WorkflowID)

	first := app.CancelWorkflow(ctx, CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: created.Workflow.Version,
	})
	if first.Outcome != Accepted {
		t.Fatalf("setup: first cancel outcome = %s (reasons: %v)", first.Outcome, first.Reasons)
	}

	second := app.CancelWorkflow(ctx, CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: first.Workflow.Version,
	})
	if second.Outcome != Rejected {
		t.Fatalf("cancelling an already-CANCELLED workflow outcome = %s, want REJECTED", second.Outcome)
	}
}
