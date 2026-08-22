// ChangeNotifier port for Runtime to ping web's per-Workflow realtime
// channel directly, for state changes that don't go through the
// DomainEvent/event_outbox pipeline. See docs/domain/execution.md.
package ports

import (
	"context"

	"github.com/google/uuid"
)

// ChangeNotifier lets a caller push the same "something changed, go
// refetch" broadcast internal/adapters/persistence/supabase.RealtimePublisher
// sends for WorkflowEvents, but for ExecutionIntent/ExecutionAttempt
// transitions — CLAIMED, DISPATCHED, terminal — which are plain
// compare-and-write updates, not DomainEvents, and so never pass through
// event_outbox. Contract mirrors Publisher: best-effort, at-least-once,
// a recoverable hint never required for correctness (the receiver always
// refetches authoritative state; a slow reconciliation poll is the
// fallback for a missed or failed notification).
type ChangeNotifier interface {
	NotifyWorkflowChanged(ctx context.Context, workflowID uuid.UUID) error
}
