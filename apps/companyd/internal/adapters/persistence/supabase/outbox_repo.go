package supabase

import (
	"context"
	"encoding/json"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
	"github.com/google/uuid"
)

// OutboxRepository implements ports.OutboxRepository against event_outbox
// joined with domain_events. See docs/architecture/events.md.
type OutboxRepository struct{ p *Pool }

func NewOutboxRepository(p *Pool) *OutboxRepository { return &OutboxRepository{p: p} }

func (r *OutboxRepository) LoadUnpublished(ctx context.Context, orgID uuid.UUID, limit int) ([]event.DomainEvent, error) {
	rows, err := r.p.pool.Query(ctx, `
		SELECT de.event_id, de.organization_id, de.event_type, de.schema_version, de.subject_type,
		       de.subject_id, de.subject_version, de.occurred_at, de.correlation_id, de.causation_id,
		       de.workflow_id, de.objective_id, de.principal_id, de.payload
		FROM event_outbox eo
		JOIN domain_events de ON de.event_id = eo.event_id
		WHERE eo.organization_id = $1 AND eo.published_at IS NULL
		ORDER BY eo.enqueued_at
		LIMIT $2`,
		orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []event.DomainEvent
	for rows.Next() {
		var e event.DomainEvent
		var payload []byte
		if err := rows.Scan(&e.EventID, &e.OrganizationID, &e.EventType, &e.SchemaVersion, &e.SubjectType,
			&e.SubjectID, &e.SubjectVersion, &e.OccurredAt, &e.CorrelationID, &e.CausationID,
			&e.WorkflowID, &e.ObjectiveID, &e.PrincipalID, &payload); err != nil {
			return nil, err
		}
		if len(payload) > 0 {
			_ = json.Unmarshal(payload, &e.Payload)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, eventIDs []uuid.UUID) error {
	if len(eventIDs) == 0 {
		return nil
	}
	_, err := r.p.pool.Exec(ctx, `UPDATE event_outbox SET published_at = now() WHERE event_id = ANY($1)`, eventIDs)
	return err
}

func (r *OutboxRepository) MarkPublishFailed(ctx context.Context, eventIDs []uuid.UUID, lastErr string) error {
	if len(eventIDs) == 0 {
		return nil
	}
	_, err := r.p.pool.Exec(ctx, `
		UPDATE event_outbox SET publish_attempts = publish_attempts + 1, last_error = $2
		WHERE event_id = ANY($1)`, eventIDs, lastErr)
	return err
}
