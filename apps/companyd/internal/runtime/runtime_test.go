package runtime

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/persistence/supabase"
	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/execution"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// requireRealRuntime builds a Runtime and its Application against the real
// Postgres-backed repositories, skipped unless DATABASE_URL is set — same
// convention as internal/application/integration_test.go's requireRealApp,
// including the fresh-organization-per-test isolation
// (fixtures.NewRegistryWithOrganization) that fix confirmed is required to
// avoid cross-test/cross-process interference via ClaimDueIntents/
// ReclaimExpiredLeases, both organization-scoped-but-otherwise-unscoped
// claim queries. provider and notifier are fakes — the two genuinely
// external dependencies; everything else (Exec, App's Repo/Pending) is
// real, so every assertion about claim/lease/fencing/commit behavior is
// proven against the actual schema, not a model of it.
func requireRealRuntime(t *testing.T, provider *fakeProvider, notifier *fakeNotifier) (*Runtime, *application.Application) {
	t.Helper()
	_ = godotenv.Load("../../.env")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping runtime integration test")
	}
	pool, err := supabase.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	reg := fixtures.NewRegistryWithOrganization(uuid.New())
	app := &application.Application{
		Repo:     supabase.NewWorkflowRepository(pool),
		Pending:  supabase.NewPendingCommandRepository(pool),
		Exec:     supabase.NewExecutionRepository(pool),
		Fixtures: reg,
		Notify:   make(chan uuid.UUID, 8),
	}
	rt := &Runtime{
		Exec:          supabase.NewExecutionRepository(pool),
		App:           app,
		Provider:      provider,
		ProviderName:  "fake-provider",
		ModelID:       "fake-model",
		Fixtures:      reg,
		PollInterval:  time.Hour, // tests drive Sweep/reclaimAbandoned explicitly, never via the ticker
		LeaseDuration: time.Minute,
		Wakeup:        make(chan uuid.UUID, 8),
		Notifier:      notifier,
	}
	rt.ensureWorkContext()
	return rt, app
}

// startedReadyWorkflow drives a real Workflow from nothing to READY —
// CreateWorkflow then StartWorkflow — the same two-call sequence
// internal/application's own tests use, duplicated locally rather than
// exported cross-package since it's a handful of lines and package-private
// helpers aren't importable across package boundaries anyway. Returns the
// workflow ID and its version once READY (1, since Create starts at
// version... check: StartWorkflow bumps PLANNED(1)->READY(2)).
func startedReadyWorkflow(t *testing.T, app *application.Application) (workflowID uuid.UUID, version int64) {
	t.Helper()
	ctx := context.Background()

	created := app.CreateWorkflow(ctx, application.CreateWorkflowRequest{RequestID: uuid.New(), IdempotencyKey: uuid.New().String(), RequestingPrincipalID: app.Fixtures.TriggerPrincipal().PrincipalID})
	if created.Outcome != application.Accepted {
		t.Fatalf("setup CreateWorkflow outcome = %s (reasons: %v)", created.Outcome, created.Reasons)
	}
	workflowID = uuid.MustParse(created.Workflow.WorkflowID)

	started := app.StartWorkflow(ctx, application.StartWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: created.Workflow.Version, RequestingPrincipalID: app.Fixtures.TriggerPrincipal().PrincipalID,
	})
	if started.Outcome != application.Accepted {
		t.Fatalf("setup StartWorkflow outcome = %s (reasons: %v)", started.Outcome, started.Reasons)
	}
	return workflowID, started.Workflow.Version
}

func loadWorkflow(t *testing.T, app *application.Application, workflowID uuid.UUID) *workflow.Workflow {
	t.Helper()
	w, err := app.Repo.LoadWorkflow(context.Background(), app.Fixtures.Organization().OrganizationID, workflowID)
	if err != nil {
		t.Fatalf("LoadWorkflow: %v", err)
	}
	return w
}

// --- 1. Normal execution -----------------------------------------------

func TestIntegration_Runtime_NormalExecution_CompletesWorkflow(t *testing.T) {
	provider := &fakeProvider{result: ports.IntelligenceResult{Text: "hello", ModelID: "fake-model", Provider: "fake-provider"}}
	notifier := &fakeNotifier{}
	rt, app := requireRealRuntime(t, provider, notifier)
	workflowID, _ := startedReadyWorkflow(t, app)

	rt.Sweep(context.Background())
	rt.Wait()

	w := loadWorkflow(t, app, workflowID)
	if w.State != workflow.StateCompleted {
		t.Fatalf("workflow state = %s, want COMPLETED", w.State)
	}
	if got := provider.callCount(); got != 1 {
		t.Fatalf("provider called %d times, want exactly 1", got)
	}
}

// --- 2. Duplicate delivery ----------------------------------------------

