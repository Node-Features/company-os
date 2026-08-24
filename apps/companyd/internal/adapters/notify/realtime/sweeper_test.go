package realtime

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/adapters/persistence/supabase"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

// requireRealSweeper builds a Sweeper against the real event_outbox schema
// — internal/adapters/persistence/supabase/integration_test.go's
// TestOutboxRepository_LoadMarkPublished_RoundTrips and
// TestOutboxRepository_MarkPublishFailed_LeavesRowUnpublished already prove
// the repository layer's persistence is correct; this package's tests
// exist to prove Sweeper's own orchestration on top of it — the part with
// no coverage at all before this file (docs/testing/concurrency-guarantees.md
// scenario 10's durable half).
func requireRealSweeper(t *testing.T, publisher *fakePublisher) (*Sweeper, *supabase.WorkflowRepository, *supabase.OutboxRepository, uuid.UUID) {
	t.Helper()
	_ = godotenv.Load("../../../../.env")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping realtime integration test")
	}
	pool, err := supabase.Connect(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	orgID := uuid.New() // fresh per test — LoadUnpublished is organization-scoped, same isolation rationale as every other real-DB test in this repo
	outbox := supabase.NewOutboxRepository(pool)
	sweeper := &Sweeper{Outbox: outbox, Publisher: publisher, OrgID: orgID, BatchSize: 100}
	return sweeper, supabase.NewWorkflowRepository(pool), outbox, orgID
}

// fakePublisher is a test double for ports.Publisher — the one genuinely
// external dependency Sweeper has (Supabase Realtime over the network).
// failNext, if >0, fails that many calls (decrementing per call) before
// succeeding — enough to prove a real retry-after-failure sequence without
// a real network needing to actually misbehave on command.
type fakePublisher struct {
	mu       sync.Mutex
	calls    int
	failNext int
}

var errPublishUnavailable = errors.New("fakePublisher: simulated realtime outage")

func (p *fakePublisher) Publish(_ context.Context, _ []event.DomainEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	if p.failNext > 0 {
		p.failNext--
		return errPublishUnavailable
	}
	return nil
}

func (p *fakePublisher) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func seedUnpublishedEvent(t *testing.T, wfRepo *supabase.WorkflowRepository, orgID uuid.UUID) event.DomainEvent {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	w := &workflow.Workflow{
		OrganizationID: orgID, WorkflowID: uuid.New(), Version: 1,
		DefinitionID: uuid.New(), DefinitionVersion: 1, ObjectiveID: uuid.New(),
		State: workflow.StatePlanned, InitiatingPrincipalID: uuid.New(), CorrelationID: uuid.New(),
		Inputs: map[string]any{"prompt": "sweeper integration test"}, CreatedAt: now, UpdatedAt: now,
	}
	evt := event.DomainEvent{
		EventID: uuid.New(), OrganizationID: orgID, EventType: event.TypeWorkflowCreated,
		SchemaVersion: 1, SubjectType: "Workflow", SubjectID: w.WorkflowID, SubjectVersion: 1,
		OccurredAt: now, CorrelationID: uuid.New(),
	}
	if err := wfRepo.CreateWorkflow(ctx, w, []event.DomainEvent{evt}, uuid.New()); err != nil {
		t.Fatalf("seed CreateWorkflow: %v", err)
	}
	return evt
}

// TestIntegration_Sweeper_PublishFailure_RetriedOnNextSweep is scenario 10's
// durable half: a DB commit that succeeds while notification fails must
// not lose the event — it must still be delivered, at-least-once, once the
// notification path recovers. One Sweep with a failing Publisher leaves
// the event unpublished (proven directly against event_outbox, not just
// Sweeper's own bookkeeping); a second Sweep — standing in for the next
// poll tick, per Sweeper.Start's loop — with the same Publisher now
// succeeding delivers it.
func TestIntegration_Sweeper_PublishFailure_RetriedOnNextSweep(t *testing.T) {
	publisher := &fakePublisher{failNext: 1}
	sweeper, wfRepo, outbox, orgID := requireRealSweeper(t, publisher)
	ctx := context.Background()
	evt := seedUnpublishedEvent(t, wfRepo, orgID)

	sweeper.Sweep(ctx) // Publish fails (failNext consumed)

	unpublished, err := outbox.LoadUnpublished(ctx, orgID, 100)
	if err != nil {
		t.Fatalf("LoadUnpublished after failed sweep: %v", err)
	}
	found := false
	for _, e := range unpublished {
		if e.EventID == evt.EventID {
			found = true
		}
	}
	if !found {
		t.Fatalf("event %s must still be unpublished after a failed Sweep, so the next one retries it", evt.EventID)
	}
	if publisher.callCount() != 1 {
		t.Fatalf("publisher called %d times after one sweep, want 1", publisher.callCount())
	}

	sweeper.Sweep(ctx) // Publish now succeeds

	if publisher.callCount() != 2 {
		t.Fatalf("publisher called %d times after two sweeps, want 2 (the second sweep must retry the same event)", publisher.callCount())
	}
	afterRetry, err := outbox.LoadUnpublished(ctx, orgID, 100)
	if err != nil {
		t.Fatalf("LoadUnpublished after retried sweep: %v", err)
	}
	for _, e := range afterRetry {
		if e.EventID == evt.EventID {
			t.Fatalf("event %s still unpublished after a successful retry sweep", evt.EventID)
		}
	}
}

// TestIntegration_Sweeper_PermanentPublishFailure_DoesNotLoopForever proves
// Sweep's failure path is bounded per call: a Publisher that always fails
// still leaves Sweep returning promptly (one Publish call per Sweep, not an
// internal retry loop) — the poll interval, not an in-process spin, is what
// paces retries. Without this, a genuinely down realtime endpoint could
// pin a goroutine spinning instead of degrading to "delayed," which is the
// documented, accepted degradation (docs/architecture/events.md).
func TestIntegration_Sweeper_PermanentPublishFailure_DoesNotLoopForever(t *testing.T) {
	publisher := &fakePublisher{failNext: 1000000}
	sweeper, wfRepo, _, orgID := requireRealSweeper(t, publisher)
	ctx := context.Background()
	seedUnpublishedEvent(t, wfRepo, orgID)

	sweeper.Sweep(ctx)

	if got := publisher.callCount(); got != 1 {
		t.Fatalf("publisher called %d times in one Sweep, want exactly 1 (Sweep must not internally retry a failing publish)", got)
	}
}
