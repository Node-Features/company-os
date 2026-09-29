package daemon

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/persistence/supabase"
	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/Node-Features/company-os/apps/companyd/internal/runtime"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// requireRealRuntime builds a Runtime and its Application against the real
// Postgres-backed repositories, skipped unless DATABASE_URL is set. This
// duplicates (rather than imports) internal/runtime's own same-named,
// unexported helper — package-private helpers aren't importable across
// package boundaries, the same reasoning runtime_test.go's own
// startedReadyWorkflow doc comment already gives for not sharing this kind
// of setup cross-package.
func requireRealRuntime(t *testing.T, provider ports.ProviderAdapter) (*runtime.Runtime, *application.Application) {
	t.Helper()
	_ = godotenv.Load("../../.env")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping daemon integration test")
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
	rt := &runtime.Runtime{
		Exec:          supabase.NewExecutionRepository(pool),
		App:           app,
		Provider:      provider,
		ProviderName:  "fake-provider",
		ModelID:       "fake-model",
		Fixtures:      reg,
		PollInterval:  time.Hour,
		LeaseDuration: time.Minute,
		Wakeup:        make(chan uuid.UUID, 8),
	}
	return rt, app
}

func startedReadyWorkflow(t *testing.T, app *application.Application) (workflowID uuid.UUID) {
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
	return workflowID
}

// blockingProvider is a local ports.ProviderAdapter double that never
// returns on its own — Generate only ends via ctx cancellation. Mirrors
// internal/runtime's fakeProvider.block idiom (a channel deliberately never
// closed proves cancellation is what ended the call, not a race against a
// real provider's own latency).
type blockingProvider struct{}

func (blockingProvider) Generate(ctx context.Context, _ ports.IntelligenceRequest) (ports.IntelligenceResult, error) {
	<-ctx.Done()
	return ports.IntelligenceResult{}, ctx.Err()
}

// TestDaemon_Shutdown_BoundedDrainThenStopWork fills failure-matrix scenario
// 9 (shutdown drain) — internal/runtime/failure_matrix_test.go's index
// entry. Runtime.StopWork itself reaching an in-flight dispatch is already
// proven by TestIntegration_Runtime_StopWork_CancelsInFlightDispatch; this
// test's job is the orchestration Daemon.Shutdown adds on top, which had no
// coverage at all before this file: bound the wait, only then call
// StopWork, and — the invariant daemon.go's own doc comment states — never
// falsify the Workflow's state in the process. A dispatch still in flight
// past the deadline has its context cancelled but is not force-killed, so
// any RecordTerminal/SaveResult call it goes on to make runs against an
// already-cancelled context and fails immediately (pgx rejects a cancelled
// context before issuing the query) — execute() logs and returns without
// ever calling Application.SubmitResult, leaving the Workflow exactly where
// it was, its lease still ticking for a later Sweep to reclaim exactly like
// any other lost worker (not re-proven here — that's scenario 1's job).
func TestDaemon_Shutdown_BoundedDrainThenStopWork(t *testing.T) {
	rt, app := requireRealRuntime(t, blockingProvider{})
	workflowID := startedReadyWorkflow(t, app)

	rt.Sweep(context.Background()) // claims and dispatches into blockingProvider, which never returns on its own

	bound := 150 * time.Millisecond
	shutdownCtx, cancel := context.WithTimeout(context.Background(), bound)
	defer cancel()

	d := New(rt)
	start := time.Now()
	err := d.Shutdown(shutdownCtx)
	elapsed := time.Since(start)

	if err != context.DeadlineExceeded {
		t.Fatalf("Shutdown err = %v, want context.DeadlineExceeded (in-flight dispatch never finishes on its own)", err)
	}
	if elapsed < bound {
		t.Fatalf("Shutdown returned after %v, want at least the %v bound (must actually wait, not return instantly)", elapsed, bound)
	}
	if elapsed > 5*time.Second {
		t.Fatalf("Shutdown took %v after its %v bound elapsed, want StopWork to unblock it promptly", elapsed, bound)
	}

	// If StopWork had not really reached the in-flight dispatch, Wait()
	// would block forever (blockingProvider never returns on its own) —
	// the same "let it hang and fail via a bounded safety net, don't race a
	// sleep" idiom internal/runtime's own tests use for this exact check.
	waitDone := make(chan struct{})
	go func() {
		rt.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-time.After(10 * time.Second):
		t.Fatal("rt.Wait() never returned after Shutdown — StopWork did not actually cancel the in-flight dispatch")
	}

	// The core invariant: a dispatch aborted by shutdown must never falsely
	// complete or fail the workflow — RecordTerminal/SaveResult against the
	// now-cancelled context fail before ever reaching SubmitResult, so the
	// workflow is left exactly as it was, safe for a later Sweep to reclaim.
	w, err := app.Repo.LoadWorkflow(context.Background(), app.Fixtures.Organization().OrganizationID, workflowID)
	if err != nil {
		t.Fatalf("LoadWorkflow: %v", err)
	}
	if w.State != workflow.StateReady {
		t.Fatalf("workflow state after shutdown aborted an in-flight dispatch = %s, want still READY (must not falsely complete or fail)", w.State)
	}
}