// TestIntegration_Runtime_DuplicateSubmitResult_Idempotent proves invariant
// 6 (retries must not duplicate logical effects) at the exact seam Runtime
// itself uses: if the same SubmitResult call is ever issued twice (a
// process retry after an ambiguous network/DB response, for instance —
// Runtime's own submitResult has exactly one such call), the second call
// must return the identical cached outcome rather than re-processing, and
// the Workflow must not transition twice.
func TestIntegration_Runtime_DuplicateSubmitResult_Idempotent(t *testing.T) {
	provider := &fakeProvider{result: ports.IntelligenceResult{Text: "hello", ModelID: "fake-model", Provider: "fake-provider"}}
	notifier := &fakeNotifier{}
	rt, app := requireRealRuntime(t, provider, notifier)
	workflowID, _ := startedReadyWorkflow(t, app)

	rt.Sweep(context.Background())
	rt.Wait()

	latest, err := app.Exec.GetLatestResult(context.Background(), app.Fixtures.Organization().OrganizationID, workflowID)
	if err != nil {
		t.Fatalf("GetLatestResult: %v", err)
	}
	w := loadWorkflow(t, app, workflowID)

	req := application.SubmitResultRequest{
		RequestID: uuid.New(), IdempotencyKey: latest.IdempotencyKey,
		ResultID: latest.ResultID, ExpectedVersion: w.Version,
	}
	// The real first submission already happened inside rt.Sweep/execute
	// above using this exact IdempotencyKey. Replaying it here is the
	// "duplicate delivery."
	replay := app.SubmitResult(context.Background(), req)
	if replay.Outcome != application.Accepted {
		t.Fatalf("replayed SubmitResult outcome = %s, want ACCEPTED (cached)", replay.Outcome)
	}

	again := loadWorkflow(t, app, workflowID)
	if again.Version != w.Version || again.State != w.State {
		t.Fatalf("workflow changed on replay: was version=%d state=%s, now version=%d state=%s",
			w.Version, w.State, again.Version, again.State)
	}
}

// --- 3. Worker crash + 4. Lease expiration ------------------------------

