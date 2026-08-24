package application

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/persistence/supabase"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/execution"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// requireRealApp builds an Application against the real Postgres-backed
// repositories (internal/adapters/persistence/supabase), skipped unless
// DATABASE_URL is set — the fake_repo_test.go fakes prove pipeline
// sequencing in isolation; this proves the same Application code actually
// works wired to the real schema supabase/migrations/ applies.
//
// Fixtures use a fresh organization ID per call (fixtures.
// NewRegistryWithOrganization), not the shared fixtures.OrganizationID
// constant — ClaimDueIntents claims due ExecutionIntents organization-wide,
// so two Applications sharing one org (whether from two tests in this
// package, two packages, or two overlapping `go test` invocations against
// this project's persistent dev database) can each claim intents the other
// created, breaking any test that polls for "claimed exactly 1 intent".
// Reproduced directly: running this package's suite concurrently with
// itself against the shared constant reliably produced exactly that failure
// ("claimed 2 intents after polling, want exactly 1"). A fresh org per test
// makes that structurally impossible rather than merely unlikely — matching
// the isolation internal/adapters/persistence/supabase's own tests already
// use (see its testOrgID doc comment) for the same reason.
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
		Repo:                 supabase.NewWorkflowRepository(pool),
		Pending:              supabase.NewPendingCommandRepository(pool),
		Exec:                 supabase.NewExecutionRepository(pool),
		Fixtures:             fixtures.NewRegistryWithOrganization(uuid.New()),
		Notify:               make(chan uuid.UUID, 4),
		Research:             supabase.NewResearchRepository(pool),
		MonitoringEvaluation: supabase.NewMonitoringEvaluationRepository(pool),
		Finance:              supabase.NewFinanceRepository(pool),
		Objective:            supabase.NewObjectiveRepository(pool),
		Knowledge:            supabase.NewKnowledgeRepository(pool),
	}
}

// submitFakeResult stands in for Runtime: claims the due ExecutionIntent
// StartWorkflow committed (exactly what Runtime's Sweep does), saves a
// Result bound to that real claimed attempt, and submits it — proving
// SubmitResult's proposal/Governance/Kernel/commit pipeline against the
// real DB without making a live provider call. Returns the real Result's
// ID too — Phase 4 Slice 2's M&E tests need it to drive RecordMetric
// against a real, persisted Result row.
func submitFakeResult(t *testing.T, app *Application, outcome result.Outcome) (Result, uuid.UUID) {
	t.Helper()
	return submitFakeResultWithProvider(t, app, outcome, "integration-test", "integration-test", 0, 0)
}

// submitFakeResultWithProvider is submitFakeResult generalized with a
// caller-chosen ProviderAdapter/ModelID/token usage — Finance's
// RecordResourceUsage (ROADMAP.md Phase 4 Slice 3) needs a Result whose
// provider matches a seeded PriceProfile and whose Output carries real
// token counts, which the fixed "integration-test" provider submitFakeResult
// uses doesn't have.
func submitFakeResultWithProvider(t *testing.T, app *Application, outcome result.Outcome, provider, modelID string, inputTokens, outputTokens int) (Result, uuid.UUID) {
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
		claims, err = app.Exec.ClaimDueIntents(ctx, app.Fixtures.Organization().OrganizationID, 10, time.Minute, "integration-test-worker")
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
		ProviderAdapter:      provider,
		ModelID:              modelID,
		Outcome:              outcome,
		Output: map[string]any{
			"text":         "integration test output",
			"inputTokens":  inputTokens,
			"outputTokens": outputTokens,
		},
		StartedAt:  now,
		ObservedAt: now,
		ReportedAt: now,
	}
	if err := app.Exec.SaveResult(ctx, res); err != nil {
		t.Fatalf("SaveResult: %v", err)
	}

	return app.SubmitResult(ctx, SubmitResultRequest{
		RequestID:       uuid.New(),
		IdempotencyKey:  uuid.New().String(),
		ResultID:        res.ResultID,
		ExpectedVersion: claim.Intent.WorkflowVersion,
	}), res.ResultID
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

	accepted, _ := submitFakeResult(t, app, result.OutcomeSucceeded)
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

