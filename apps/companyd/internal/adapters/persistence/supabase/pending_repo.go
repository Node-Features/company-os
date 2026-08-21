package supabase

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/google/uuid"
)

// PendingCommandRepository implements ports.PendingCommandRepository.
// Unexercised by this slice's always-ALLOW policy, implemented per
// application.md's "never bypassed."
type PendingCommandRepository struct{ p *Pool }

func NewPendingCommandRepository(p *Pool) *PendingCommandRepository { return &PendingCommandRepository{p: p} }

func (r *PendingCommandRepository) CreatePendingApproval(ctx context.Context, pc *command.PendingCommand, decisionID uuid.UUID, appr *approval.Approval) error {
	tx, err := r.p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO pending_commands (pending_command_id, organization_id, status, command_type, proposal_digest,
		                               governed_payload, expected_workflow_id, expected_workflow_version,
		                               governance_decision_id, approval_id, request_id, idempotency_key,
		                               correlation_id, created_at, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		pc.PendingCommandID, pc.OrganizationID, pc.Status, pc.CommandType, pc.ProposalDigest, pc.GovernedPayload,
		pc.ExpectedWorkflowID, pc.ExpectedWorkflowVersion, decisionID, pc.ApprovalID, pc.RequestID, pc.IdempotencyKey,
		pc.CorrelationID, pc.CreatedAt, pc.ExpiresAt); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO approvals (approval_id, organization_id, pending_command_id, requesting_principal_id, action,
		                        resource_type, resource_id, proposal_digest, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		appr.ApprovalID, appr.OrganizationID, appr.PendingCommandID, appr.RequestingPrincipalID, appr.Action,
		appr.ResourceType, appr.ResourceID, appr.ProposalDigest, appr.Status, appr.CreatedAt); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