// TestIntegration_Runtime_WorkerCrash_ReclaimedAndRetried simulates a
// worker that claims an ExecutionAttempt and then vanishes — crashes,
// killed, network partition — never calling RecordDispatched or
// RecordTerminal. Nothing in this codebase can force-detect a dead
// goroutine; the lease timeout is the only mechanism, hence "worker crash"
// and "lease expiration" are the same underlying test at this level (the
// persistence-layer tests already isolate the pure timing boundary — see
// TestExecutionRepository_ReclaimExpiredLeases_LeavesNonExpiredAlone).
func TestIntegration_Runtime_WorkerCrash_ReclaimedAndRetried(t *testing.T) {
	provider := &fakeProvider{}
	notifier := &fakeNotifier{}
	rt, app := requireRealRuntime(t, provider, notifier)
	workflowID, _ := startedReadyWorkflow(t, app)
	orgID := app.Fixtures.Organization().OrganizationID

	// Claim directly (bypassing execute()) with an already-expired lease —
	// exactly what a crashed worker's claim looks like from the outside:
	// CLAIMED, never dispatched, never terminal.
	claims, err := rt.Exec.ClaimDueIntents(context.Background(), orgID, 10, -time.Minute, "crashed-worker")
	if err != nil {
		t.Fatalf("ClaimDueIntents: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("claimed %d intents, want 1", len(claims))
	}

	rt.reclaimAbandoned(context.Background())

	if got := provider.callCount(); got != 0 {
		t.Fatalf("provider called %d times, want 0 (the crashed claim must never itself dispatch)", got)
	}
	// attempt_number(1) < MaxAttempts(3) — must be scheduled for retry, not
	// failed outright, and the Workflow must still be READY: the lost
	// worker did not permanently strand this work (invariant 7).
	w := loadWorkflow(t, app, workflowID)
	if w.State != workflow.StateReady {
		t.Fatalf("workflow state = %s, want still READY after one reclaimed attempt", w.State)
	}
	units, err := rt.Exec.ListExecutionUnits(context.Background(), orgID, workflowID)
	if err != nil {
		t.Fatalf("ListExecutionUnits: %v", err)
	}
	if len(units) != 1 || units[0].IntentStatus != "PENDING" {
		t.Fatalf("intent status after reclaim = %+v, want exactly one unit, status PENDING", units)
	}
	// Not asserting DueAt is strictly in the future: computeBackoff uses
	// full jitter (rand.Int63n(backoff+1)), which can legitimately return
	// 0 — an immediately-due retry is still a correct schedule, not a
	// timing bug. The real invariant already checked above (PENDING, not
	// FAILED_TERMINAL) is what actually matters here.
}

// TestIntegration_Runtime_LeaseReclaim_ExhaustsAfterMaxAttempts drives the
// same crash three times in a row (the fixture CapabilityDefinition's
// MaxAttempts) and confirms the fourth reclaim gives up correctly: no more
// retries, the Workflow is failed rather than left claimed forever. Uses
// ExecutionRepository.ScheduleRetry directly with an already-past dueAt
// between rounds instead of waiting out Runtime's own real (jittered)
// backoff — this manufactures "three prior attempts already happened"
// deterministically without a real sleep, the same backdating idiom every
// other test in this file and its persistence-layer sibling already uses
// for due_at/lease_expires_at.
func TestIntegration_Runtime_LeaseReclaim_ExhaustsAfterMaxAttempts(t *testing.T) {
	provider := &fakeProvider{}
	notifier := &fakeNotifier{}
	rt, app := requireRealRuntime(t, provider, notifier)
	workflowID, _ := startedReadyWorkflow(t, app)
	orgID := app.Fixtures.Organization().OrganizationID
	ctx := context.Background()

	maxAttempts := app.Fixtures.Capability().Retry.MaxAttempts
	var intentID uuid.UUID
	for i := 0; i < maxAttempts; i++ {
		claims, err := rt.Exec.ClaimDueIntents(ctx, orgID, 10, -time.Minute, "crashed-worker")
		if err != nil {
			t.Fatalf("round %d ClaimDueIntents: %v", i, err)
		}
		if len(claims) != 1 {
			t.Fatalf("round %d claimed %d intents, want 1", i, len(claims))
		}
		intentID = claims[0].Intent.IntentID

		if i < maxAttempts-1 {
			// Not exhausted yet: reclaim should schedule a retry. Force it
			// immediately due instead of waiting for real backoff.
			rt.reclaimAbandoned(ctx)
			if err := rt.Exec.ScheduleRetry(ctx, orgID, intentID, time.Now().UTC().Add(-time.Second)); err != nil {
				t.Fatalf("round %d ScheduleRetry override: %v", i, err)
			}
		}
	}

	// This is the maxAttempts-th claimed-but-abandoned attempt — reclaiming
	// it now must exhaust retries and fail the Workflow.
	rt.reclaimAbandoned(ctx)

	w := loadWorkflow(t, app, workflowID)
	if w.State != workflow.StateFailed {
		t.Fatalf("workflow state = %s, want FAILED after %d exhausted lease-expired attempts", w.State, maxAttempts)
	}
	if provider.callCount() != 0 {
		t.Fatalf("provider called %d times, want 0 — every attempt here was abandoned before dispatch, never real work", provider.callCount())
	}
}

// --- 5. Concurrent claim (Runtime-level) --------------------------------

func TestIntegration_Runtime_ConcurrentSweep_DispatchesExactlyOnce(t *testing.T) {
	provider := &fakeProvider{result: ports.IntelligenceResult{Text: "hi", ModelID: "fake-model", Provider: "fake-provider"}}
	notifier := &fakeNotifier{}
	rt, app := requireRealRuntime(t, provider, notifier)
	workflowID, _ := startedReadyWorkflow(t, app)

	done := make(chan struct{})
	for i := 0; i < 5; i++ {
		go func() {
			rt.Sweep(context.Background())
			done <- struct{}{}
		}()
	}
	for i := 0; i < 5; i++ {
		<-done
	}
	rt.Wait()

	if got := provider.callCount(); got != 1 {
		t.Fatalf("provider called %d times across 5 concurrent sweeps, want exactly 1", got)
	}
	w := loadWorkflow(t, app, workflowID)
	if w.State != workflow.StateCompleted {
		t.Fatalf("workflow state = %s, want COMPLETED", w.State)
	}
}

// TestIntegration_Runtime_CompletionRacesLeaseExpiry_ExactlyOneWins is
// scenario 7 of docs/testing/concurrency-guarantees.md. Unlike the
// worker-crash/lease-reclaim tests above (which construct a specific
// sequential ordering — crash, then reclaim), this launches RecordTerminal
// and ReclaimExpiredLeases as two genuinely unsynchronized goroutines
// against the same already-expired attempt, no gate between them: the only
// invariant worth proving is "whichever wins, never both, never neither" —
// forcing a specific interleaving would hide exactly the race this exists
// to exercise.
func TestIntegration_Runtime_CompletionRacesLeaseExpiry_ExactlyOneWins(t *testing.T) {
	provider := &fakeProvider{}
	notifier := &fakeNotifier{}
	rt, app := requireRealRuntime(t, provider, notifier)
	startedReadyWorkflow(t, app)
	orgID := app.Fixtures.Organization().OrganizationID
	ctx := context.Background()

	claims, err := rt.Exec.ClaimDueIntents(ctx, orgID, 10, -time.Minute, "worker")
	if err != nil {
		t.Fatalf("ClaimDueIntents: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("claimed %d intents, want 1", len(claims))
	}
	attempt := claims[0].Attempt
	if attempt.LeaseFencingToken == nil {
		t.Fatal("claimed attempt has no LeaseFencingToken")
	}
	fencingToken := *attempt.LeaseFencingToken

	var wg sync.WaitGroup
	var recordErr error
	var reclaimed []execution.ClaimedExecution
	var reclaimErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		recordErr = rt.Exec.RecordTerminal(ctx, attempt.AttemptID, fencingToken, execution.StatusSucceeded, nil)
	}()
	go func() {
		defer wg.Done()
		reclaimed, reclaimErr = rt.Exec.ReclaimExpiredLeases(ctx, orgID, 10)
	}()
	wg.Wait()

	if reclaimErr != nil {
		t.Fatalf("ReclaimExpiredLeases: %v", reclaimErr)
	}
	reclaimedThisAttempt := false
	for _, c := range reclaimed {
		if c.Attempt.AttemptID == attempt.AttemptID {
			reclaimedThisAttempt = true
		}
	}
	completed := recordErr == nil
	if completed == reclaimedThisAttempt {
		t.Fatalf("completion succeeded=%v, reclaimed=%v — want exactly one to win, never both and never neither", completed, reclaimedThisAttempt)
	}
	if !completed && !errors.Is(recordErr, ports.ErrConflict) {
		t.Fatalf("RecordTerminal lost the race with err = %v, want ports.ErrConflict (stale fencing token)", recordErr)
	}
}

// --- 6. Retry -------------------------------------------------------------

func TestIntegration_Runtime_ProviderRetryableFailure_SchedulesRetryNotFailure(t *testing.T) {
	provider := &fakeProvider{err: context.DeadlineExceeded, retryable: true}
	notifier := &fakeNotifier{}
	rt, app := requireRealRuntime(t, provider, notifier)
	workflowID, _ := startedReadyWorkflow(t, app)
	orgID := app.Fixtures.Organization().OrganizationID

	rt.Sweep(context.Background())
	rt.Wait()

	w := loadWorkflow(t, app, workflowID)
	if w.State != workflow.StateReady {
		t.Fatalf("workflow state = %s, want still READY after one retryable failure", w.State)
	}
	units, err := rt.Exec.ListExecutionUnits(context.Background(), orgID, workflowID)
	if err != nil {
		t.Fatalf("ListExecutionUnits: %v", err)
	}
	if len(units) != 1 || units[0].IntentStatus != "PENDING" {
		t.Fatalf("intent status after retryable failure = %+v, want PENDING (rescheduled)", units)
	}
	if len(units[0].Attempts) != 1 || units[0].Attempts[0].Status != "FAILED_RETRYABLE" {
		t.Fatalf("attempts = %+v, want exactly one FAILED_RETRYABLE", units[0].Attempts)
	}
}

// --- 7. Provider failure (terminal) --------------------------------------

func TestIntegration_Runtime_ProviderTerminalFailure_FailsWorkflow(t *testing.T) {
	provider := &fakeProvider{err: context.Canceled, retryable: false}
	notifier := &fakeNotifier{}
	rt, app := requireRealRuntime(t, provider, notifier)
	workflowID, _ := startedReadyWorkflow(t, app)

	rt.Sweep(context.Background())
	rt.Wait()

	w := loadWorkflow(t, app, workflowID)
	if w.State != workflow.StateFailed {
		t.Fatalf("workflow state = %s, want FAILED after a non-retryable provider error", w.State)
	}
	if got := provider.callCount(); got != 1 {
		t.Fatalf("provider called %d times, want exactly 1 (non-retryable errors must not retry)", got)
	}
}

// --- 9. Runtime notification failure --------------------------------------

func TestIntegration_Runtime_NotifierFailure_DoesNotBlockDispatch(t *testing.T) {
	provider := &fakeProvider{result: ports.IntelligenceResult{Text: "hi", ModelID: "fake-model", Provider: "fake-provider"}}
	notifier := &fakeNotifier{failAll: true}
	rt, app := requireRealRuntime(t, provider, notifier)
	workflowID, _ := startedReadyWorkflow(t, app)

	rt.Sweep(context.Background())
	rt.Wait()

	w := loadWorkflow(t, app, workflowID)
	if w.State != workflow.StateCompleted {
		t.Fatalf("workflow state = %s, want COMPLETED — a failing Notifier (best-effort hint, invariant 3) must never block a real state transition", w.State)
	}
	if notifier.callCount() == 0 {
		t.Fatalf("notifier was never called — test isn't actually exercising the failure path")
	}
}

// --- 10. Stale version -----------------------------------------------------

// TestIntegration_Runtime_StaleIntentVersion_DispatchDenied exercises
// Runtime's own handling of AuthorizeDispatch's DENY outcome, complementing
// (not duplicating) internal/application's
// TestIntegration_AuthorizeDispatch_StaleWorkflowVersionDenied, which tests
// AuthorizeDispatch directly. No legitimate Application-level path bumps a
// READY Workflow's version while its intent stays outstanding (that test's
// own comment explains why), so a hand-built stale intent is the correct,
// established way to reach this state — exactly what that test already
// does; this one instead drives it through execute() itself to prove
// Runtime reacts correctly: no dispatch, attempt marked terminal, no panic.
func TestIntegration_Runtime_StaleIntentVersion_DispatchDenied(t *testing.T) {
	provider := &fakeProvider{result: ports.IntelligenceResult{Text: "should never be reached"}}
	notifier := &fakeNotifier{}
	rt, app := requireRealRuntime(t, provider, notifier)
	workflowID, version := startedReadyWorkflow(t, app)
	orgID := app.Fixtures.Organization().OrganizationID

	claims, err := rt.Exec.ClaimDueIntents(context.Background(), orgID, 10, time.Minute, "worker")
	if err != nil {
		t.Fatalf("ClaimDueIntents: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("claimed %d intents, want 1", len(claims))
	}
	stale := claims[0].Intent
	stale.WorkflowVersion = version - 1 // deliberately wrong, per the established pattern

	rt.execute(context.Background(), claims[0].Attempt, stale)

	if got := provider.callCount(); got != 0 {
		t.Fatalf("provider called %d times, want 0 (a DENY must never reach dispatch)", got)
	}
	units, err := rt.Exec.ListExecutionUnits(context.Background(), orgID, workflowID)
	if err != nil {
		t.Fatalf("ListExecutionUnits: %v", err)
	}
	if len(units[0].Attempts) != 1 || units[0].Attempts[0].Status != "FAILED_TERMINAL" {
		t.Fatalf("attempts = %+v, want exactly one FAILED_TERMINAL", units[0].Attempts)
	}
}

// --- 11. Cancellation -------------------------------------------------------

// TestIntegration_Runtime_CancellationDuringDispatch_LateResultRejected
// proves invariant 9 in the cancellation-specific case: a Result for an
// attempt that was in flight when its Workflow got cancelled must never be
// silently applied once the Workflow has already moved on.
func TestIntegration_Runtime_CancellationDuringDispatch_LateResultRejected(t *testing.T) {
	provider := &fakeProvider{}
	notifier := &fakeNotifier{}
	rt, app := requireRealRuntime(t, provider, notifier)
	workflowID, version := startedReadyWorkflow(t, app)
	ctx := context.Background()

	claims, err := rt.Exec.ClaimDueIntents(ctx, app.Fixtures.Organization().OrganizationID, 10, time.Minute, "worker")
	if err != nil {
		t.Fatalf("ClaimDueIntents: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("claimed %d intents, want 1", len(claims))
	}
	inFlight := claims[0]

	// Cancel the READY workflow — Phase 3 Slice 2 semantics: READY cancel
	// requires approval, so drive it through that path to a real CANCELLED.
	pending := app.CancelWorkflow(ctx, application.CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: version, RequestingPrincipalID: app.Fixtures.TriggerPrincipal().PrincipalID,
	})
	if pending.Outcome != application.ApprovalRequired || pending.ApprovalID == nil {
		t.Fatalf("CancelWorkflow outcome = %s (reasons: %v), want APPROVAL_REQUIRED", pending.Outcome, pending.Reasons)
	}
	resolved := app.ResolveApproval(ctx, application.ResolveApprovalRequest{ApprovalID: *pending.ApprovalID, Approve: true, DecidingPrincipal: app.Fixtures.ApproverPrincipal()})
	if resolved.Outcome != application.Accepted || resolved.Workflow.State != "CANCELLED" {
		t.Fatalf("ResolveApproval outcome = %s, workflow state = %s, want ACCEPTED/CANCELLED", resolved.Outcome, resolved.Workflow.State)
	}

	// The attempt claimed before cancellation now "completes" — its
	// worker never knew the Workflow moved on.
	resultID := uuid.New()
	now := time.Now().UTC()
	res := &result.Result{
		ResultID:             resultID,
		OrganizationID:       inFlight.Intent.OrganizationID,
		ResultType:           "INTELLIGENCE_TEXT_GENERATION",
		WorkflowID:           workflowID,
		ObjectiveID:          app.Fixtures.Objective().ObjectiveID,
		IntentID:             inFlight.Intent.IntentID,
		AttemptID:            inFlight.Attempt.AttemptID,
		CapabilityRequestID:  inFlight.Attempt.CapabilityRequestID,
		IdempotencyKey:       inFlight.Intent.IdempotencyKey + ":" + resultID.String(),
		ProducingPrincipalID: app.Fixtures.TriggerPrincipal().PrincipalID,
		ProviderAdapter:      "fake-provider",
		ModelID:              "fake-model",
		Outcome:              result.OutcomeSucceeded,
		Output:               map[string]any{"text": "arrived too late"},
		StartedAt:            inFlight.Attempt.CreatedAt,
		ObservedAt:           now,
		ReportedAt:           now,
	}
	if err := app.Exec.SaveResult(ctx, res); err != nil {
		t.Fatalf("save late result: %v", err)
	}

	late := app.SubmitResult(ctx, application.SubmitResultRequest{
		RequestID: uuid.New(), IdempotencyKey: res.IdempotencyKey,
		ResultID: res.ResultID, ExpectedVersion: version, // the version this attempt was claimed against — now stale
	})
	// Rejected, not Conflict: ValidateResultProposal's Kernel-level check
	// (current.State != StateReady) catches this deterministically before
	// CommitTransition's version CAS is ever reached, since the
	// cancellation already fully committed — current.State is CANCELLED by
	// the time this call runs, not merely a newer version of READY. This
	// is the same "two individually-correct rejection categories depending
	// on where staleness is detected" distinction the 2026-08-24
	// test-determinism investigation (root cause A) already documented for
	// the equivalent Load/Commit race in StartWorkflow; here it isn't even
	// a race, since the cancellation is fully sequenced before this call —
	// so the Kernel-level path is the only one reachable, deterministically.
	if late.Outcome != application.Rejected {
		t.Fatalf("late SubmitResult after cancellation outcome = %s, want REJECTED", late.Outcome)
	}

	w := loadWorkflow(t, app, workflowID)
	if w.State != "CANCELLED" {
		t.Fatalf("workflow state after rejected late result = %s, want still CANCELLED (must not be overwritten)", w.State)
	}
}

// TestIntegration_Runtime_CancellationRacesInFlightDispatch_LateResultRejected
// strengthens TestIntegration_Runtime_CancellationDuringDispatch_LateResultRejected
// above to a genuine race: that test's own comment says "it isn't even a
// race, since the cancellation is fully sequenced before this call" —
// dispatch is never attempted at all. Here dispatch genuinely starts
// (AuthorizeDispatch and RecordDispatched both run for real) and
// Provider.Generate blocks mid-call, so CancelWorkflow/ResolveApproval run
// while the dispatch is provably still in flight — the same
// blocking-channel technique TestIntegration_Runtime_StopWork_CancelsInFlightDispatch
// uses for the identical reason: a channel proves concurrency
// deterministically, a sleep only makes it likely.
func TestIntegration_Runtime_CancellationRacesInFlightDispatch_LateResultRejected(t *testing.T) {
	block := make(chan struct{})
	provider := &fakeProvider{block: block, result: ports.IntelligenceResult{Text: "arrived too late", ModelID: "fake-model", Provider: "fake-provider"}}
	notifier := &fakeNotifier{notified: make(chan struct{})}
	rt, app := requireRealRuntime(t, provider, notifier)
	workflowID, version := startedReadyWorkflow(t, app)
	ctx := context.Background()

	rt.Sweep(ctx) // spawns the dispatch goroutine

	// Wait for a real signal that AuthorizeDispatch+RecordDispatched
	// already committed and Generate has been entered (notifyChanged only
	// runs after RecordDispatched succeeds, immediately before Generate) —
	// not just that Sweep returned, which only guarantees claiming
	// happened, not dispatch. Without this, the cancellation below could
	// race ahead of dispatch entirely and this test would only be
	// "probably" concurrent, exactly the sleep-shaped gap this matrix is
	// supposed to close.
	select {
	case <-notifier.notified:
	case <-time.After(10 * time.Second):
		t.Fatal("dispatch never notified — RecordDispatched must commit before cancellation races it")
	}

	// Cancellation runs synchronously on this goroutine while the dispatch
	// goroutine sits blocked inside Generate — genuinely concurrent with an
	// in-flight dispatch, not sequenced around one that never started.
	pending := app.CancelWorkflow(ctx, application.CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: version, RequestingPrincipalID: app.Fixtures.TriggerPrincipal().PrincipalID,
	})
	if pending.Outcome != application.ApprovalRequired || pending.ApprovalID == nil {
		t.Fatalf("CancelWorkflow outcome = %s (reasons: %v), want APPROVAL_REQUIRED", pending.Outcome, pending.Reasons)
	}
	resolved := app.ResolveApproval(ctx, application.ResolveApprovalRequest{ApprovalID: *pending.ApprovalID, Approve: true, DecidingPrincipal: app.Fixtures.ApproverPrincipal()})
	if resolved.Outcome != application.Accepted || resolved.Workflow.State != "CANCELLED" {
		t.Fatalf("ResolveApproval outcome = %s, workflow state = %s, want ACCEPTED/CANCELLED", resolved.Outcome, resolved.Workflow.State)
	}

	// Let the still-in-flight dispatch finish; execute()'s own SubmitResult
	// call must be rejected against the by-then-CANCELLED workflow, not
	// silently applied on top of it.
	close(block)
	rt.Wait()

	w := loadWorkflow(t, app, workflowID)
	if w.State != "CANCELLED" {
		t.Fatalf("workflow state after an in-flight dispatch completed post-cancellation = %s, want still CANCELLED (must not be overwritten)", w.State)
	}
}

// --- 12. Recovery after restart --------------------------------------------

// TestIntegration_Runtime_RecoveryAfterRestart_SecondRuntimeReclaimsAndCompletes
// is the definitive proof for invariant 7 across a full process boundary,
// not just within one: a second, entirely independent Runtime+Application
// pair (simulating a fresh companyd process after a restart, sharing
// nothing in memory with the first — only the database) reclaims the first
// "process"'s abandoned work and drives it to completion.
func TestIntegration_Runtime_RecoveryAfterRestart_SecondRuntimeReclaimsAndCompletes(t *testing.T) {
	providerA := &fakeProvider{}
	notifierA := &fakeNotifier{}
	rtA, appA := requireRealRuntime(t, providerA, notifierA)
	workflowID, _ := startedReadyWorkflow(t, appA)
	orgID := appA.Fixtures.Organization().OrganizationID

	// "Process A" claims and then crashes — never dispatches.
	claimedByA, err := rtA.Exec.ClaimDueIntents(context.Background(), orgID, 10, -time.Minute, "process-a-worker")
	if err != nil {
		t.Fatalf("ClaimDueIntents (process A): %v", err)
	}
	if len(claimedByA) != 1 {
		t.Fatalf("process A claimed %d intents, want 1", len(claimedByA))
	}
	intentID := claimedByA[0].Intent.IntentID

	// "Process B": a second, independent Runtime — same organization
	// fixture (fixed by construction below, not a fresh
	// NewRegistryWithOrganization call, so it addresses the same rows),
	// same database, zero shared Go state with rtA/appA.
	providerB := &fakeProvider{result: ports.IntelligenceResult{Text: "recovered", ModelID: "fake-model", Provider: "fake-provider"}}
	notifierB := &fakeNotifier{}
	pool, err := supabase.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("connect (process B): %v", err)
	}
	t.Cleanup(pool.Close)
	appB := &application.Application{
		Repo: appA.Repo, Pending: appA.Pending, Exec: supabase.NewExecutionRepository(pool),
		Fixtures: appA.Fixtures, Notify: make(chan uuid.UUID, 8),
	}
	rtB := &Runtime{
		Exec: supabase.NewExecutionRepository(pool), App: appB, Provider: providerB,
		ProviderName: "fake-provider-b", ModelID: "fake-model", Fixtures: appA.Fixtures,
		PollInterval: time.Hour, LeaseDuration: time.Minute, Wakeup: make(chan uuid.UUID, 8), Notifier: notifierB,
	}
	rtB.ensureWorkContext()

	// Reclaim first (recovers the abandoned attempt, schedules a retry with
	// a real — possibly nonzero-jitter — backoff), then force that retry
	// immediately due rather than waiting out the real backoff window, same
	// deterministic-bypass idiom used elsewhere in this file. Then a
	// second Sweep claims and completes it.
	rtB.reclaimAbandoned(context.Background())
	if err := rtB.Exec.ScheduleRetry(context.Background(), orgID, intentID, time.Now().UTC().Add(-time.Second)); err != nil {
		t.Fatalf("ScheduleRetry override: %v", err)
	}
	rtB.Sweep(context.Background())
	rtB.Wait()

	if got := providerA.callCount(); got != 0 {
		t.Fatalf("process A's provider called %d times — it crashed, it must never have dispatched", got)
	}
	if got := providerB.callCount(); got != 1 {
		t.Fatalf("process B's provider called %d times, want exactly 1 (it recovered and completed the abandoned work)", got)
	}
	w := loadWorkflow(t, appA, workflowID)
	if w.State != workflow.StateCompleted {
		t.Fatalf("workflow state = %s, want COMPLETED (recovered by process B after process A's crash)", w.State)
	}
}