// TestIntegration_CreateWorkflow_PersistsGovernanceDecision closes the gap
// this session's DB inspection found: evaluateGovernance's ALLOW branch
// used to return without ever calling SaveGovernanceDecision, so a plain
// CREATE_WORKFLOW's ALLOW decision was never written to governance_decisions
// at all. governance.md: "Policy, authority, approval, and decision records
// are persisted before dependent execution continues" — this proves it
// against the real database for CreateWorkflow specifically, not inferred
// from a different command that happens to share the same code path.
func TestIntegration_CreateWorkflow_PersistsGovernanceDecision(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()

	created := app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String()})
	if created.Outcome != Accepted {
		t.Fatalf("CreateWorkflow outcome = %s (reasons: %v)", created.Outcome, created.Reasons)
	}

	// CreateWorkflow's Result deliberately doesn't expose the governance
	// decision ID — that's internal audit plumbing, not public API — so
	// GovernanceDecisionExists looks the row up by the same (org, action,
	// resource) identity kernelwf.newProposal assigned this exact command:
	// action "workflow.create", resource type "Workflow", resource ID the
	// new Workflow's own ID.
	exists, err := app.Repo.GovernanceDecisionExists(ctx, app.Fixtures.Organization().OrganizationID, "workflow.create", "Workflow", created.Workflow.WorkflowID)
	if err != nil {
		t.Fatalf("GovernanceDecisionExists: %v", err)
	}
	if !exists {
		t.Fatalf("CreateWorkflow's ALLOW decision for Workflow %s was never persisted to governance_decisions", created.Workflow.WorkflowID)
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

	rejected, _ := submitFakeResult(t, app, result.OutcomeFailed)
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

// TestIntegration_CancelWorkflow_ReadyToCancelled_ClosesOutstandingIntent
// now goes through Phase 3 Slice 2's approval round trip — cancelling a
// READY Workflow no longer completes immediately (that's exactly
// cancelAutonomyRequirement's point) — but the same durable-closure
// invariant must still hold once the human approves and resumption
// commits.
func TestIntegration_CancelWorkflow_ReadyToCancelled_ClosesOutstandingIntent(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	workflowID, version := startedReadyWorkflow(t, app)

	pending := app.CancelWorkflow(ctx, CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: version,
	})
	if pending.Outcome != ApprovalRequired || pending.ApprovalID == nil {
		t.Fatalf("CancelWorkflow outcome = %s (reasons: %v), want APPROVAL_REQUIRED with an ApprovalID", pending.Outcome, pending.Reasons)
	}

	cancelled := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *pending.ApprovalID, Approve: true})
	if cancelled.Outcome != Accepted {
		t.Fatalf("ResolveApproval(approve=true) outcome = %s (reasons: %v)", cancelled.Outcome, cancelled.Reasons)
	}
	if cancelled.Workflow.State != "CANCELLED" {
		t.Fatalf("state after cancel = %s, want CANCELLED", cancelled.Workflow.State)
	}

	// The outstanding ExecutionIntent START_WORKFLOW produced must now be
	// durably closed — Runtime's claim query (status = 'PENDING') must
	// never pick it up, per docs/domain/execution.md's invariants.
	claims, err := app.Exec.ClaimDueIntents(ctx, app.Fixtures.Organization().OrganizationID, 10, time.Minute, "integration-test-worker")
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

