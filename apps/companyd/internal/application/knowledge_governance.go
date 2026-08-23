package application

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/governance"
	"github.com/google/uuid"
)

// evaluateKnowledgeApprovalGovernance is ROADMAP.md Phase 5 Slice 2's
// governance helper — structurally a near-duplicate of
// objective_governance.go's evaluateObjectiveProposalGovernance, typed on
// KnowledgeApprovalCommandEnvelope instead of ObjectiveProposalCommandEnvelope.
// Not a generalization of it: same reasoning objective_governance.go
// documents (evaluateGovernance reads fields directly off the concrete
// WorkflowCommandEnvelope type, and ResolveApproval's resume dispatch is a
// closed switch over CommandType). The one real addition over Objective's
// helper: excludedPrincipalID implements separation-of-duties (the review
// requester must not be the candidate's producer) as a real Governance DENY
// via governance.Request.ExcludedPrincipalID, not a Kernel legality check.
func (a *Application) evaluateKnowledgeApprovalGovernance(ctx context.Context, cmd command.KnowledgeApprovalCommandEnvelope, proposal command.GovernedCommandProposal, excludedPrincipalID uuid.UUID, approvalEvidence *governance.ApprovalEvidence) (policy.GovernanceDecision, Result, bool) {
	decision, err := governance.Evaluate(ctx, governance.Request{
		RequestID:            cmd.RequestID,
		CorrelationID:        cmd.CorrelationID,
		OrganizationID:       cmd.OrganizationID,
		PrincipalID:          cmd.RequestingPrincipalID,
		EvidencePresent:      true,
		Action:               proposal.Action,
		ResourceType:         proposal.ResourceType,
		ResourceID:           proposal.ResourceID,
		ProposalDigest:       proposal.ProposalDigest,
		TrustedContextDigest: proposal.TrustedContextDigest,
		ExcludedPrincipalID:  &excludedPrincipalID,
		ApprovalEvidence:     approvalEvidence,
	})
	if err != nil {
		return decision, Result{Outcome: Unavailable, Reasons: []string{err.Error()}}, false
	}

	if err := a.Repo.SaveGovernanceDecision(ctx, decision.DecisionID, decision.OrganizationID, decision.RequestID, decision.CorrelationID, decision.PrincipalID,
		decision.Action, decision.ResourceType, decision.ResourceID, decision.ProposalDigest, decision.TrustedContextDigest,
		decision.PolicyVersion, string(decision.AutonomyLevel), string(decision.Outcome), decision.MatchedRuleID, decision.Reason); err != nil {
		return decision, Result{Outcome: Unavailable, Reasons: []string{err.Error()}}, false
	}

	switch decision.Outcome {
	case policy.DecisionDeny:
		return decision, Result{Outcome: Denied, Reasons: []string{command.ReasonGovernanceDenied}}, false

	case policy.DecisionRequireApproval:
		payload, err := json.Marshal(cmd)
		if err != nil {
			return decision, Result{Outcome: Unavailable, Reasons: []string{err.Error()}}, false
		}
		pc := &command.PendingCommand{
			PendingCommandID:     uuid.New(),
			OrganizationID:       cmd.OrganizationID,
			Status:               "PENDING",
			CommandType:          cmd.CommandType,
			ProposalDigest:       proposal.ProposalDigest,
			GovernedPayload:      payload,
			GovernanceDecisionID: decision.DecisionID,
			RequestID:            cmd.RequestID,
			IdempotencyKey:       cmd.IdempotencyKey,
			CorrelationID:        cmd.CorrelationID,
			CreatedAt:            time.Now().UTC(),
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
		return decision, Result{Outcome: ApprovalRequired, ApprovalID: &appr.ApprovalID}, false

	default: // ALLOW
		return decision, Result{}, true
	}
}