// TestIntegration_Runtime_RestartMidDispatch_FencedOffAfterSecondProcessCompletes
// strengthens the recovery test above to the harder case it doesn't cover:
// process A's dispatch genuinely starts — AuthorizeDispatch and
// RecordDispatched both commit for real, a notification fires — before
// being abandoned mid-Generate (Go cannot kill a goroutine; not waiting on
// it is the closest analogue to a process dying mid-call, same technique as
// the cancellation-race test above). This also folds in scenario 11
// (notification succeeds, then the worker crashes before completion):
// notifierA.callCount() > 0 proves a real notification already fired
// before process B ever gets involved, and the assertions below prove that
// stale notification never causes a second side effect.
func TestIntegration_Runtime_RestartMidDispatch_FencedOffAfterSecondProcessCompletes(t *testing.T) {
	block := make(chan struct{})
	providerA := &fakeProvider{block: block}
	notifierA := &fakeNotifier{notified: make(chan struct{})}
	rtA, appA := requireRealRuntime(t, providerA, notifierA)
	workflowID, _ := startedReadyWorkflow(t, appA)
	orgID := appA.Fixtures.Organization().OrganizationID

	// Force an immediately-expired lease at claim time (the same
	// negative-duration idiom every other test in this file uses), so
	// reclaim below can act on this DISPATCHED attempt without waiting out
	// LeaseDuration for real. RecordDispatched itself doesn't touch
	// lease_expires_at (no renewal mechanism exists — scenario 6), so the
	// lease set at claim carries through dispatch unchanged.
	rtA.LeaseDuration = -time.Second

	// "Process A" dispatches for real and is then abandoned mid-Generate —
	// its goroutine is deliberately never Wait()ed on here; it stays parked
	// on <-block until this test unblocks it at the very end. Sweep itself
	// only claims synchronously; RecordDispatched/notifyChanged/Generate run
	// in dispatchBounded's asynchronous goroutine, so this test must not
	// assume they've happened just because Sweep returned — it blocks on
	// notifierA.notified instead, a real signal (not a sleep or poll) that
	// notifyChanged actually ran, which only happens after RecordDispatched
	// committed. This is exactly scenario 11's precondition: the
	// notification must have genuinely fired before process B ever gets
	// involved, not merely be assumed to have.
	rtA.Sweep(context.Background())
	select {
	case <-notifierA.notified:
	case <-time.After(10 * time.Second):
		t.Fatal("process A's dispatch never notified — RecordDispatched must commit (and notify) before reclaim runs")
	}

	units, err := rtA.Exec.ListExecutionUnits(context.Background(), orgID, workflowID)
	if err != nil {
		t.Fatalf("ListExecutionUnits: %v", err)
	}
	if len(units) != 1 || len(units[0].Attempts) != 1 {
		t.Fatalf("units = %+v, want exactly 1 unit with 1 attempt", units)
	}
	intentID := units[0].IntentID

	// "Process B": independent Runtime/Application, same database, zero
	// shared Go state — the same pattern
	// TestIntegration_Runtime_RecoveryAfterRestart_SecondRuntimeReclaimsAndCompletes
	// uses, applied to a mid-dispatch abandonment instead of a pre-dispatch
	// one.
	providerB := &fakeProvider{result: ports.IntelligenceResult{Text: "recovered", ModelID: "fake-model", Provider: "fake-provider"}}
	notifierB := &fakeNotifier{}
	pool, err := supabase.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("connect (process B): %v", err)
	}
	t.Cleanup(pool.Close)
	appB := &application.Application{
		Repo: appA.Repo, Pending: appA.Pending, Exec: supabase.NewExecutionRepository(pool),
		Fixtures: appA.Fixtures, Notify: make(chan uuid.UUID, 8),
	}
	rtB := &Runtime{
		Exec: supabase.NewExecutionRepository(pool), App: appB, Provider: providerB,
		ProviderName: "fake-provider-b", ModelID: "fake-model", Fixtures: appA.Fixtures,
		PollInterval: time.Hour, LeaseDuration: time.Minute, Wakeup: make(chan uuid.UUID, 8), Notifier: notifierB,
	}
	rtB.ensureWorkContext()

	rtB.reclaimAbandoned(context.Background())
	if err := rtB.Exec.ScheduleRetry(context.Background(), orgID, intentID, time.Now().UTC().Add(-time.Second)); err != nil {
		t.Fatalf("ScheduleRetry override: %v", err)
	}
	rtB.Sweep(context.Background())
	rtB.Wait()

	if got := providerB.callCount(); got != 1 {
		t.Fatalf("process B's provider called %d times, want exactly 1 (it recovered process A's mid-dispatch abandonment)", got)
	}
	w := loadWorkflow(t, appA, workflowID)
	if w.State != workflow.StateCompleted {
		t.Fatalf("workflow state = %s, want COMPLETED after process B's recovery", w.State)
	}

	// Now let process A's stuck dispatch finish. Its fencing token is
	// stale — reclaim already bumped it — so its own completion must be
	// fenced off (ports.ErrConflict inside submitResult's RecordTerminal
	// call, logged not fatal), never double-applied on top of process B's
	// already-COMPLETED workflow. providerA itself still gets called once
	// (Generate finally returns) — that's expected and fine; what matters
	// is the workflow state afterward.
	close(block)
	waitDone := make(chan struct{})
	go func() { rtA.Wait(); close(waitDone) }()
	select {
	case <-waitDone:
	case <-time.After(10 * time.Second):
		t.Fatal("process A's abandoned dispatch never finished after unblocking")
	}
	w = loadWorkflow(t, appA, workflowID)
	if w.State != workflow.StateCompleted {
		t.Fatalf("workflow state after process A's late completion = %s, want still COMPLETED (must not be double-applied)", w.State)
	}
}

