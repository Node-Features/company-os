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

// governanceOptions bundles evaluateGovernance's per-call context so its
// signature doesn't keep growing a new positional *uuid.UUID/pointer
// argument per governance capability. Zero value carries no resource-
// instance requirement of any kind.
type governanceOptions struct {
	ResourceOwnerPrincipalID     *uuid.UUID
	AdditionalAutonomyRequirement *policy.AutonomyLevel
	ApprovalEvidence              *governance.ApprovalEvidence
}

// evaluateGovernance implements application.md steps 4-6: submit the exact
// proposal to Governance, persist the resulting decision unconditionally —
// governance.md: "Policy, authority, approval, and decision records are
// persisted before dependent execution continues" — and on
// REQUIRE_APPROVAL additionally persist the pending record. Only on ALLOW
// does the caller proceed to the Kernel's final decision (ok=true).
func (a *Application) evaluateGovernance(ctx context.Context, cmd command.WorkflowCommandEnvelope, proposal command.GovernedCommandProposal, evidencePresent bool, opts governanceOptions) (policy.GovernanceDecision, Result, bool) {
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
		ResourceOwnerPrincipalID:      opts.ResourceOwnerPrincipalID,
		AdditionalAutonomyRequirement: opts.AdditionalAutonomyRequirement,
		ApprovalEvidence:              opts.ApprovalEvidence,
	})
	if err != nil {
		return decision, Result{Outcome: Unavailable, Reasons: []string{err.Error()}}, false
	}

	// Persisted for every outcome, not only DENY: ALLOW decisions are
	// referenced by execution_intents.governance_decision_id, and
	// REQUIRE_APPROVAL decisions are referenced by pending_commands'
	// governance_decision_id foreign key — both would otherwise point at a
	// row that was never written. A failure here is surfaced as
	// Unavailable rather than silently letting an unrecorded decision
	// through.
	if err := a.Repo.SaveGovernanceDecision(ctx, decision.DecisionID, decision.OrganizationID, decision.RequestID, decision.CorrelationID, decision.PrincipalID,
		decision.Action, decision.ResourceType, decision.ResourceID, decision.ProposalDigest, decision.TrustedContextDigest,
		decision.PolicyVersion, string(decision.AutonomyLevel), string(decision.Outcome), decision.MatchedRuleID, decision.Reason); err != nil {
		return decision, Result{Outcome: Unavailable, Reasons: []string{err.Error()}}, false
	}

	switch decision.Outcome {
	case policy.DecisionDeny:
		return decision, Result{Outcome: Denied, Reasons: []string{command.ReasonGovernanceDenied}}, false

	case policy.DecisionRequireApproval:
		// GovernedPayload persists the exact WorkflowCommandEnvelope so a
		// later resumption (Application.ResolveApproval) can replay it
		// unmutated — kernel/workflow/digest.go's ProposalDigest hashes the
		// entire envelope, so only a byte-for-byte replay reproduces the
		// same digest ApprovalEvidence must match. Also satisfies the
		// column's NOT NULL constraint, unset before this fix.
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
		return decision, Result{Outcome: ApprovalRequired, ApprovalID: &appr.ApprovalID}, false

	default: // ALLOW
		return decision, Result{}, true
	}
}
