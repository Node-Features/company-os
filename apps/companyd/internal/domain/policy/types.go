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

// Rule is one hardcoded first-slice policy entry (first-slice plan
// decision #12 — policy.md leaves policy administration as future work).
type Rule struct {
	RuleID       string
	Effect       Effect
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