// TestIntegration_CancelWorkflow_NonInitiatorDenied proves ROADMAP.md Phase
// 3 Slice 1's governed DENY path end-to-end against the real database and
// the real HTTP-reachable Application.CancelWorkflow entry point: a
// Workflow whose InitiatingPrincipalID differs from the fixture's trigger
// Principal (seeded directly through the real ports.AuthoritativeStateRepository.CreateWorkflow,
// the same bypass-the-use-case seeding pattern submitFakeResult already
// uses for Results — no HTTP endpoint lets a caller assert a different
// Principal yet, since that's Phase 3 Slice 3/4's job) is DENIED, not
// executed: the Workflow must remain PLANNED at its original version.
func TestIntegration_CancelWorkflow_NonInitiatorDenied(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	reg := app.Fixtures

	w := &workflow.Workflow{
		OrganizationID:        reg.Organization().OrganizationID,
		WorkflowID:            uuid.New(),
		Version:               1,
		DefinitionID:          reg.WorkflowDefinition().DefinitionID,
		DefinitionVersion:     reg.WorkflowDefinition().Version,
		ObjectiveID:           reg.Objective().ObjectiveID,
		State:                 workflow.StatePlanned,
		InitiatingPrincipalID: uuid.New(), // deliberately not the fixture trigger Principal
		CorrelationID:         uuid.New(),
		Inputs:                map[string]any{"prompt": "owned by someone else"},
		CreatedAt:             time.Now().UTC(),
		UpdatedAt:             time.Now().UTC(),
	}
	if err := app.Repo.CreateWorkflow(ctx, w, nil, uuid.New()); err != nil {
		t.Fatalf("seed CreateWorkflow: %v", err)
	}

	denied := app.CancelWorkflow(ctx, CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: w.WorkflowID, ExpectedVersion: w.Version,
	})
	if denied.Outcome != Denied {
		t.Fatalf("CancelWorkflow by a non-initiating Principal outcome = %s (reasons: %v), want DENIED", denied.Outcome, denied.Reasons)
	}

	status, err := app.GetWorkflowStatus(ctx, w.WorkflowID)
	if err != nil {
		t.Fatalf("GetWorkflowStatus: %v", err)
	}
	if status.Workflow.State != "PLANNED" || status.Workflow.Version != 1 {
		t.Fatalf("status = %+v, want unchanged PLANNED/1 — a DENIED decision must never execute", status)
	}
}

// startedReadyWorkflow creates a Workflow and starts it to READY — the
// shared setup for every Phase 3 Slice 2 REQUIRE_APPROVAL test below,
// since cancelAutonomyRequirement only escalates a READY cancel. Some
// callers (e.g. TestIntegration_CancelWorkflow_ReadyRequiresApproval,
// TestIntegration_ResolveApproval_RejectedLeavesWorkflowReady) deliberately
// leave the Workflow READY with its ExecutionIntent still PENDING — that's
// what they're asserting. Left alone, that PENDING intent would sit
// claimable forever: ClaimDueIntents is organization-wide, not scoped to
// one workflow, so an unrelated test's submitFakeResult could sweep it up
// later and see more claims than it expects (observed: 4 claimed instead
// of 1). t.Cleanup sweeps it away regardless of what the test itself did —
// claiming an already-CLAIMED/CLOSED intent from a test that DID resolve
// its approval is a harmless no-op.
func startedReadyWorkflow(t *testing.T, app *Application) (workflowID uuid.UUID, version int64) {
	t.Helper()
	ctx := context.Background()

	created := app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String()})
	if created.Outcome != Accepted {
		t.Fatalf("setup CreateWorkflow outcome = %s (reasons: %v)", created.Outcome, created.Reasons)
	}
	workflowID = uuid.MustParse(created.Workflow.WorkflowID)

	started := app.StartWorkflow(ctx, StartWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: created.Workflow.Version,
	})
	if started.Outcome != Accepted {
		t.Fatalf("setup StartWorkflow outcome = %s (reasons: %v)", started.Outcome, started.Reasons)
	}
	t.Cleanup(func() {
		_, _ = app.Exec.ClaimDueIntents(context.Background(), app.Fixtures.Organization().OrganizationID, 10, time.Minute, "integration-test-cleanup")
	})
	return workflowID, started.Workflow.Version
}

// TestIntegration_CancelWorkflow_ReadyRequiresApproval proves ROADMAP.md
// Phase 3 Slice 2's concrete REQUIRE_APPROVAL trigger against the real
// database: cancelling an already-READY (dispatched) Workflow does not
// execute immediately — it returns APPROVAL_REQUIRED with an ApprovalID,
// and the Workflow must remain untouched (still READY, same version).
func TestIntegration_CancelWorkflow_ReadyRequiresApproval(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	workflowID, version := startedReadyWorkflow(t, app)

	res := app.CancelWorkflow(ctx, CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: version,
	})
	if res.Outcome != ApprovalRequired {
		t.Fatalf("CancelWorkflow on a READY Workflow outcome = %s (reasons: %v), want APPROVAL_REQUIRED", res.Outcome, res.Reasons)
	}
	if res.ApprovalID == nil {
		t.Fatal("expected a non-nil ApprovalID on APPROVAL_REQUIRED")
	}

	status, err := app.GetWorkflowStatus(ctx, workflowID)
	if err != nil {
		t.Fatalf("GetWorkflowStatus: %v", err)
	}
	if status.Workflow.State != "READY" || status.Workflow.Version != version {
		t.Fatalf("status = %+v, want unchanged READY/%d — APPROVAL_REQUIRED must never execute", status, version)
	}
}

