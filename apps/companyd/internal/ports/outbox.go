// OutboxRepository port for reading and marking event_outbox rows. See
// docs/architecture/events.md and internal/ports/publisher.go.
package ports

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
	"github.com/google/uuid"
)

// OutboxRepository is the read/ack side of the outbox
// internal/adapters/persistence/supabase/workflow_repo.go already writes:
// every DomainEvent committed with a Workflow state change gets one
// event_outbox row, atomically, in the same transaction. Nothing consumed
// those rows before Phase 1 Slice 8 — this port is what internal/adapters/
// notify/realtime.Sweeper uses to do so.
type OutboxRepository interface {
	// LoadUnpublished returns up to limit not-yet-published events for
	// orgID, oldest enqueued first. It is a plain read, not a claim — two
	// concurrent callers can return the same rows. That's acceptable, not a
	// bug: Publisher's contract (internal/ports/publisher.go) is
	// at-least-once, and a duplicate broadcast is exactly what "any
	// consumer must deduplicate by EventID" already expects. Today there is
	// only ever one Sweeper (ADR-0004's single-process topology).
	LoadUnpublished(ctx context.Context, orgID uuid.UUID, limit int) ([]event.DomainEvent, error)

	// MarkPublished records published_at=now() for every given event.
	MarkPublished(ctx context.Context, eventIDs []uuid.UUID) error

	// MarkPublishFailed increments publish_attempts and records lastErr for
	// every given event, leaving published_at NULL so the next Sweep
	// retries it — Publisher's contract is at-least-once, not
	// all-or-nothing, so a partial batch failure is expected and safe.
	MarkPublishFailed(ctx context.Context, eventIDs []uuid.UUID, lastErr string) error
}
