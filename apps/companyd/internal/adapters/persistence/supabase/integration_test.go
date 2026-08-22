package supabase

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// requirePool skips the test unless DATABASE_URL is set, per the
// first-slice plan's verification section — these tests exercise the real
// schema applied by supabase/migrations/, not a mock.
func requirePool(t *testing.T) *Pool {
	t.Helper()
	_ = godotenv.Load("../../../../.env")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	pool, err := Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// testOrgID returns a fresh organization_id per call, not the shared
// fixtures.OrganizationID constant. These tests construct workflows,
// intents, and events directly against the repositories, bypassing
// Kernel/Application entirely — there's no FK to a real organizations
// table requiring a specific ID. Using a fresh one per test keeps this
// package's tests isolated from internal/application's own integration
// tests, which do need the real fixture org (Application hardcodes it this
// slice) and run concurrently as a separate package binary under
// `go test ./...`. Sharing one fixed org ID between the two suites let
// them race over the same due ExecutionIntents via ClaimDueIntents's
// broad, non-workflow-scoped claim — a real flake, not a product bug.
func testOrgID() uuid.UUID { return uuid.New() }

func newTestWorkflow(orgID uuid.UUID) *workflow.Workflow {
	now := time.Now().UTC()
	return &workflow.Workflow{
		OrganizationID:        orgID,
		WorkflowID:            uuid.New(),
		Version:               1,
		DefinitionID:          uuid.New(),
		DefinitionVersion:     1,
		ObjectiveID:           uuid.New(),
		State:                 workflow.StatePlanned,
		InitiatingPrincipalID: uuid.New(),
		CorrelationID:         uuid.New(),
		Inputs:                map[string]any{"prompt": "integration test"},
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}

func newTestEvent(orgID, workflowID uuid.UUID, version int64) event.DomainEvent {
	return event.DomainEvent{
		EventID: uuid.New(), OrganizationID: orgID, EventType: event.TypeWorkflowCreated,
		SchemaVersion: 1, SubjectType: "Workflow", SubjectID: workflowID, SubjectVersion: version,
		OccurredAt: time.Now().UTC(), CorrelationID: uuid.New(),
	}
}

func TestWorkflowRepository_CreateAndLoad_RoundTrips(t *testing.T) {
	pool := requirePool(t)
	repo := NewWorkflowRepository(pool)
	ctx := context.Background()

	w := newTestWorkflow(testOrgID())
	events := []event.DomainEvent{newTestEvent(w.OrganizationID, w.WorkflowID, 1)}

	if err := repo.CreateWorkflow(ctx, w, events, uuid.New()); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	loaded, err := repo.LoadWorkflow(ctx, w.OrganizationID, w.WorkflowID)
	if err != nil {
		t.Fatalf("LoadWorkflow: %v", err)
	}
	if loaded.State != workflow.StatePlanned || loaded.Version != 1 {
		t.Errorf("loaded = %+v, want state=PLANNED version=1", loaded)
	}
}

func TestWorkflowRepository_CreateWorkflow_DuplicateIsConflict(t *testing.T) {
	pool := requirePool(t)
	repo := NewWorkflowRepository(pool)
	ctx := context.Background()

	w := newTestWorkflow(testOrgID())
	events := []event.DomainEvent{newTestEvent(w.OrganizationID, w.WorkflowID, 1)}

	if err := repo.CreateWorkflow(ctx, w, events, uuid.New()); err != nil {
		t.Fatalf("first CreateWorkflow: %v", err)
	}
	err := repo.CreateWorkflow(ctx, w, events, uuid.New())
	if err != ports.ErrConflict {
		t.Fatalf("second CreateWorkflow error = %v, want ports.ErrConflict", err)
	}
}

func TestWorkflowRepository_CommitTransition_StaleVersionRejected(t *testing.T) {
	pool := requirePool(t)
	repo := NewWorkflowRepository(pool)
	ctx := context.Background()

	w := newTestWorkflow(testOrgID())
	if err := repo.CreateWorkflow(ctx, w, []event.DomainEvent{newTestEvent(w.OrganizationID, w.WorkflowID, 1)}, uuid.New()); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	next := *w
	next.Version = 2
	next.State = workflow.StateReady
	staleExpected := int64(99) // wrong on purpose
	err := repo.CommitTransition(ctx, &next, staleExpected, []event.DomainEvent{newTestEvent(w.OrganizationID, w.WorkflowID, 2)}, uuid.New(), nil, nil, nil, nil, false)
	if err != ports.ErrConflict {
		t.Fatalf("CommitTransition with stale version error = %v, want ports.ErrConflict", err)
	}

	// The workflow must be unchanged — a rejected compare-and-write commits nothing.
	reloaded, loadErr := repo.LoadWorkflow(ctx, w.OrganizationID, w.WorkflowID)
	if loadErr != nil {
		t.Fatalf("LoadWorkflow after failed commit: %v", loadErr)
	}
	if reloaded.Version != 1 || reloaded.State != workflow.StatePlanned {
		t.Errorf("workflow changed despite rejected commit: %+v", reloaded)
	}
}

func TestWorkflowRepository_CommitTransition_AtomicWithEvents(t *testing.T) {
	pool := requirePool(t)
	repo := NewWorkflowRepository(pool)
	ctx := context.Background()

	w := newTestWorkflow(testOrgID())
	if err := repo.CreateWorkflow(ctx, w, []event.DomainEvent{newTestEvent(w.OrganizationID, w.WorkflowID, 1)}, uuid.New()); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	next := *w
	next.Version = 2
	next.State = workflow.StateReady
	evt := newTestEvent(w.OrganizationID, w.WorkflowID, 2)
	if err := repo.CommitTransition(ctx, &next, 1, []event.DomainEvent{evt}, uuid.New(), nil, nil, nil, nil, false); err != nil {
		t.Fatalf("CommitTransition: %v", err)
	}

	loaded, err := repo.LoadWorkflow(ctx, w.OrganizationID, w.WorkflowID)
	if err != nil {
		t.Fatalf("LoadWorkflow: %v", err)
	}
	if loaded.Version != 2 || loaded.State != workflow.StateReady {
		t.Errorf("loaded = %+v, want version=2 state=READY", loaded)
	}

	var eventCount int
	if err := pool.pool.QueryRow(ctx, `SELECT count(*) FROM domain_events WHERE event_id = $1`, evt.EventID).Scan(&eventCount); err != nil {
		t.Fatalf("count domain_events: %v", err)
	}
	if eventCount != 1 {
		t.Errorf("domain_events rows for committed event = %d, want 1 (state+events must commit atomically)", eventCount)
	}
}

func TestExecutionRepository_ClaimDueIntents_NoDuplicateUnderConcurrency(t *testing.T) {
	pool := requirePool(t)
	wfRepo := NewWorkflowRepository(pool)
	execRepo := NewExecutionRepository(pool)
	ctx := context.Background()
	orgID := testOrgID()

	w := newTestWorkflow(orgID)
	if err := wfRepo.CreateWorkflow(ctx, w, []event.DomainEvent{newTestEvent(w.OrganizationID, w.WorkflowID, 1)}, uuid.New()); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	intent := &workflow.ExecutionIntent{
		IntentID: uuid.New(), OrganizationID: orgID, WorkflowID: w.WorkflowID, WorkflowVersion: 2,
		CapabilityDefinitionID: uuid.New(), CapabilityDefinitionVersion: 1, GovernanceDecisionID: uuid.New(),
		IdempotencyKey: uuid.New().String(),
		// Backdated, not exactly now: ClaimDueIntents compares against the
		// DB server's clock, and this test's assertion (claimed exactly
		// once) shouldn't be sensitive to sub-second client/server clock
		// skew — that's a real but separate operational concern, not what
		// this test is checking.
		DueAt:  time.Now().UTC().Add(-time.Second),
		Inputs: map[string]any{},
	}
	next := *w
	next.Version = 2
	next.State = workflow.StateReady
	if err := wfRepo.CommitTransition(ctx, &next, 1, []event.DomainEvent{newTestEvent(orgID, w.WorkflowID, 2)}, uuid.New(), intent, nil, nil, nil, false); err != nil {
		t.Fatalf("CommitTransition with intent: %v", err)
	}

	var wg sync.WaitGroup
	claimed := make([][]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			claims, err := execRepo.ClaimDueIntents(ctx, orgID, 10, time.Minute, "worker")
			if err != nil {
				t.Errorf("ClaimDueIntents: %v", err)
				return
			}
			for _, c := range claims {
				claimed[i] = append(claimed[i], c.Attempt.AttemptID)
			}
		}(i)
	}
	wg.Wait()

	total := 0
	sawIntent := false
	for _, ids := range claimed {
		total += len(ids)
		for range ids {
			if sawIntent {
				t.Fatal("intent claimed more than once concurrently — FOR UPDATE SKIP LOCKED not preventing duplicate claims")
			}
			sawIntent = true
		}
	}
	if total != 1 {
		t.Fatalf("total claims across 5 concurrent sweeps = %d, want exactly 1", total)
	}
}

func TestOutboxRepository_LoadMarkPublished_RoundTrips(t *testing.T) {
	pool := requirePool(t)
	wfRepo := NewWorkflowRepository(pool)
	outbox := NewOutboxRepository(pool)
	ctx := context.Background()
	orgID := testOrgID()

	w := newTestWorkflow(orgID)
	evt := newTestEvent(orgID, w.WorkflowID, 1)
	if err := wfRepo.CreateWorkflow(ctx, w, []event.DomainEvent{evt}, uuid.New()); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	unpublished, err := outbox.LoadUnpublished(ctx, orgID, 100)
	if err != nil {
		t.Fatalf("LoadUnpublished: %v", err)
	}
	found := false
	for _, e := range unpublished {
		if e.EventID == evt.EventID {
			found = true
			if e.EventType != evt.EventType || e.SubjectID != evt.SubjectID {
				t.Errorf("loaded event = %+v, want it to round-trip newTestEvent's fields", e)
			}
		}
	}
	if !found {
		t.Fatalf("CreateWorkflow's event %s not found among unpublished outbox rows", evt.EventID)
	}

	if err := outbox.MarkPublished(ctx, []uuid.UUID{evt.EventID}); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}

	afterPublish, err := outbox.LoadUnpublished(ctx, orgID, 100)
	if err != nil {
		t.Fatalf("LoadUnpublished after MarkPublished: %v", err)
	}
	for _, e := range afterPublish {
		if e.EventID == evt.EventID {
			t.Fatalf("event %s still returned by LoadUnpublished after MarkPublished", evt.EventID)
		}
	}
}