// TestIntegration_ResolveApproval_ApprovedResumesAndCancels proves the full
// round trip: approving the pending Approval immediately completes the
// original CANCEL_WORKFLOW (docs/domain/approval.md's resolution steps
// 3-7) — Workflow reaches CANCELLED, and a second resolution attempt on
// the same (now-consumed) Approval is rejected as a conflict, proving no
// double execution.
func TestIntegration_ResolveApproval_ApprovedResumesAndCancels(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	workflowID, version := startedReadyWorkflow(t, app)

	pending := app.CancelWorkflow(ctx, CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: version,
	})
	if pending.Outcome != ApprovalRequired || pending.ApprovalID == nil {
		t.Fatalf("setup: CancelWorkflow outcome = %s (reasons: %v), want APPROVAL_REQUIRED with an ApprovalID", pending.Outcome, pending.Reasons)
	}

	resolved := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *pending.ApprovalID, Approve: true})
	if resolved.Outcome != Accepted {
		t.Fatalf("ResolveApproval(approve=true) outcome = %s (reasons: %v), want ACCEPTED", resolved.Outcome, resolved.Reasons)
	}
	if resolved.Workflow == nil || resolved.Workflow.State != "CANCELLED" {
		t.Fatalf("resolved.Workflow = %+v, want state CANCELLED", resolved.Workflow)
	}

	status, err := app.GetWorkflowStatus(ctx, workflowID)
	if err != nil {
		t.Fatalf("GetWorkflowStatus: %v", err)
	}
	if status.Workflow.State != "CANCELLED" {
		t.Fatalf("final state = %s, want CANCELLED", status.Workflow.State)
	}

	again := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *pending.ApprovalID, Approve: true})
	if again.Outcome != Conflict {
		t.Fatalf("resolving an already-consumed Approval again outcome = %s, want CONFLICT (no double execution)", again.Outcome)
	}
}

// TestIntegration_ResolveApproval_RejectedLeavesWorkflowReady proves
// rejecting the Approval does not touch the Workflow at all — it stays
// READY, unlike approval which resumes and cancels it.
func TestIntegration_ResolveApproval_RejectedLeavesWorkflowReady(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	workflowID, version := startedReadyWorkflow(t, app)

	pending := app.CancelWorkflow(ctx, CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: version,
	})
	if pending.Outcome != ApprovalRequired || pending.ApprovalID == nil {
		t.Fatalf("setup: CancelWorkflow outcome = %s (reasons: %v), want APPROVAL_REQUIRED with an ApprovalID", pending.Outcome, pending.Reasons)
	}

	reason := "not today"
	resolved := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *pending.ApprovalID, Approve: false, Reason: &reason})
	if resolved.Outcome != Rejected {
		t.Fatalf("ResolveApproval(approve=false) outcome = %s (reasons: %v), want REJECTED", resolved.Outcome, resolved.Reasons)
	}

	status, err := app.GetWorkflowStatus(ctx, workflowID)
	if err != nil {
		t.Fatalf("GetWorkflowStatus: %v", err)
	}
	if status.Workflow.State != "READY" || status.Workflow.Version != version {
		t.Fatalf("status = %+v, want unchanged READY/%d — a rejected Approval must never execute", status, version)
	}
}

// TestIntegration_ResolveApproval_UnknownApprovalConflict proves an
// unresolvable ApprovalID (never existed, or already decided) is a
// CONFLICT, not a crash or a silent no-op success.
func TestIntegration_ResolveApproval_UnknownApprovalConflict(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()

	res := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: uuid.New(), Approve: true})
	if res.Outcome != Conflict {
		t.Fatalf("ResolveApproval on an unknown ApprovalID outcome = %s, want CONFLICT", res.Outcome)
	}
}

