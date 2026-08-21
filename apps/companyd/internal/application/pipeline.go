package application

import (
	"context"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/governance"
	"github.com/google/uuid"
)

// evaluateGovernance implements application.md steps 4-6: submit the exact
// proposal to Governance, and on REQUIRE_APPROVAL atomically persist the
// pending record. DENY is persisted as an audit decision and returned as
// Denied. Only on ALLOW does the caller proceed to the Kernel's final
// decision (ok=true).
func (a *Application) evaluateGovernance(ctx context.Context, cmd command.WorkflowCommandEnvelope, proposal command.GovernedCommandProposal, evidencePresent bool) (policy.GovernanceDecision, Result, bool) {
	decision, err := governance.Evaluate(ctx, governance.Request{
		RequestID:            cmd.RequestID,
		CorrelationID:        cmd.CorrelationID,
		OrganizationID:       cmd.OrganizationID,
		PrincipalID:          cmd.RequestingPrincipalID,
		EvidencePresent:      evidencePresent,
		Action:                proposal.Action,
		ResourceType:          proposal.ResourceType,
		ResourceID:            proposal.ResourceID,
		ProposalDigest:        proposal.ProposalDigest,
		TrustedContextDigest:  proposal.TrustedContextDigest,
	})
	if err != nil {
		return decision, Result{Outcome: Unavailable, Reasons: []string{err.Error()}}, false
	}

	switch decision.Outcome {
	case policy.DecisionDeny:
		_ = a.Repo.SaveGovernanceDecision(ctx, decision.DecisionID, decision.OrganizationID, decision.RequestID, decision.CorrelationID, decision.PrincipalID,
			decision.Action, decision.ResourceType, decision.ResourceID, decision.ProposalDigest, decision.TrustedContextDigest,
			decision.PolicyVersion, string(decision.AutonomyLevel), string(decision.Outcome), decision.MatchedRuleID, decision.Reason)
		return decision, Result{Outcome: Denied, Reasons: []string{command.ReasonGovernanceDenied}}, false

	case policy.DecisionRequireApproval:
		pc := &command.PendingCommand{
			PendingCommandID:     uuid.New(),
			OrganizationID:       cmd.OrganizationID,
			Status:               "PENDING",
			CommandType:          cmd.CommandType,
			ProposalDigest:       proposal.ProposalDigest,
			GovernanceDecisionID: decision.DecisionID,
			RequestID:            cmd.RequestID,
			IdempotencyKey:       cmd.IdempotencyKey,
			CorrelationID:        cmd.CorrelationID,
			CreatedAt:            time.Now().UTC(),
		}
		if cmd.ExpectedVersion != nil {
			pc.ExpectedWorkflowID = &cmd.WorkflowID
			pc.ExpectedWorkflowVersion = cmd.ExpectedVersion
		}
		appr := &approval.Approval{
			ApprovalID:            uuid.New(),
			OrganizationID:        cmd.OrganizationID,
			PendingCommandID:      pc.PendingCommandID,
			RequestingPrincipalID: cmd.RequestingPrincipalID,
			Action:                proposal.Action,
			ResourceType:          proposal.ResourceType,
			ResourceID:            proposal.ResourceID,
			ProposalDigest:        proposal.ProposalDigest,
			Status:                approval.StatusPending,
			CreatedAt:             time.Now().UTC(),
		}
		pc.ApprovalID = &appr.ApprovalID
		if err := a.Pending.CreatePendingApproval(ctx, pc, decision.DecisionID, appr); err != nil {
			return decision, Result{Outcome: Unavailable, Reasons: []string{err.Error()}}, false
		}
		return decision, Result{Outcome: ApprovalRequired}, false

	default: // ALLOW
		return decision, Result{}, true
	}
}
