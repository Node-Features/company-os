package supabase

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/execution"
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

// TestWorkflowRepository_CreateWorkflow_RollbackLeavesNoOrphanEvents proves
// scenario 9 of docs/testing/concurrency-guarantees.md: notification can
// never race a DB commit into an inconsistent state, because insertEvents
// (event_outbox included) already runs inside CreateWorkflow's own
// transaction, before tx.Commit — not as a separate step that could
// observe a domain write the transaction later abandons. Rather than
// racing two goroutines (there is nothing to race — same-transaction
// placement makes the ordering structural, not probabilistic), this
// forces the transaction to fail *after* one of its two events has already
// been inserted (but not committed) and asserts that "already inserted
// within an active transaction" is not the same as "durable": the
// still-uncommitted event, and the Workflow row itself, are gone once the
// second event's primary-key collision aborts the transaction.
func TestWorkflowRepository_CreateWorkflow_RollbackLeavesNoOrphanEvents(t *testing.T) {
	pool := requirePool(t)
	repo := NewWorkflowRepository(pool)
	ctx := context.Background()
	orgID := testOrgID()

	// An independently committed event, so a later transaction has a real,
	// already-taken event_id to collide with.
	w1 := newTestWorkflow(orgID)
	committedEvent := newTestEvent(w1.OrganizationID, w1.WorkflowID, 1)
	if err := repo.CreateWorkflow(ctx, w1, []event.DomainEvent{committedEvent}, uuid.New()); err != nil {
		t.Fatalf("seed CreateWorkflow: %v", err)
	}

	// A second, otherwise-independent Workflow whose CreateWorkflow call
	// carries two events: freshEvent inserts cleanly first, then
	// collidingEvent (committedEvent's own event_id, already PRIMARY KEY'd
	// from the seed above) aborts the transaction mid-insertEvents.
	w2 := newTestWorkflow(orgID)
	freshEvent := newTestEvent(w2.OrganizationID, w2.WorkflowID, 1)
	collidingEvent := committedEvent
	collidingEvent.SubjectID = w2.WorkflowID

	if err := repo.CreateWorkflow(ctx, w2, []event.DomainEvent{freshEvent, collidingEvent}, uuid.New()); err == nil {
		t.Fatal("CreateWorkflow with a colliding event_id succeeded, want a primary-key violation")
	}

	if _, loadErr := repo.LoadWorkflow(ctx, w2.OrganizationID, w2.WorkflowID); loadErr != ports.ErrNotFound {
		t.Fatalf("LoadWorkflow after rolled-back CreateWorkflow: err = %v, want ports.ErrNotFound (the Workflow insert must have rolled back too)", loadErr)
	}

	var eventCount, outboxCount int
	if err := pool.pool.QueryRow(ctx, `SELECT count(*) FROM domain_events WHERE event_id = $1`, freshEvent.EventID).Scan(&eventCount); err != nil {
		t.Fatalf("count domain_events: %v", err)
	}
	if err := pool.pool.QueryRow(ctx, `SELECT count(*) FROM event_outbox WHERE event_id = $1`, freshEvent.EventID).Scan(&outboxCount); err != nil {
		t.Fatalf("count event_outbox: %v", err)
	}
	if eventCount != 0 || outboxCount != 0 {
		t.Fatalf("orphan rows for the fresh event that preceded the collision: domain_events=%d event_outbox=%d, want 0/0 — insertEvents's per-event inserts must be atomic with the rest of the transaction, not committed piecemeal", eventCount, outboxCount)
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

// claimedAttemptFixture creates a Workflow with one due ExecutionIntent and
// claims it with leaseDuration (pass a negative duration to get an
// already-expired lease deterministically, with no need to sleep — the
// same backdating trick claimedIntentFixture-adjacent tests above use for
// due_at, applied here to lease_expires_at instead). Returns the single
// claimed attempt.
func claimedAttemptFixture(t *testing.T, ctx context.Context, wfRepo *WorkflowRepository, execRepo *ExecutionRepository, orgID uuid.UUID, leaseDuration time.Duration) execution.ExecutionAttempt {
	t.Helper()
	w := newTestWorkflow(orgID)
	if err := wfRepo.CreateWorkflow(ctx, w, []event.DomainEvent{newTestEvent(w.OrganizationID, w.WorkflowID, 1)}, uuid.New()); err != nil {
		t.Fatalf("CreateWorkflow: %v", err)
	}
	intent := &workflow.ExecutionIntent{
		IntentID: uuid.New(), OrganizationID: orgID, WorkflowID: w.WorkflowID, WorkflowVersion: 2,
		CapabilityDefinitionID: uuid.New(), CapabilityDefinitionVersion: 1, GovernanceDecisionID: uuid.New(),
		IdempotencyKey: uuid.New().String(),
		DueAt:          time.Now().UTC().Add(-time.Second),
		Inputs:         map[string]any{},
	}
	next := *w
	next.Version = 2
	next.State = workflow.StateReady
	if err := wfRepo.CommitTransition(ctx, &next, 1, []event.DomainEvent{newTestEvent(orgID, w.WorkflowID, 2)}, uuid.New(), intent, nil, nil, nil, false); err != nil {
		t.Fatalf("CommitTransition with intent: %v", err)
	}

	claims, err := execRepo.ClaimDueIntents(ctx, orgID, 10, leaseDuration, "worker")
	if err != nil {
		t.Fatalf("ClaimDueIntents: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("claimed %d intents, want exactly 1", len(claims))
	}
	return claims[0].Attempt
}

// TestExecutionRepository_ReclaimExpiredLeases_RecoversAbandonedAttempt is
// the core proof for invariant 7 ("lost workers must not permanently
// strand work"): an attempt whose lease has expired while still CLAIMED
// (the worker crashed, was killed, or lost its process between claiming
// and reporting back) is found, transitioned to LEASE_EXPIRED, and its
// lease_fencing_token is changed — see the next test for why that specific
// side effect matters.
func TestExecutionRepository_ReclaimExpiredLeases_RecoversAbandonedAttempt(t *testing.T) {
	pool := requirePool(t)
	wfRepo := NewWorkflowRepository(pool)
	execRepo := NewExecutionRepository(pool)
	ctx := context.Background()
	orgID := testOrgID()

	abandoned := claimedAttemptFixture(t, ctx, wfRepo, execRepo, orgID, -time.Minute)

	reclaimed, err := execRepo.ReclaimExpiredLeases(ctx, orgID, 10)
	if err != nil {
		t.Fatalf("ReclaimExpiredLeases: %v", err)
	}
	if len(reclaimed) != 1 {
		t.Fatalf("reclaimed %d attempts, want exactly 1", len(reclaimed))
	}
	got := reclaimed[0]
	if got.Attempt.AttemptID != abandoned.AttemptID {
		t.Fatalf("reclaimed attempt %s, want %s", got.Attempt.AttemptID, abandoned.AttemptID)
	}
	if got.Attempt.Status != execution.StatusLeaseExpired {
		t.Fatalf("reclaimed attempt status = %s, want LEASE_EXPIRED", got.Attempt.Status)
	}
	if got.Intent.WorkflowID != abandoned.WorkflowID || got.Intent.IdempotencyKey == "" {
		t.Fatalf("reclaimed ClaimedExecution's Intent not populated correctly: %+v", got.Intent)
	}

	// A second reclaim pass must find nothing — the attempt is now
	// genuinely terminal (execution.StatusLeaseExpired.IsTerminal()),
	// excluded from ReclaimExpiredLeases' own status filter, and must never
	// be reclaimed twice.
	again, err := execRepo.ReclaimExpiredLeases(ctx, orgID, 10)
	if err != nil {
		t.Fatalf("second ReclaimExpiredLeases: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second reclaim found %d attempts, want 0 (already-reclaimed attempt must not be reclaimed again)", len(again))
	}
}

// TestExecutionRepository_ReclaimExpiredLeases_BumpsFencingTokenSoStaleWorkerFailsClosed
// is the direct proof of the fencing-token safety argument documented on
// ports.ExecutionRepository.ReclaimExpiredLeases: once an attempt is
// reclaimed, the *original* worker's lease_fencing_token is no longer
// valid, so if that worker eventually does call back (it was not actually
// dead, just slow, or the process is a zombie that never got killed), its
// report is rejected — never silently accepted as if it still held the
// lease. This is what makes reclaim safe under invariant 8 even though Go
// cannot forcibly stop the original goroutine.
func TestExecutionRepository_ReclaimExpiredLeases_BumpsFencingTokenSoStaleWorkerFailsClosed(t *testing.T) {
	pool := requirePool(t)
	wfRepo := NewWorkflowRepository(pool)
	execRepo := NewExecutionRepository(pool)
	ctx := context.Background()
	orgID := testOrgID()

	abandoned := claimedAttemptFixture(t, ctx, wfRepo, execRepo, orgID, -time.Minute)
	originalToken := *abandoned.LeaseFencingToken

	if _, err := execRepo.ReclaimExpiredLeases(ctx, orgID, 10); err != nil {
		t.Fatalf("ReclaimExpiredLeases: %v", err)
	}

	// The zombie worker, unaware it has been reclaimed, tries to report in
	// using its original (now-stale) fencing token.
	err := execRepo.RecordDispatched(ctx, abandoned.AttemptID, originalToken, "zombie-provider-run")
	if !errors.Is(err, ports.ErrConflict) {
		t.Fatalf("RecordDispatched with pre-reclaim fencing token error = %v, want ports.ErrConflict", err)
	}
	err = execRepo.RecordTerminal(ctx, abandoned.AttemptID, originalToken, execution.StatusSucceeded, nil)
	if !errors.Is(err, ports.ErrConflict) {
		t.Fatalf("RecordTerminal with pre-reclaim fencing token error = %v, want ports.ErrConflict", err)
	}

	// Confirm the row genuinely wasn't touched by either stale call.
	var status string
	if err := pool.pool.QueryRow(ctx, `SELECT status FROM execution_attempts WHERE attempt_id=$1`, abandoned.AttemptID).Scan(&status); err != nil {
		t.Fatalf("query attempt status: %v", err)
	}
	if status != string(execution.StatusLeaseExpired) {
		t.Fatalf("attempt status = %s, want LEASE_EXPIRED (a stale-token write must never change it)", status)
	}
}

// TestExecutionRepository_ReclaimExpiredLeases_LeavesNonExpiredAlone proves
// ReclaimExpiredLeases doesn't reclaim work that's merely in progress —
// only genuinely lease-expired attempts.
func TestExecutionRepository_ReclaimExpiredLeases_LeavesNonExpiredAlone(t *testing.T) {
	pool := requirePool(t)
	wfRepo := NewWorkflowRepository(pool)
	execRepo := NewExecutionRepository(pool)
	ctx := context.Background()
	orgID := testOrgID()

	stillActive := claimedAttemptFixture(t, ctx, wfRepo, execRepo, orgID, time.Hour)

	reclaimed, err := execRepo.ReclaimExpiredLeases(ctx, orgID, 10)
	if err != nil {
		t.Fatalf("ReclaimExpiredLeases: %v", err)
	}
	for _, c := range reclaimed {
		if c.Attempt.AttemptID == stillActive.AttemptID {
			t.Fatalf("reclaimed an attempt with a lease still an hour from expiry")
		}
	}

	// The still-active attempt's own fencing token must be untouched too —
	// reclaim must not have side-effected a row it correctly excluded.
	var token int64
	if err := pool.pool.QueryRow(ctx, `SELECT lease_fencing_token FROM execution_attempts WHERE attempt_id=$1`, stillActive.AttemptID).Scan(&token); err != nil {
		t.Fatalf("query fencing token: %v", err)
	}
	if token != *stillActive.LeaseFencingToken {
		t.Fatalf("fencing token changed from %d to %d for an attempt that was never reclaimed", *stillActive.LeaseFencingToken, token)
	}
}

// TestExecutionRepository_ReclaimExpiredLeases_ConcurrentReclaimersNoDuplicate
// mirrors TestExecutionRepository_ClaimDueIntents_NoDuplicateUnderConcurrency
// exactly, for the new claim path: FOR UPDATE SKIP LOCKED must make
// concurrent reclaimers (e.g. two companyd processes, or overlapping Sweep
// cycles) safe against each other the same way it already does for initial
// claims.
func TestExecutionRepository_ReclaimExpiredLeases_ConcurrentReclaimersNoDuplicate(t *testing.T) {
	pool := requirePool(t)
	wfRepo := NewWorkflowRepository(pool)
	execRepo := NewExecutionRepository(pool)
	ctx := context.Background()
	orgID := testOrgID()

	abandoned := claimedAttemptFixture(t, ctx, wfRepo, execRepo, orgID, -time.Minute)

	var wg sync.WaitGroup
	reclaimedBy := make([][]uuid.UUID, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			reclaimed, err := execRepo.ReclaimExpiredLeases(ctx, orgID, 10)
			if err != nil {
				t.Errorf("ReclaimExpiredLeases: %v", err)
				return
			}
			for _, c := range reclaimed {
				reclaimedBy[i] = append(reclaimedBy[i], c.Attempt.AttemptID)
			}
		}(i)
	}
	wg.Wait()

	total := 0
	sawIt := false
	for _, ids := range reclaimedBy {
		total += len(ids)
		for _, id := range ids {
			if id == abandoned.AttemptID {
				if sawIt {
					t.Fatal("attempt reclaimed more than once concurrently — FOR UPDATE SKIP LOCKED not preventing duplicate reclaim")
				}
				sawIt = true
			}
		}
	}
	if total != 1 {
		t.Fatalf("total reclaims across 5 concurrent sweeps = %d, want exactly 1", total)
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