func TestOutboxRepository_MarkPublishFailed_LeavesRowUnpublished(t *testing.T) {
	pool := requirePool(t)
	wfRepo := NewWorkflowRepository(pool)
	outbox := NewOutboxRepository(pool)
	ctx := context.Background()
	orgID := testOrgID()

	w := newTestWorkflow(orgID)
	evt := newTestEvent(orgID, w.WorkflowID, 1)
	if err := wfRepo.CreateWorkflow(ctx, w, []event.DomainEvent{evt}, uuid.New()); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}

	if err := outbox.MarkPublishFailed(ctx, []uuid.UUID{evt.EventID}, "realtime unreachable"); err != nil {
		t.Fatalf("MarkPublishFailed: %v", err)
	}

	unpublished, err := outbox.LoadUnpublished(ctx, orgID, 100)
	if err != nil {
		t.Fatalf("LoadUnpublished: %v", err)
	}
	found := false
	for _, e := range unpublished {
		if e.EventID == evt.EventID {
			found = true
		}
	}
	if !found {
		t.Fatalf("event %s should still be unpublished after MarkPublishFailed, so a later Sweep retries it", evt.EventID)
	}

	var attempts int
	var lastErr *string
	if err := pool.pool.QueryRow(ctx, `SELECT publish_attempts, last_error FROM event_outbox WHERE event_id = $1`, evt.EventID).Scan(&attempts, &lastErr); err != nil {
		t.Fatalf("query event_outbox: %v", err)
	}
	if attempts != 1 {
		t.Errorf("publish_attempts = %d, want 1", attempts)
	}
	if lastErr == nil || *lastErr != "realtime unreachable" {
		t.Errorf("last_error = %v, want \"realtime unreachable\"", lastErr)
	}
}

