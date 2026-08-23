package governance

import (
	"context"
	"errors"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/google/uuid"
)

// ErrNoEvidence is returned when Request.Evidence is absent — this slice's
// stand-in for real Authority/Identity validation (docs/architecture/governance.md
// steps 1-3), the concrete gap Phase 2 closes with real Supabase-Auth JWT
// verification (first-slice plan decision #5).
var ErrNoEvidence = errors.New("governance: missing principal evidence")

// Request is the minimal first-slice Governance decision request.
// See docs/architecture/governance.md.
type Request struct {
	RequestID      uuid.UUID
	CorrelationID  uuid.UUID
	OrganizationID uuid.UUID
	PrincipalID    uuid.UUID
	// EvidencePresent stands in for real authentication evidence
	// validation this slice (decision #5).
	EvidencePresent      bool
	// Role is this slice's stand-in for a resolved role/delegation lookup —
	// see policy.Role's doc comment (ADR-0008). Zero value "" matches only
	// role-agnostic rules (Role-less Rules), never a role-scoped one — an
	// unrecognized or absent Role fails closed, it does not fall back to
	// "any role."
	Role                  policy.Role
	Action                string
	ResourceType          string
	ResourceID            string
	ProposalDigest        string
	TrustedContextDigest  string
	// ResourceOwnerPrincipalID, when non-nil, is the resource's trusted,
	// server-resolved owning Principal (e.g. a Workflow's
	// InitiatingPrincipalID loaded from persistence) — never a
	// client-asserted value. It stands in for a real per-resource
	// Authority grant (governance.md step 3: "Confirm the Principal has
	// active Authority covering the request") the same way
	// EvidencePresent stands in for real authentication evidence: this
	// slice's Rule model (policy.Rule) is action-scoped only and cannot
	// express "only this resource's own initiator," so ownership is
	// checked directly rather than through the permit/forbid rule table.
	// Nil means this action carries no resource-instance ownership
	// requirement.
	ResourceOwnerPrincipalID *uuid.UUID
	// AdditionalAutonomyRequirement, when non-nil, is a resource-instance
	// autonomy requirement the caller resolved from trusted Context (e.g.
	// "this Workflow is READY, an in-flight cancel needs human sign-off") —
	// composed with the matched policy Rule's autonomy using
	// most-restrictive-wins (governance.md step 5: "human_only >
	// approval_required > automatic"). Rule.Autonomy alone can only express
	// a per-action requirement; this is how a per-resource-instance one
	// composes with it without inventing a second Rule table.
	AdditionalAutonomyRequirement *policy.AutonomyLevel
	// ApprovalEvidence, when non-nil, asserts a specific resolved Approval
	// the caller has already verified is APPROVED, unconsumed, and
	// unexpired — Governance trusts this evidence rather than querying the
	// Approval store itself, the same way EvidencePresent stands in for
	// real authentication evidence this slice. Only unlocks ALLOW when its
	// ProposalDigest exactly matches this request's (governance.md step 7:
	// "unless valid unconsumed approval evidence exactly matches the
	// request").
	ApprovalEvidence *ApprovalEvidence
}

// ApprovalEvidence is the minimal caller-supplied assertion Evaluate needs
// for governance.md step 7. See Request.ApprovalEvidence.
type ApprovalEvidence struct {
	ApprovalID     uuid.UUID
	ProposalDigest string
}

// moreRestrictive returns whichever of a/b is more restrictive under
// governance.md step 5's ordering: human_only > approval_required >
// automatic.
func moreRestrictive(a, b policy.AutonomyLevel) policy.AutonomyLevel {
	rank := map[policy.AutonomyLevel]int{
		policy.AutonomyAutomatic:        0,
		policy.AutonomyApprovalRequired: 1,
		policy.AutonomyHumanOnly:        2,
	}
	if rank[a] >= rank[b] {
		return a
	}
	return b
}

// Evaluate implements the docs/architecture/governance.md 8-step pipeline
// in miniature: evidence presence stands in for steps 1-2 (real identity
// verification is ROADMAP.md Phase 3 Slice 3 work); ResourceOwnerPrincipalID
// implements step 3's per-resource Authority check for actions that declare
// one (ROADMAP.md Phase 3 Slice 1 — the first concrete, HTTP-reachable DENY
// policy); step 4 evaluates the hardcoded first-slice policy set
// default-deny; steps 5-7 compose autonomy (trivial with one rule set);
// step 8 (dispatch-time re-evaluation) is satisfied by the caller invoking
// Evaluate again immediately before dispatch, not by anything inside
// Evaluate itself.
func Evaluate(ctx context.Context, req Request) (policy.GovernanceDecision, error) {
	now := time.Now().UTC()
	decision := policy.GovernanceDecision{
		DecisionID:            uuid.New(),
		OrganizationID:        req.OrganizationID,
		RequestID:             req.RequestID,
		CorrelationID:         req.CorrelationID,
		PrincipalID:           req.PrincipalID,
		Action:                req.Action,
		ResourceType:          req.ResourceType,
		ResourceID:            req.ResourceID,
		ProposalDigest:        req.ProposalDigest,
		TrustedContextDigest:  req.TrustedContextDigest,
		PolicyVersion:         PolicyVersion,
		DecidedAt:             now,
	}

	if !req.EvidencePresent {
		decision.Outcome = policy.DecisionDeny
		reason := "missing principal evidence"
		decision.Reason = &reason
		decision.AutonomyLevel = policy.AutonomyHumanOnly
		return decision, nil
	}

	if req.ResourceOwnerPrincipalID != nil && *req.ResourceOwnerPrincipalID != req.PrincipalID {
		decision.Outcome = policy.DecisionDeny
		reason := "principal does not own this resource"
		decision.Reason = &reason
		decision.AutonomyLevel = policy.AutonomyHumanOnly
		return decision, nil
	}

	rule, matched := matchRule(req.Role, req.Action)
	if !matched || rule.Effect != policy.EffectPermit {
		decision.Outcome = policy.DecisionDeny
		reason := "no matching permit rule (default-deny)"
		decision.Reason = &reason
		decision.AutonomyLevel = policy.AutonomyHumanOnly
		return decision, nil
	}

	decision.MatchedRuleID = &rule.RuleID
	effectiveAutonomy := rule.Autonomy
	if req.AdditionalAutonomyRequirement != nil {
		effectiveAutonomy = moreRestrictive(effectiveAutonomy, *req.AdditionalAutonomyRequirement)
	}
	decision.AutonomyLevel = effectiveAutonomy

	switch effectiveAutonomy {
	case policy.AutonomyHumanOnly:
		decision.Outcome = policy.DecisionDeny
		reason := "human_only autonomy, no human requester in this slice"
		decision.Reason = &reason
	case policy.AutonomyApprovalRequired:
		if req.ApprovalEvidence != nil && req.ApprovalEvidence.ProposalDigest == req.ProposalDigest {
			decision.Outcome = policy.DecisionAllow
		} else {
			decision.Outcome = policy.DecisionRequireApproval
		}
	default:
		decision.Outcome = policy.DecisionAllow
	}
	return decision, nil
}
