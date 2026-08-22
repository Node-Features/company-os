// Publisher port for post-commit event notification. See
// docs/architecture/events.md.
package ports

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
)

// Publisher is the post-commit notification port docs/architecture/events.md
// calls an "outbox-equivalent mechanism." Nothing implements it yet:
// internal/adapters/persistence/supabase/workflow_repo.go writes rows into
// event_outbox inside the same transaction as the Workflow state change
// (the write side, already real), but no code anywhere reads that table,
// calls Publish, or ever sets published_at — there is no stub, no-op or
// otherwise, only an unread table today. internal/adapters/notify/realtime
// is the intended home for a real implementation, deferred to ROADMAP.md
// Phase 1 Slice 8.
//
// This is not the "EventBus" that was asked for: docs/architecture/events.md
// and docs/references/feature-provenance.md's JARVIS row both explicitly
// reject in-process bus delivery or live publish-subscribe as an
// authoritative organizational record. There is no Subscribe or Unsubscribe
// method here because nothing in companyd subscribes at the Go level — the
// one planned consumer (Phase 1 Slice 8's realtime activity layer) is
// `web`'s browser client connecting directly to Supabase Realtime, entirely
// outside this process and this interface.
//
// Contract: Publish must only be called with events already committed
// inside the same transaction as the Workflow state change that produced
// them (events.md: "Publication begins only from a persisted Event") — it
// must never be called with an event that might still be rolled back.
// Publish does NOT guarantee delivery ordering across events, does NOT
// guarantee exactly-once delivery — events.md: "Delivery is treated as
// at-least-once, delayed, and potentially out of order," so any consumer
// must deduplicate by EventID itself — and does NOT guarantee synchronous
// completion before returning; a failed or partial Publish must be safely
// retryable from the outbox's unpublished rows without re-deriving or
// duplicating the source Event. A returned error means at least one event
// in the batch was not confirmed published — it does NOT mean none were;
// callers must not assume all-or-nothing batch semantics.
type Publisher interface {
	Publish(ctx context.Context, events []event.DomainEvent) error
}
