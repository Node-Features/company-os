package supabase

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// WorkflowRepository implements ports.AuthoritativeStateRepository against
// Supabase Postgres: one Workflow per (organization_id, workflow_id),
// compare-and-write only. See docs/architecture/persistence.md.
type WorkflowRepository struct{ p *Pool }

func NewWorkflowRepository(p *Pool) *WorkflowRepository { return &WorkflowRepository{p: p} }

func (r *WorkflowRepository) LoadWorkflow(ctx context.Context, orgID, workflowID uuid.UUID) (*workflow.Workflow, error) {
	row := r.p.pool.QueryRow(ctx, `
		SELECT organization_id, workflow_id, version, definition_id, definition_version, objective_id,
		       state, initiating_principal_id, correlation_id, causation_id, inputs, wait_reason,
		       terminal_reason, created_at, updated_at
		FROM workflows WHERE organization_id = $1 AND workflow_id = $2`, orgID, workflowID)

	var w workflow.Workflow
	var inputs []byte
	if err := row.Scan(&w.OrganizationID, &w.WorkflowID, &w.Version, &w.DefinitionID, &w.DefinitionVersion,
		&w.ObjectiveID, &w.State, &w.InitiatingPrincipalID, &w.CorrelationID, &w.CausationID, &inputs,
		&w.WaitReason, &w.TerminalReason, &w.CreatedAt, &w.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	if len(inputs) > 0 {
		_ = json.Unmarshal(inputs, &w.Inputs)
	}
	return &w, nil
}

func (r *WorkflowRepository) CreateWorkflow(ctx context.Context, w *workflow.Workflow, events []event.DomainEvent, decisionID uuid.UUID) error {
	tx, err := r.p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	inputs, _ := json.Marshal(w.Inputs)
	_, err = tx.Exec(ctx, `
		INSERT INTO workflows (organization_id, workflow_id, version, definition_id, definition_version,
		                        objective_id, state, initiating_principal_id, correlation_id, causation_id,
		                        inputs, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		w.OrganizationID, w.WorkflowID, w.Version, w.DefinitionID, w.DefinitionVersion, w.ObjectiveID,
		w.State, w.InitiatingPrincipalID, w.CorrelationID, w.CausationID, inputs, w.CreatedAt, w.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ports.ErrConflict
		}
		return err
	}

	if err := insertEvents(ctx, tx, events); err != nil {
		return err
	}

	_ = decisionID // recorded via SaveGovernanceDecision by the caller before commit
	return tx.Commit(ctx)
}

func (r *WorkflowRepository) CommitTransition(
	ctx context.Context,
	w *workflow.Workflow,
	expectedVersion int64,
	events []event.DomainEvent,
	decisionID uuid.UUID,
	intent *workflow.ExecutionIntent,
	closePendingCommand *uuid.UUID,
	consumeApproval *uuid.UUID,
	decideResult *ports.ResultDecision,
	closeOutstandingIntent bool,
) error {
	tx, err := r.p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE workflows SET version=$1, state=$2, wait_reason=$3, terminal_reason=$4, updated_at=$5
		WHERE organization_id=$6 AND workflow_id=$7 AND version=$8`,
		w.Version, w.State, w.WaitReason, w.TerminalReason, w.UpdatedAt,
		w.OrganizationID, w.WorkflowID, expectedVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ports.ErrConflict
	}

	if err := insertEvents(ctx, tx, events); err != nil {
		return err
	}

	if intent != nil {
		constraints, _ := json.Marshal(intent.Constraints)
		inputs, _ := json.Marshal(intent.Inputs)
		_, err = tx.Exec(ctx, `
			INSERT INTO execution_intents (intent_id, organization_id, workflow_id, workflow_version,
			                                capability_definition_id, capability_definition_version,
			                                governance_decision_id, idempotency_key, constraints, due_at, inputs)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			intent.IntentID, intent.OrganizationID, intent.WorkflowID, intent.WorkflowVersion,
			intent.CapabilityDefinitionID, intent.CapabilityDefinitionVersion, intent.GovernanceDecisionID,
			intent.IdempotencyKey, constraints, intent.DueAt, inputs)
		if err != nil {
			return err
		}
	}

	if closePendingCommand != nil {
		if _, err := tx.Exec(ctx, `UPDATE pending_commands SET status='RESOLVED', closed_at=now() WHERE pending_command_id=$1`, *closePendingCommand); err != nil {
			return err
		}
	}
	if consumeApproval != nil {
		if _, err := tx.Exec(ctx, `UPDATE approvals SET status='CONSUMED', decided_at=now() WHERE approval_id=$1`, *consumeApproval); err != nil {
			return err
		}
	}
	if decideResult != nil {
		if _, err := tx.Exec(ctx, `UPDATE results SET accepted=$1, decided_at=now() WHERE result_id=$2`, decideResult.Accepted, decideResult.ResultID); err != nil {
			return err
		}
	}

	if closeOutstandingIntent {
		if _, err := tx.Exec(ctx, `
			UPDATE execution_intents SET status='CLOSED'
			WHERE organization_id=$1 AND workflow_id=$2 AND status='PENDING'`,
			w.OrganizationID, w.WorkflowID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *WorkflowRepository) SaveGovernanceDecision(ctx context.Context, decisionID, orgID, requestID, correlationID, principalID uuid.UUID,
	action, resourceType, resourceID, proposalDigest, trustedContextDigest, policyVersion, autonomyLevel, outcome string,
	matchedRuleID, reason *string) error {
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO governance_decisions (decision_id, organization_id, request_id, correlation_id, principal_id,
		                                   action, resource_type, resource_id, proposal_digest, trusted_context_digest,
		                                   policy_version, autonomy_level, outcome, matched_rule_id, reason)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		decisionID, orgID, requestID, correlationID, principalID, action, resourceType, resourceID,
		proposalDigest, trustedContextDigest, policyVersion, autonomyLevel, outcome, matchedRuleID, reason)
	return err
}

func (r *WorkflowRepository) IdempotencyLookup(ctx context.Context, orgID uuid.UUID, key string) (bool, string, error) {
	var outcome string
	err := r.p.pool.QueryRow(ctx, `SELECT outcome FROM idempotency_keys WHERE organization_id=$1 AND idempotency_key=$2`, orgID, key).Scan(&outcome)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, outcome, nil
}

func (r *WorkflowRepository) IdempotencyStore(ctx context.Context, orgID, requestID uuid.UUID, key, outcome string) error {
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO idempotency_keys (organization_id, idempotency_key, request_id, outcome)
		VALUES ($1,$2,$3,$4) ON CONFLICT (organization_id, idempotency_key) DO NOTHING`,
		orgID, key, requestID, outcome)
	return err
}

func insertEvents(ctx context.Context, tx pgx.Tx, events []event.DomainEvent) error {
	for _, e := range events {
		payload, _ := json.Marshal(e.Payload)
		if _, err := tx.Exec(ctx, `
			INSERT INTO domain_events (event_id, organization_id, event_type, schema_version, subject_type,
			                            subject_id, subject_version, occurred_at, correlation_id, causation_id,
			                            workflow_id, objective_id, principal_id, payload)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
			e.EventID, e.OrganizationID, e.EventType, e.SchemaVersion, e.SubjectType, e.SubjectID,
			e.SubjectVersion, e.OccurredAt, e.CorrelationID, e.CausationID, e.WorkflowID, e.ObjectiveID,
			e.PrincipalID, payload); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO event_outbox (event_id, organization_id) VALUES ($1,$2)`, e.EventID, e.OrganizationID); err != nil {
			return err
		}
	}
	return nil
}

// isUniqueViolation reports whether err is a Postgres unique_violation
// (SQLSTATE 23505), used to translate a duplicate-key CreateWorkflow insert
// into ports.ErrConflict.
func isUniqueViolation(err error) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