func TestRealtimePublisher_Publish_WorkflowEventSucceeds(t *testing.T) {
	pool := requirePool(t)
	publisher := NewRealtimePublisher(pool)
	ctx := context.Background()

	wfID := uuid.New()
	evt := newTestEvent(testOrgID(), wfID, 1)
	evt.WorkflowID = &wfID

	// This is the one real risk in switching from an HTTP call to a SQL
	// call: does the DATABASE_URL role have EXECUTE on realtime.send()? A
	// permission-denied error here means that grant is missing on the
	// Supabase project, not a bug in this code.
	if err := publisher.Publish(ctx, []event.DomainEvent{evt}); err != nil {
		t.Fatalf("Publish: %v (does the DATABASE_URL role have EXECUTE on realtime.send?)", err)
	}
}

func TestRealtimePublisher_Publish_NoWorkflowEventIsNoop(t *testing.T) {
	pool := requirePool(t)
	publisher := NewRealtimePublisher(pool)
	ctx := context.Background()

	evt := newTestEvent(testOrgID(), uuid.New(), 1)
	evt.WorkflowID = nil

	if err := publisher.Publish(ctx, []event.DomainEvent{evt}); err != nil {
		t.Fatalf("Publish with no WorkflowID should be a no-op, got error: %v", err)
	}
}
