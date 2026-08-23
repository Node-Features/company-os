package supabase

import (
	"context"
	"errors"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/objective"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ObjectiveRepository implements ports.ObjectiveRepository.
type ObjectiveRepository struct{ p *Pool }

func NewObjectiveRepository(p *Pool) *ObjectiveRepository {
	return &ObjectiveRepository{p: p}
}

func (r *ObjectiveRepository) CreateObjective(ctx context.Context, o *objective.Objective, closePendingCommand *uuid.UUID, consumeApproval *uuid.UUID) error {
	tx, err := r.p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO objectives (objective_id, organization_id, title, intent, status, owner_principal_id,
		                         source_type, source_id, governance_decision_id, approval_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		o.ObjectiveID, o.OrganizationID, o.Title, o.Intent, o.Status, o.OwnerPrincipalID,
		o.SourceType, o.SourceID, o.GovernanceDecisionID, o.ApprovalID, o.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return ports.ErrConflict
		}
		return err
	}

	// closePendingCommand/consumeApproval mirror
	// ports.AuthoritativeStateRepository.CommitTransition's parameters of
	// the same name — set only when called from the REQUIRE_APPROVAL
	// resume path, closed atomically with the Objective insert.
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

	return tx.Commit(ctx)
}

func (r *ObjectiveRepository) GetObjective(ctx context.Context, organizationID, objectiveID uuid.UUID) (objective.Objective, error) {
	return r.get(ctx, `
		SELECT objective_id, organization_id, title, intent, status, owner_principal_id,
		       source_type, source_id, governance_decision_id, approval_id, created_at
		FROM objectives WHERE organization_id=$1 AND objective_id=$2`,
		organizationID, objectiveID)
}

func (r *ObjectiveRepository) GetObjectiveBySource(ctx context.Context, organizationID uuid.UUID, sourceType objective.SourceType, sourceID uuid.UUID) (objective.Objective, error) {
	return r.get(ctx, `
		SELECT objective_id, organization_id, title, intent, status, owner_principal_id,
		       source_type, source_id, governance_decision_id, approval_id, created_at
		FROM objectives WHERE organization_id=$1 AND source_type=$2 AND source_id=$3`,
		organizationID, sourceType, sourceID)
}

func (r *ObjectiveRepository) get(ctx context.Context, query string, args ...any) (objective.Objective, error) {
	var o objective.Objective
	var status, sourceType string
	err := r.p.pool.QueryRow(ctx, query, args...).Scan(
		&o.ObjectiveID, &o.OrganizationID, &o.Title, &o.Intent, &status, &o.OwnerPrincipalID,
		&sourceType, &o.SourceID, &o.GovernanceDecisionID, &o.ApprovalID, &o.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return objective.Objective{}, ports.ErrNotFound
	}
	if err != nil {
		return objective.Objective{}, err
	}
	o.Status = objective.Status(status)
	o.SourceType = objective.SourceType(sourceType)
	return o, nil
}

var _ ports.ObjectiveRepository = (*ObjectiveRepository)(nil)
