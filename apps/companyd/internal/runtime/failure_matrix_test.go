package runtime

// This file is the index for the runtime's failure-injection test matrix: 9
// scenarios (process crash mid-execution, DB failure during commit, provider
// timeout, duplicate dispatch, lease expiry, stale worker rejoining,
// duplicate result delivery, outbox write failure, shutdown drain), each set
// up, injected, recovered from, and asserted against its documented
// invariant. Six of the nine already had deterministic coverage elsewhere in
// this codebase before this file existed — duplicating them here would
// violate this package's own no-redundant-tests discipline
// (docs/testing/concurrency-guarantees.md), so this index only points to
// them; only the two genuine runtime-scoped gaps are defined below. Shutdown
// drain is proven in internal/daemon/daemon_test.go (a different package —
// Daemon, not Runtime, owns that orchestration); outbox write failure is
// proven in internal/adapters/notify/realtime/sweeper_test.go (a different
// package — the outbox is not part of Runtime's own commit path at all).
//
//  1. Process crash mid-execution — a persisted intent is never lost; the
//     lease expires and a later Sweep reclaims it as a new attempt
//     (docs/domain/execution.md invariant 7; docs/testing/failure-injection.md).
//     Proven by: TestIntegration_Runtime_WorkerCrash_ReclaimedAndRetried,
//     TestIntegration_Runtime_RecoveryAfterRestart_SecondRuntimeReclaimsAndCompletes,
//     TestIntegration_Runtime_RestartMidDispatch_FencedOffAfterSecondProcessCompletes
//     (runtime_test.go).
//
//  2. DB failure during commit — a commit-path write (RecordTerminal) that
//     fails must leave the attempt exactly where a crashed worker would
//     (non-terminal, lease still ticking), never falsely completing or
//     losing the workflow (docs/domain/execution.md: an unpersisted-into-
//     the-workflow result never becomes authoritative).
//     Proven by: TestIntegration_Runtime_DBFailureDuringRecordTerminal_ReclaimedAndRetried
//     (this file — no prior coverage existed for this).
//
//  3. Provider timeout — dispatchCtx's own deadline elapsing produces a raw,
//     unclassified ctx.Err(), which ports.IsRetryable treats as terminal by
//     default: "fail closed on retries/fallback rather than looping on an
//     error nobody vouched for" (ports/provideradapter.go).
//     Proven by: TestIntegration_Runtime_ProviderRetryableFailure_SchedulesRetryNotFailure
//     (runtime_test.go — the adapter-classified-retryable form) and
//     TestIntegration_Runtime_ProviderRealTimeout_FailsClosedNotRetried
//     (this file — the real-deadline, unclassified form; a materially
//     different code path from the former).
//
//  4. Duplicate dispatch — exactly one claim succeeds under FOR UPDATE SKIP
//     LOCKED; the other observes no claim (docs/testing/failure-injection.md).
//     Proven by: TestExecutionRepository_ClaimDueIntents_NoDuplicateUnderConcurrency
//     (adapters/persistence/supabase/integration_test.go),
//     TestIntegration_Runtime_ConcurrentSweep_DispatchesExactlyOnce (runtime_test.go).
//
//  5. Lease expiry — a lease that expires under a slow/lost attempt is
//     recovered without corrupting state if the original worker later
//     reports back (docs/domain/execution.md invariants 7-8).
//     Proven by: TestExecutionRepository_ReclaimExpiredLeases_RecoversAbandonedAttempt
//     (adapters/persistence/supabase/integration_test.go),
//     TestIntegration_Runtime_LeaseReclaim_ExhaustsAfterMaxAttempts,
//     TestIntegration_Runtime_CompletionRacesLeaseExpiry_ExactlyOneWins (runtime_test.go).
//
//  6. Stale worker rejoining — a fenced-off worker's late RecordDispatched/
//     RecordTerminal call using its old fencing token fails cleanly
//     (ports.ErrConflict), never double-applying on top of whatever
//     superseded it (internal/ports/persistence.go's ReclaimExpiredLeases
//     doc comment).
//     Proven by: TestExecutionRepository_ReclaimExpiredLeases_BumpsFencingTokenSoStaleWorkerFailsClosed
//     (adapters/persistence/supabase/integration_test.go),
//     TestIntegration_Runtime_RestartMidDispatch_FencedOffAfterSecondProcessCompletes (runtime_test.go).
//
//  7. Duplicate result delivery — the idempotency-key replay guard returns
//     the original outcome, not a second transition
//     (docs/testing/failure-injection.md).
//     Proven by: TestIntegration_Runtime_DuplicateSubmitResult_Idempotent (runtime_test.go).
//
//  8. Outbox write failure — an unpublished event_outbox row is retried on
//     the next Sweep via MarkPublishFailed, never silently dropped
//     (docs/testing/failure-injection.md). Scoped to a different package:
//     the outbox is written atomically alongside Workflow state changes
//     (internal/adapters/persistence/supabase/workflow_repo.go) and
//     published by an independent poll loop (internal/adapters/notify/realtime),
//     not by Runtime.Sweep.
//     Proven by: TestOutboxRepository_MarkPublishFailed_LeavesRowUnpublished
//     (adapters/persistence/supabase/integration_test.go),
//     TestIntegration_Sweeper_PublishFailure_RetriedOnNextSweep,
//     TestIntegration_Sweeper_PermanentPublishFailure_DoesNotLoopForever
//     (adapters/notify/realtime/sweeper_test.go).
//
//  9. Shutdown drain — bounded wait for in-flight work to finish, then
//     StopWork narrows (never eliminates) the abandonment window; the lease
//     still expires and is reclaimed normally on a later Sweep
//     (docs/architecture/daemon.md; daemon.go's own Shutdown doc comment).
//     Scoped to a different package: Daemon, not Runtime, owns the
//     bounded-wait-then-force policy.
//     Proven by: TestIntegration_Runtime_StopWork_CancelsInFlightDispatch
//     (runtime_test.go — proves StopWork itself reaches an in-flight
//     dispatch) and TestDaemon_Shutdown_BoundedDrainThenStopWork
//     (internal/daemon/daemon_test.go — proves the orchestration around when
//     StopWork fires, previously untested).

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/persistence/supabase"
	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// TestIntegration_Runtime_DBFailureDuringRecordTerminal_ReclaimedAndRetried
