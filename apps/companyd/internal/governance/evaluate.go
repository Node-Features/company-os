package governance

import (
	"context"
	"errors"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
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
	EvidencePresent bool
	// Role is this slice's stand-in for a resolved role/delegation lookup —
	// see policy.Role's doc comment (ADR-0008). Zero value "" matches only
	// role-agnostic rules (Role-less Rules), never a role-scoped one — an
	// unrecognized or absent Role fails closed, it does not fall back to
	// "any role."
	Role policy.Role
	// RequestingPrincipalKind is the requester's real principal.Kind
	// (HUMAN/AGENT/SERVICE/PROVIDER) — needed only to make a HUMAN_ONLY
	// autonomy requirement (step 6 below) real rather than dead code: it is
	// the fact "is this requester actually human," which nothing else in
	// this Request carries (EvidencePresent is a bare bool, Role is a
	// caller-asserted string with no Kind attached). Populated today from
	// whatever principal.Principal value the caller already trusts as the
	// requester (a fixtures.Registry value in every current call site) —
	// this field does not by itself require or imply real JWT-authenticated
	// caller identity, which is separately tracked
	// (docs/audit/gap-approval-principal-attribution.md). Zero value
	// (empty Kind) is never treated as human — a HUMAN_ONLY requirement
	// fails closed against an unset Kind, same as every other missing-input
	// case in this function.
	RequestingPrincipalKind principal.Kind
	Action                  string
	ResourceType            string
	ResourceID              string
	ProposalDigest          string
	TrustedContextDigest    string
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
	// ExcludedPrincipalID, when non-nil, is a resource-derived Principal the
	// requester must NOT be — the inverse of ResourceOwnerPrincipalID.
	// Implements per-resource separation-of-duties (docs/architecture/
	// knowledge.md's knowledge.approve step 6: "require independence from
	// authorship") as a real Governance-layer authorization check, not a
	// Kernel legality one — same precedent as ResourceOwnerPrincipalID
	// moving CancelWorkflow's ownership check out of Kernel (ROADMAP.md
	// Phase 3 Slice 1). Nil means this action carries no exclusion.
	ExcludedPrincipalID *uuid.UUID
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
// in miniature, and is the concrete realization of
// docs/architecture/authority-model.md's conceptual evaluator stages:
//
//   - Normalize / Resolve Principal / Resolve Resource: done by the caller
//     before Evaluate is invoked — Application assembles Request from the
//     already-normalized GovernedCommandProposal (canonical-digest,
//     Kernel-resolved resource facts) and whatever principal.Principal
//     value it currently trusts as the requester.
//   - Resolve Capability / Resolve Policy: matchRule's Role x Action
//     lookup below — "Capability" has no separate type in this codebase
//     (docs/adr/ADR-0008-authority-capability-model.md; that name is
//     already taken by docs/domain/capability.md's dispatch-contract
//     concept) — what a Role may do is fully expressed by which Action
//     rules match it.
//   - Evaluate Authority: evidence presence stands in for steps 1-2 (real
//     identity verification is ROADMAP.md Phase 3 Slice 3 work);
//     ResourceOwnerPrincipalID/ExcludedPrincipalID implement step 3's
//     per-resource Authority/Constraints check for actions that declare
//     one; matchRule's default-deny match is step 4.
//   - Evaluate Autonomy: steps 5-6, most-restrictive-wins composition plus
//     the real (not dead-code) HUMAN_ONLY branch.
//   - Evaluate Approval / Decision: step 7's ApprovalEvidence check,
//     producing the final Decision.
//
// Step 8 (dispatch-time re-evaluation) is satisfied by the caller invoking
// Evaluate again immediately before dispatch, not by anything inside
// Evaluate itself.
func Evaluate(ctx context.Context, req Request) (policy.GovernanceDecision, error) {
	now := time.Now().UTC()
	decision := policy.GovernanceDecision{
		DecisionID:           uuid.New(),
		OrganizationID:       req.OrganizationID,
		RequestID:            req.RequestID,
		CorrelationID:        req.CorrelationID,
		PrincipalID:          req.PrincipalID,
		Action:               req.Action,
		ResourceType:         req.ResourceType,
		ResourceID:           req.ResourceID,
		ProposalDigest:       req.ProposalDigest,
		TrustedContextDigest: req.TrustedContextDigest,
		PolicyVersion:        PolicyVersion,
		DecidedAt:            now,
	}

	// --- Resolve Authority: evidence, resource-instance constraints, and
	// default-deny Role x Action policy match. Any failure here fails
	// closed before autonomy is even considered — AutonomyLevel is set to
	// the most restrictive label (HUMAN_ONLY) as an audit sentinel, not a
	// claim that a HUMAN_ONLY rule was actually matched.
	if !req.EvidencePresent {
		decision.Outcome = policy.DecisionDenied
		reason := "missing principal evidence"
		decision.Reason = &reason
		decision.AutonomyLevel = policy.AutonomyHumanOnly
		return decision, nil
	}

	if req.ResourceOwnerPrincipalID != nil && *req.ResourceOwnerPrincipalID != req.PrincipalID {
		decision.Outcome = policy.DecisionDenied
		reason := "principal does not own this resource"
		decision.Reason = &reason
		decision.AutonomyLevel = policy.AutonomyHumanOnly
		return decision, nil
	}

	if req.ExcludedPrincipalID != nil && *req.ExcludedPrincipalID == req.PrincipalID {
		decision.Outcome = policy.DecisionDenied
		reason := "principal excluded from this resource (separation of duties)"
		decision.Reason = &reason
		decision.AutonomyLevel = policy.AutonomyHumanOnly
		return decision, nil
	}

	rule, matched := matchRule(req.Role, req.Action)
	if !matched || rule.Effect != policy.EffectPermit {
		decision.Outcome = policy.DecisionDenied
		reason := "no matching permit rule (default-deny)"
		decision.Reason = &reason
		decision.AutonomyLevel = policy.AutonomyHumanOnly
		return decision, nil
	}
	decision.MatchedRuleID = &rule.RuleID

	// --- Evaluate Autonomy: compose the matched rule's autonomy with any
	// resource-instance requirement the caller resolved from trusted
	// context, most-restrictive-wins (governance.md step 5).
	effectiveAutonomy := rule.Autonomy
	if req.AdditionalAutonomyRequirement != nil {
		effectiveAutonomy = moreRestrictive(effectiveAutonomy, *req.AdditionalAutonomyRequirement)
	}
	decision.AutonomyLevel = effectiveAutonomy

	// --- Evaluate Approval: dispatch on the effective autonomy class to
	// produce the final Decision.
	switch effectiveAutonomy {
	case policy.AutonomyHumanOnly:
		// governance.md step 6: "an agent or service request is DENY; an
		// eligible human principal continues." A human requester gets a
		// real, distinct DecisionHumanOnly (not DecisionAutomatic) so the
		// audit trail always shows exactly why this specific request
		// proceeded. Zero-value RequestingPrincipalKind ("") is never
		// treated as human — fails closed, same as every other
		// missing-input case in this function.
		if req.RequestingPrincipalKind == principal.KindHuman {
			decision.Outcome = policy.DecisionHumanOnly
		} else {
			decision.Outcome = policy.DecisionDenied
			reason := "human_only autonomy requires a human requester"
			decision.Reason = &reason
		}
	case policy.AutonomyApprovalRequired:
		if req.ApprovalEvidence != nil && req.ApprovalEvidence.ProposalDigest == req.ProposalDigest {
			decision.Outcome = policy.DecisionAutomatic
		} else {
			decision.Outcome = policy.DecisionRequireApproval
		}
	default:
		decision.Outcome = policy.DecisionAutomatic
	}
	return decision, nil
}