// --- Shutdown mechanics (StopWork / workCtx separation) --------------------

// TestIntegration_Runtime_StopWork_CancelsInFlightDispatch is the direct
// proof for daemon.Daemon.Shutdown's new behavior: when a bounded drain
// wait gives up, it calls Runtime.StopWork, which must actually reach an
// in-flight dispatch (execute()'s dispatchCtx is derived from workCtx) and
// unblock it — otherwise Shutdown's timeout would be cosmetic, the
// underlying goroutine still running unbounded. This also implicitly
// proves the other half of the same fix: Sweep's own ctx (used only for
// claim/reclaim queries) is independent of workCtx, so a caller can cancel
// in-flight work without that cancellation having already torn down the
// claim path first — the two contexts really are separate, not aliases.
func TestIntegration_Runtime_StopWork_CancelsInFlightDispatch(t *testing.T) {
	// block is deliberately never closed — if execute() reaches Generate at
	// all, only ctx cancellation can end that call. In practice
	// AuthorizeDispatch's own context-aware DB query (LoadWorkflow) is
	// usually the first thing to observe the cancellation, before Generate
	// is even reached — which is a fine, equally valid proof of the same
	// claim (workCtx cancellation reaches the in-flight goroutine), so this
	// test doesn't assert exactly where cancellation is first observed,
	// only that the goroutine reliably exits instead of running forever.
	block := make(chan struct{})
	provider := &fakeProvider{block: block}
	notifier := &fakeNotifier{}
	rt, app := requireRealRuntime(t, provider, notifier)
	startedReadyWorkflow(t, app)

	rt.Sweep(context.Background())

	// No sleep before StopWork: whether the spawned goroutine has reached
	// Generate's blocking select yet or not doesn't matter to correctness
	// here. Either ctx.Done() is already closed by the time Generate's
	// select runs, or StopWork's cancellation arrives while Generate is
	// already parked in that select — both converge on the identical
	// outcome (Generate returns ctx.Err()), so inserting a delay would only
	// add latency without proving anything a race-free ordering doesn't
	// already prove.
	rt.StopWork()

	waitDone := make(chan struct{})
	go func() {
		rt.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-time.After(10 * time.Second):
		t.Fatal("rt.Wait() never returned after StopWork — in-flight dispatch was not actually cancelled")
	}
}
