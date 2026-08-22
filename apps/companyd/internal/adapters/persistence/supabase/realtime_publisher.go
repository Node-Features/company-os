package supabase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// RealtimePublisher implements ports.Publisher using Supabase's "Broadcast
// from Database" realtime.send() SQL function, called over the same
// DATABASE_URL pool every other repository in this package uses — not an
// HTTP call to Supabase's Realtime REST API, and no separate service-role
// credential. web's browser client (holding only the publishable key)
// subscribes to the resulting broadcast directly; see
// internal/ports/publisher.go and internal/adapters/notify/realtime.
type RealtimePublisher struct{ p *Pool }

func NewRealtimePublisher(p *Pool) *RealtimePublisher { return &RealtimePublisher{p: p} }

// broadcastTopic is the per-Workflow channel web's browser client
// subscribes to. Scoped to one Workflow, matching Phase 1 Slice 8's scope
// note: "start with one Workflow's live status."
func broadcastTopic(workflowID uuid.UUID) string {
	return "workflow:" + workflowID.String()
}

// broadcastEventName is the one event name web's WorkflowTrigger listens
// for on every workflow:<id> topic.
const broadcastEventName = "workflow_changed"

// hint is the minimal, non-authoritative payload a browser receives — a
// strict subset of event.DomainEvent's fields, per events.md's "Publishers
// and consumers use the canonical Event minimum contract without extending
// its meaning locally." The receiver treats this as "something changed, go
// refetch GET /v1/workflows/{id}," never as trustworthy state, per
// events.md's "notification is a recoverable hint, not a dependency."
type hint struct {
	EventID        string `json:"eventId"`
	EventType      string `json:"eventType"`
	WorkflowID     string `json:"workflowId"`
	SubjectVersion int64  `json:"subjectVersion"`
	OccurredAt     string `json:"occurredAt"`
}

// Publish satisfies ports.Publisher. Only events carrying a WorkflowID are
// broadcast — every first-slice WorkflowEvent type does
// (internal/domain/event/types.go). An event with no WorkflowID is skipped,
// not an error: there is no topic to publish it on yet. realtime.send's
// last argument (private=false) makes this a public channel — no
// realtime.messages RLS policy is needed for web's anon-key client to
// receive it, mirroring the zero-RLS-policy posture every table in this
// schema already has.
func (r *RealtimePublisher) Publish(ctx context.Context, events []event.DomainEvent) error {
	for _, e := range events {
		if e.WorkflowID == nil {
			continue
		}
		payload, err := json.Marshal(hint{
			EventID:        e.EventID.String(),
			EventType:      e.EventType,
			WorkflowID:     e.WorkflowID.String(),
			SubjectVersion: e.SubjectVersion,
			OccurredAt:     e.OccurredAt.Format("2006-01-02T15:04:05Z07:00"),
		})
		if err != nil {
			return err
		}
		if err := r.send(ctx, *e.WorkflowID, payload); err != nil {
			return err
		}
	}
	return nil
}

// notifyHint is NotifyWorkflowChanged's payload — deliberately not a
// event.DomainEvent-shaped hint (no eventId/subjectVersion) since a Runtime
// ExecutionUnit transition (CLAIMED/DISPATCHED/terminal) never produces one.
// web treats every payload on this channel identically either way: a reason
// to refetch GET /v1/workflows/{id}, never trustworthy state on its own.
type notifyHint struct {
	EventType  string `json:"eventType"`
	WorkflowID string `json:"workflowId"`
	OccurredAt string `json:"occurredAt"`
}

// NotifyWorkflowChanged satisfies ports.ChangeNotifier: a best-effort ping
// on the same workflow:<id> broadcast topic Publish uses, for a Runtime
// ExecutionUnit transition that has no DomainEvent/event_outbox row to
// publish from.
func (r *RealtimePublisher) NotifyWorkflowChanged(ctx context.Context, workflowID uuid.UUID) error {
	payload, err := json.Marshal(notifyHint{
		EventType:  "EXECUTION_UNIT_CHANGED",
		WorkflowID: workflowID.String(),
		OccurredAt: time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
	})
	if err != nil {
		return err
	}
	return r.send(ctx, workflowID, payload)
}

func (r *RealtimePublisher) send(ctx context.Context, workflowID uuid.UUID, payload []byte) error {
	_, err := r.p.pool.Exec(ctx, `SELECT realtime.send($1::jsonb, $2, $3, false)`,
		payload, broadcastEventName, broadcastTopic(workflowID))
	return err
}

var _ ports.Publisher = (*RealtimePublisher)(nil)
var _ ports.ChangeNotifier = (*RealtimePublisher)(nil)