// TestIntegration_CancelWorkflow_PlannedStaysAutomatic is a regression
// check: cancelAutonomyRequirement only escalates a READY cancel, so a
// PLANNED Workflow's cancel must still complete immediately, exactly as it
// did before this slice (ROADMAP.md Phase 3 Slice 1's behavior).
func TestIntegration_CancelWorkflow_PlannedStaysAutomatic(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()

	created := app.CreateWorkflow(ctx, CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String()})
	if created.Outcome != Accepted {
		t.Fatalf("CreateWorkflow outcome = %s (reasons: %v)", created.Outcome, created.Reasons)
	}
	workflowID := uuid.MustParse(created.Workflow.WorkflowID)

	res := app.CancelWorkflow(ctx, CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: created.Workflow.Version,
	})
	if res.Outcome != Accepted {
		t.Fatalf("CancelWorkflow on a PLANNED Workflow outcome = %s (reasons: %v), want ACCEPTED (no approval needed)", res.Outcome, res.Reasons)
	}
	if res.Workflow.State != "CANCELLED" {
		t.Fatalf("state = %s, want CANCELLED", res.Workflow.State)
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

// TestIntegration_AuthorizeDispatch_StaleWorkflowVersionDenied proves
// ROADMAP.md Phase 3 Slice 5's dispatch-time re-evaluation (governance.md
// step 8: "Re-evaluate policy, authority, context, and approval at dispatch
// time") against the real database: internal/runtime.Runtime.execute calls
// Application.AuthorizeDispatch immediately before every dispatch, passing
// the ExecutionIntent it already claimed. If the Workflow's persisted
// version has since moved past what the intent was claimed against — the
// mechanism already implemented in workflow_start.go's AuthorizeDispatch —
// the exact same governed action that was ALLOWed at START_WORKFLOW time
// must now be DENIED rather than dispatched, per the invariant "no governed
// action reaches an executor without a current persisted ALLOW decision."
func TestIntegration_AuthorizeDispatch_StaleWorkflowVersionDenied(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	workflowID, version := startedReadyWorkflow(t, app)

	// A hand-built stale intent, not a real claimed row: AuthorizeDispatch's
	// contract only depends on WorkflowVersion mismatching the Workflow's
	// current persisted version, so this exercises the exact same
	// comparison Runtime's real claimed intent would hit after a
	// concurrent modification, without needing to manufacture that race
	// through the DB.
	staleIntent := workflow.ExecutionIntent{
		IntentID:        uuid.New(),
		OrganizationID:  app.Fixtures.Organization().OrganizationID,
		WorkflowID:      workflowID,
		WorkflowVersion: version - 1,
	}

	decision, err := app.AuthorizeDispatch(ctx, staleIntent)
	if err != nil {
		t.Fatalf("AuthorizeDispatch: %v", err)
	}
	if decision.Outcome != policy.DecisionDenied {
		t.Fatalf("AuthorizeDispatch on a stale intent version outcome = %s, want DENY", decision.Outcome)
	}
	if decision.Reason == nil || *decision.Reason != "stale intent version" {
		t.Fatalf("decision.Reason = %v, want \"stale intent version\"", decision.Reason)
	}
}

// TestIntegration_AuthorizeDispatch_CurrentVersionAllowed is the companion
// happy-path proof for the same Slice 5 mechanism: a real claimed intent,
// still matching the Workflow's current version, must be re-authorized with
// a fresh ALLOW — the re-evaluation is real, not a rubber stamp that always
// denies.
func TestIntegration_AuthorizeDispatch_CurrentVersionAllowed(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	workflowID, _ := startedReadyWorkflow(t, app)

	var claim *execution.ClaimedExecution
	for attempt := 0; attempt < 30; attempt++ {
		claims, err := app.Exec.ClaimDueIntents(ctx, app.Fixtures.Organization().OrganizationID, 10, time.Minute, "integration-test-worker")
		if err != nil {
			t.Fatalf("ClaimDueIntents: %v", err)
		}
		for i := range claims {
			if claims[i].Intent.WorkflowID == workflowID {
				claim = &claims[i]
				break
			}
		}
		if claim != nil {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if claim == nil {
		t.Fatal("did not claim this test's ExecutionIntent after polling")
	}

	decision, err := app.AuthorizeDispatch(ctx, claim.Intent)
	if err != nil {
		t.Fatalf("AuthorizeDispatch: %v", err)
	}
	if decision.Outcome != policy.DecisionAutomatic {
		t.Fatalf("AuthorizeDispatch on a current-version intent outcome = %s (reason: %v), want ALLOW", decision.Outcome, decision.Reason)
	}
}
