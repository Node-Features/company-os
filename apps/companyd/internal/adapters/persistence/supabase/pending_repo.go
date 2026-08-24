package supabase

import (
	"context"
	"errors"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PendingCommandRepository implements ports.PendingCommandRepository.
// Unexercised by this slice's always-ALLOW policy, implemented per
// application.md's "never bypassed."
type PendingCommandRepository struct{ p *Pool }

func NewPendingCommandRepository(p *Pool) *PendingCommandRepository {
	return &PendingCommandRepository{p: p}
}

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

// ResolveApproval locks the Approval (and its linked PendingCommand) with
// SELECT ... FOR UPDATE before deciding anything, so the structural
// invariant checks below (expiry, self-approval, human decider) and the
// eventual status write are all part of one atomic transaction — no
// TOCTOU window, and a losing concurrent caller sees the same
// well-defined outcomes a solo caller would. See the ports interface doc
// comment for the exact decision order.
func (r *PendingCommandRepository) ResolveApproval(ctx context.Context, approvalID uuid.UUID, decidingPrincipal principal.Principal, approve bool, reason *string) (*command.PendingCommand, *approval.Approval, error) {
	tx, err := r.p.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	var appr approval.Approval
	err = tx.QueryRow(ctx, `
		SELECT approval_id, organization_id, pending_command_id, requesting_principal_id, action,
		       resource_type, resource_id, proposal_digest, status, decided_by_principal_id, decided_at, reason, created_at
		FROM approvals WHERE approval_id=$1 FOR UPDATE`, approvalID,
	).Scan(&appr.ApprovalID, &appr.OrganizationID, &appr.PendingCommandID, &appr.RequestingPrincipalID, &appr.Action,
		&appr.ResourceType, &appr.ResourceID, &appr.ProposalDigest, &appr.Status, &appr.DecidedByPrincipalID, &appr.DecidedAt, &appr.Reason, &appr.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ports.ErrConflict
	}
	if err != nil {
		return nil, nil, err
	}
	if appr.Status != approval.StatusPending {
		// Already decided (or otherwise not PENDING) — same
		// compare-and-write-conflict shape the prior single-UPDATE
		// implementation returned for this case.
		return nil, nil, ports.ErrConflict
	}

	var pc command.PendingCommand
	err = tx.QueryRow(ctx, `
		SELECT pending_command_id, organization_id, status, command_type, proposal_digest, governed_payload,
		       expected_workflow_id, expected_workflow_version, governance_decision_id, approval_id, request_id,
		       idempotency_key, correlation_id, created_at, expires_at, closed_at, closure_reason
		FROM pending_commands WHERE pending_command_id=$1 FOR UPDATE`, appr.PendingCommandID,
	).Scan(&pc.PendingCommandID, &pc.OrganizationID, &pc.Status, &pc.CommandType, &pc.ProposalDigest, &pc.GovernedPayload,
		&pc.ExpectedWorkflowID, &pc.ExpectedWorkflowVersion, &pc.GovernanceDecisionID, &pc.ApprovalID, &pc.RequestID,
		&pc.IdempotencyKey, &pc.CorrelationID, &pc.CreatedAt, &pc.ExpiresAt, &pc.ClosedAt, &pc.ClosureReason)
	if err != nil {
		return nil, nil, err
	}

	if pc.ExpiresAt != nil && pc.ExpiresAt.Before(time.Now().UTC()) {
		if _, err := tx.Exec(ctx, `UPDATE approvals SET status='EXPIRED' WHERE approval_id=$1`, approvalID); err != nil {
			return nil, nil, err
		}
		if _, err := tx.Exec(ctx, `UPDATE pending_commands SET status='EXPIRED', closed_at=now(), closure_reason='EXPIRED' WHERE pending_command_id=$1`, pc.PendingCommandID); err != nil {
			return nil, nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, nil, err
		}
		return nil, nil, ports.ErrApprovalExpired
	}

	// Self-approval and non-human-decider checks intentionally do not
	// mutate anything — rolling back (the deferred tx.Rollback) leaves the
	// Approval exactly PENDING, so a legitimate different/human decider
	// can still resolve it later. An illegitimate attempt must not burn
	// the approval.
	if decidingPrincipal.PrincipalID == appr.RequestingPrincipalID {
		return nil, nil, ports.ErrSelfApproval
	}
	if decidingPrincipal.Kind != principal.KindHuman {
		return nil, nil, ports.ErrNonHumanDecider
	}

	status := approval.StatusRejected
	if approve {
		status = approval.StatusApproved
	}
	if _, err := tx.Exec(ctx, `
		UPDATE approvals SET status=$1, decided_by_principal_id=$2, decided_at=now(), reason=$3
		WHERE approval_id=$4`,
		status, decidingPrincipal.PrincipalID, reason, approvalID); err != nil {
		return nil, nil, err
	}
	appr.Status = status
	appr.DecidedByPrincipalID = &decidingPrincipal.PrincipalID
	appr.Reason = reason

	if !approve {
		if _, err := tx.Exec(ctx, `UPDATE pending_commands SET status='CANCELLED', closed_at=now(), closure_reason='REJECTED' WHERE pending_command_id=$1`, pc.PendingCommandID); err != nil {
			return nil, nil, err
		}
		pc.Status = "CANCELLED"
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return &pc, &appr, nil
}
