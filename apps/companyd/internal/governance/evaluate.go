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
}

// Evaluate implements the docs/architecture/governance.md 8-step pipeline
// in miniature: evidence presence stands in for steps 1-3 (real Authority
// validation is Phase 2 work); step 4 evaluates the hardcoded first-slice
// policy set default-deny; steps 5-7 compose autonomy (trivial with one
// rule set); step 8 (dispatch-time re-evaluation) is satisfied by the
// caller invoking Evaluate again immediately before dispatch, not by
// anything inside Evaluate itself.
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

	rule, matched := matchRule(req.Role, req.Action)
	if !matched || rule.Effect != policy.EffectPermit {
		decision.Outcome = policy.DecisionDeny
		reason := "no matching permit rule (default-deny)"
		decision.Reason = &reason
		decision.AutonomyLevel = policy.AutonomyHumanOnly
		return decision, nil
	}

	decision.MatchedRuleID = &rule.RuleID
	decision.AutonomyLevel = rule.Autonomy

	switch rule.Autonomy {
	case policy.AutonomyHumanOnly:
		decision.Outcome = policy.DecisionDeny
		reason := "human_only autonomy, no human requester in this slice"
		decision.Reason = &reason
	case policy.AutonomyApprovalRequired:
		decision.Outcome = policy.DecisionRequireApproval
	default:
		decision.Outcome = policy.DecisionAllow
	}
	return decision, nil
}
