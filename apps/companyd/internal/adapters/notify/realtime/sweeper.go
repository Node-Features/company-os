package realtime

import (
	"context"
	"log"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// Sweeper is the read side of event_outbox: it discovers not-yet-published
// rows and drives them through Publisher, marking each published (or
// recording a failure for retry next Sweep) — the "outbox-equivalent
// mechanism" docs/architecture/events.md's persistence step 3 requires,
// unimplemented until Phase 1 Slice 8. Shaped after
// internal/runtime.Runtime's own Start/Sweep poll loop for the same reason
// that one exists: cheap, always-correct-eventually, no external scheduler.
type Sweeper struct {
	Outbox       ports.OutboxRepository
	Publisher    ports.Publisher
	OrgID        uuid.UUID
	PollInterval time.Duration
	BatchSize    int
}

// Start runs the poll loop until ctx is cancelled. There is no wake-up fast
// path (unlike Runtime.Wakeup) — PollInterval alone directly bounds
// publish latency, and is kept short (around 1s) since the query it drives
// is a cheap, indexed read.
func (s *Sweeper) Start(ctx context.Context) {
	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Sweep(ctx)
		}
	}
}

// Sweep runs exactly one load-publish-mark pass.
func (s *Sweeper) Sweep(ctx context.Context) {
	events, err := s.Outbox.LoadUnpublished(ctx, s.OrgID, s.BatchSize)
	if err != nil {
		log.Printf("realtime: load unpublished events: %v", err)
		return
	}
	if len(events) == 0 {
		return
	}

	ids := make([]uuid.UUID, len(events))
	for i, e := range events {
		ids[i] = e.EventID
	}

	if err := s.Publisher.Publish(ctx, events); err != nil {
		log.Printf("realtime: publish %d event(s): %v", len(events), err)
		if err := s.Outbox.MarkPublishFailed(ctx, ids, err.Error()); err != nil {
			log.Printf("realtime: mark publish failed: %v", err)
		}
		return
	}

	if err := s.Outbox.MarkPublished(ctx, ids); err != nil {
		log.Printf("realtime: mark published: %v", err)
	}
}
