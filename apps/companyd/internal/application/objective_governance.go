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

// evaluateObjectiveProposalGovernance is ROADMAP.md Phase 4 Slice 4's
// governance helper — structurally a near-duplicate of pipeline.go's
// evaluateGovernance, but typed on ObjectiveProposalCommandEnvelope
// instead of WorkflowCommandEnvelope. Not a generalization of
// evaluateGovernance: that function reads cmd.RequestID/OrganizationID/etc.
// directly off the concrete WorkflowCommandEnvelope type, and
// ResolveApproval's resume dispatch is a closed switch over CommandType —
// generalizing either would touch tested, working Workflow code for one
// new, structurally different caller. Also unlike evaluateAutomaticGovernance
// (Research/M&E/Finance's helper), this one DOES persist a PendingCommand
// on REQUIRE_APPROVAL — this is the first governed action whose policy
// (governance/policy.go's "objective-propose" rule) actually reaches that
// outcome.
func (a *Application) evaluateObjectiveProposalGovernance(ctx context.Context, cmd command.ObjectiveProposalCommandEnvelope, proposal command.GovernedCommandProposal, approvalEvidence *governance.ApprovalEvidence) (policy.GovernanceDecision, Result, bool) {
	decision, err := governance.Evaluate(ctx, governance.Request{
		RequestID:               cmd.RequestID,
		CorrelationID:           cmd.CorrelationID,
		OrganizationID:          cmd.OrganizationID,
		PrincipalID:             cmd.RequestingPrincipalID,
		EvidencePresent:         true,
		RequestingPrincipalKind: cmd.RequestingPrincipalKind,
		Action:                  proposal.Action,
		ResourceType:            proposal.ResourceType,
		ResourceID:              proposal.ResourceID,
		ProposalDigest:          proposal.ProposalDigest,
		TrustedContextDigest:    proposal.TrustedContextDigest,
		ApprovalEvidence:        approvalEvidence,
	})
	if err != nil {
		return decision, Result{Outcome: Unavailable, Reasons: []string{err.Error()}}, false
	}

	// Persisted for every outcome, not only DENIED — same reasoning
	// evaluateGovernance's identical call documents: execution_intents and
	// pending_commands both reference governance_decisions by ID.
	if err := a.Repo.SaveGovernanceDecision(ctx, decision.DecisionID, decision.OrganizationID, decision.RequestID, decision.CorrelationID, decision.PrincipalID,
		decision.Action, decision.ResourceType, decision.ResourceID, decision.ProposalDigest, decision.TrustedContextDigest,
		decision.PolicyVersion, string(decision.AutonomyLevel), string(decision.Outcome), decision.MatchedRuleID, decision.Reason); err != nil {
		return decision, Result{Outcome: Unavailable, Reasons: []string{err.Error()}}, false
	}

	switch decision.Outcome {
	case policy.DecisionDenied:
		return decision, Result{Outcome: Denied, Reasons: []string{command.ReasonGovernanceDenied}}, false

	case policy.DecisionRequireApproval:
		// GovernedPayload persists the exact ObjectiveProposalCommandEnvelope
		// so resumeProposeObjective can replay it byte-for-byte — same
		// digest-stability reasoning evaluateGovernance's identical branch
		// documents for Workflow commands.
		payload, err := json.Marshal(cmd)
		if err != nil {
			return decision, Result{Outcome: Unavailable, Reasons: []string{err.Error()}}, false
		}
		now := time.Now().UTC()
		expiresAt := now.Add(approvalTTL)
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
			CreatedAt:            now,
			ExpiresAt:            &expiresAt,
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
			CreatedAt:             now,
		}
		pc.ApprovalID = &appr.ApprovalID
		if err := a.Pending.CreatePendingApproval(ctx, pc, decision.DecisionID, appr); err != nil {
			return decision, Result{Outcome: Unavailable, Reasons: []string{err.Error()}}, false
		}
		return decision, Result{Outcome: ApprovalRequired, ApprovalID: &appr.ApprovalID}, false

	default: // AUTOMATIC or HUMAN_ONLY
		return decision, Result{}, true
	}
}
