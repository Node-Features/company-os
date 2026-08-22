package policy

import (
	"time"

	"github.com/google/uuid"
)

// Decision is Governance's exactly-one-of-three output. See
// docs/architecture/governance.md.
type Decision string

const (
	DecisionAllow           Decision = "ALLOW"
	DecisionDeny             Decision = "DENY"
	DecisionRequireApproval  Decision = "REQUIRE_APPROVAL"
)

// AutonomyLevel composes most-restrictive-wins across matched policies.
type AutonomyLevel string

const (
	AutonomyAutomatic        AutonomyLevel = "AUTOMATIC"
	AutonomyApprovalRequired AutonomyLevel = "APPROVAL_REQUIRED"
	AutonomyHumanOnly        AutonomyLevel = "HUMAN_ONLY"
)

// Effect is whether a Rule permits or forbids a matching Action.
type Effect string

const (
	EffectPermit Effect = "PERMIT"
	EffectForbid Effect = "FORBID"
)

// Resource is the governed Action's target.
type Resource struct {
	Type string
	ID   string
}

// PrincipalRef is the minimal Principal shape Governance needs — avoids an
// import of internal/domain/principal to keep policy dependency-free per
// its doc comment.
type PrincipalRef struct {
	PrincipalID   uuid.UUID
	Kind          string
	Authenticated bool
}

// Role identifies a first-slice illustrative agent/principal role for
// role-scoped policy matching (ADR-0008). It is this slice's stand-in for a
// real role/delegation lookup — docs/domain/principal.md's "delegation
// references" — the same way Request.EvidencePresent stands in for real
// authentication evidence: a caller-asserted Role is trusted here, not yet
// verified against a persisted binding. Not to be confused with
// docs/domain/capability.md's Capability — a provider-independent dispatch
// contract (e.g. "generate text"), a different concept that already owns
// that name.
type Role string

// Rule is one hardcoded first-slice policy entry (first-slice plan
// decision #12 — policy.md leaves policy administration as future work).
// An empty Role matches every role, preserving every rule written before
// ADR-0008 added role-scoped matching.
type Rule struct {
	RuleID       string
	Effect       Effect
	Role         Role
	ActionPrefix string
	Action       string
	Autonomy     AutonomyLevel
}

// GovernanceDecision is the persisted record of one Evaluate call.
type GovernanceDecision struct {
	DecisionID            uuid.UUID
	OrganizationID        uuid.UUID
	RequestID             uuid.UUID
	CorrelationID         uuid.UUID
	PrincipalID           uuid.UUID
	Action                string
	ResourceType          string
	ResourceID            string
	ProposalDigest        string
	TrustedContextDigest  string
	PolicyVersion         string
	AutonomyLevel         AutonomyLevel
	Outcome               Decision
	MatchedRuleID         *string
	Reason                *string
	DecidedAt             time.Time
}