// fills matrix scenario 2. SaveResult commits for real (the provider call
// itself succeeded); RecordTerminal — the actual "commit" of the terminal
// transition — is injected to fail once. execute() logs and returns without
// ever calling Application.SubmitResult, so the attempt is left exactly
// where a crashed worker's DISPATCHED attempt would be: non-terminal, lease
// still ticking, its Result row persisted but never authoritative. A second,
// independent "process" (same restart-recovery shape as
// TestIntegration_Runtime_RestartMidDispatch_FencedOffAfterSecondProcessCompletes)
// then reclaims and completes it normally.
func TestIntegration_Runtime_DBFailureDuringRecordTerminal_ReclaimedAndRetried(t *testing.T) {
	provider := &fakeProvider{result: ports.IntelligenceResult{Text: "hello", ModelID: "fake-model", Provider: "fake-provider"}}
	notifier := &fakeNotifier{}
	rtA, appA := requireRealRuntime(t, provider, notifier)
	workflowID, _ := startedReadyWorkflow(t, appA)
	orgID := appA.Fixtures.Organization().OrganizationID

	realExec := rtA.Exec
	rtA.Exec = &commitFailingExec{ExecutionRepository: realExec, failRecordTerminal: 1, err: errors.New("simulated commit failure")}
	// So process B's reclaim below can act without waiting out a real lease
	// — the same negative-duration idiom every other test in this package
	// uses; RecordDispatched doesn't touch lease_expires_at (no renewal
	// mechanism exists), so the lease set at claim carries through dispatch
	// unchanged.
	rtA.LeaseDuration = -time.Second

	rtA.Sweep(context.Background())
	rtA.Wait()

	if got := provider.callCount(); got != 1 {
		t.Fatalf("provider called %d times, want exactly 1 (dispatch itself must succeed; only the commit fails)", got)
	}
	w := loadWorkflow(t, appA, workflowID)
	if w.State != workflow.StateReady {
		t.Fatalf("workflow state = %s, want still READY (a Result whose terminal commit failed must never become authoritative)", w.State)
	}

	units, err := realExec.ListExecutionUnits(context.Background(), orgID, workflowID)
	if err != nil {
		t.Fatalf("ListExecutionUnits: %v", err)
	}
	if len(units) != 1 || len(units[0].Attempts) != 1 {
		t.Fatalf("units = %+v, want exactly 1 unit with 1 attempt", units)
	}
	intentID := units[0].IntentID

	// "Process B": independent Runtime/Application, real Exec (no injected
	// failure), same database, zero shared Go state with rtA/appA.
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
		t.Fatalf("process B's provider called %d times, want exactly 1 (it recovered process A's failed commit)", got)
	}
	final := loadWorkflow(t, appA, workflowID)
	if final.State != workflow.StateCompleted {
		t.Fatalf("workflow state = %s, want COMPLETED after process B's recovery", final.State)
	}
}

// TestIntegration_Runtime_ProviderRealTimeout_FailsClosedNotRetried fills
// matrix scenario 3's real-deadline form. block is deliberately never
// closed, so Generate only returns once dispatchCtx's own timeout elapses —
// a raw ctx.Err(), not a provider-classified error. ports.IsRetryable treats
// any unclassified error as terminal by design (provideradapter.go), so this
// must fail the workflow outright on the first attempt rather than
// scheduling a retry — the opposite of
// TestIntegration_Runtime_ProviderRetryableFailure_SchedulesRetryNotFailure,
// which injects a pre-classified retryable error instead of letting a real
// deadline fire.
func TestIntegration_Runtime_ProviderRealTimeout_FailsClosedNotRetried(t *testing.T) {
	block := make(chan struct{})
	provider := &fakeProvider{block: block}
	notifier := &fakeNotifier{}
	rt, app := requireRealRuntime(t, provider, notifier)
	shortTimeout := app.Fixtures.WithCapabilityTimeout(50 * time.Millisecond)
	app.Fixtures = shortTimeout
	rt.Fixtures = shortTimeout
	workflowID, _ := startedReadyWorkflow(t, app)

	rt.Sweep(context.Background())
	// Returns once the real 50ms dispatchCtx deadline elapses inside
	// Generate's block/ctx.Done() select — no sleep needed, Wait() itself is
	// the deterministic block.
	rt.Wait()

	if got := provider.callCount(); got != 1 {
		t.Fatalf("provider called %d times, want exactly 1", got)
	}
	w := loadWorkflow(t, app, workflowID)
	if w.State != workflow.StateFailed {
		t.Fatalf("workflow state = %s, want FAILED — a raw ctx.Err() from a real deadline is unclassified and must fail closed on the first attempt, not retry", w.State)
	}
}
