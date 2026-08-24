package application

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/governance"
	"github.com/google/uuid"
)

// evaluateAutomaticGovernance is the governance call shared by every
// AUTOMATIC-autonomy-only use case (Research's five actions, Phase 4 Slice
// 1; M&E's two actions, Phase 4 Slice 2) — simpler than evaluateGovernance
// (workflow_start.go's helper), which is tied to
// command.WorkflowCommandEnvelope/GovernedCommandProposal for Workflow's
// approval-replay path. Every caller of this helper is AUTOMATIC autonomy by
// its own department's documented scope boundary, so REQUIRE_APPROVAL has no
// persistence path here — if policy ever raises one of these actions to
// APPROVAL_REQUIRED, that action needs the Approval machinery
// workflow_cancel.go/approval_resolve.go already built, not a
// silently-different one improvised here.
func (a *Application) evaluateAutomaticGovernance(ctx context.Context, requestID, organizationID, principalID uuid.UUID, action, resourceType, resourceID, proposalDigest string) (policy.GovernanceDecision, Result, bool) {
	decision, err := governance.Evaluate(ctx, governance.Request{
		RequestID:            requestID,
		CorrelationID:        requestID,
		OrganizationID:       organizationID,
		PrincipalID:          principalID,
		EvidencePresent:      true,
		Action:               action,
		ResourceType:         resourceType,
		ResourceID:           resourceID,
		ProposalDigest:       proposalDigest,
		TrustedContextDigest: proposalDigest,
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
	case policy.DecisionDenied:
		return decision, Result{Outcome: Denied, Reasons: []string{command.ReasonGovernanceDenied}}, false
	case policy.DecisionRequireApproval:
		return decision, Result{Outcome: Unavailable, Reasons: []string{"approval_required_not_supported_for_this_action"}}, false
	default: // AUTOMATIC or HUMAN_ONLY
		return decision, Result{}, true
	}
}
