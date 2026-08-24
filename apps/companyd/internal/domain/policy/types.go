package policy

import (
	"time"

	"github.com/google/uuid"
)

// Decision is Governance's exactly-one-of-four output. See
// docs/architecture/governance.md and docs/architecture/authority-model.md.
//
// AUTOMATIC and HUMAN_ONLY are both "proceed" dispositions — see Allows().
// They are kept distinct rather than collapsed into one "ALLOW" value
// because they answer a different audit question: AUTOMATIC means no human
// involvement was required at all; HUMAN_ONLY means this specific request
// proceeded because it was made directly by an eligible human principal
// under an autonomy requirement that forbids automatic/agent/service
// execution. A REQUIRE_APPROVAL request that later resumes with matching
// ApprovalEvidence reports AUTOMATIC on the resumed decision (the gate is
// now cleared) — the fact that approval was originally required is not
// lost, it stays on record via that decision's AutonomyLevel and the
// linked ApprovalID, not via the Outcome value.
//
// Renamed 2026-08-24 (docs/adr/ADR-0010-authority-model-formalization.md)
// from the prior three-value ALLOW/DENY/REQUIRE_APPROVAL vocabulary. This
// is a forward-only rename: historical GovernanceDecision rows persisted
// under the old vocabulary keep their original ALLOW/DENY spelling exactly
// as recorded — governance.md's own "audit records are append-only
// organizational evidence" invariant means old evidence is never rewritten
// to match new terminology. The DB CHECK constraint on
// governance_decisions.outcome was widened additively to accept both
// vocabularies (see the migration in that ADR's Implementation section).
type Decision string

const (
	// DecisionAutomatic means the request proceeded with no human
	// involvement required — either no autonomy restriction applied, or a
	// prior REQUIRE_APPROVAL gate was just cleared by matching
	// ApprovalEvidence.
	DecisionAutomatic Decision = "AUTOMATIC"
	// DecisionRequireApproval means the request is otherwise eligible but
	// lacks valid, unconsumed approval evidence — not a weak allow, must
	// never be sent to an executor (governance.md).
	DecisionRequireApproval Decision = "REQUIRE_APPROVAL"
	// DecisionHumanOnly means the request proceeded specifically because it
	// was made directly by an eligible human principal under a human_only
	// autonomy requirement — a non-human requester under the same
	// requirement gets DecisionDenied instead, never this value.
	DecisionHumanOnly Decision = "HUMAN_ONLY"
	// DecisionDenied means the request does not proceed: no matching
	// permit, a matching forbid, a failed resource-ownership/exclusion
	// check, missing evidence, or an ineligible requester under a
	// human_only requirement.
	DecisionDenied Decision = "DENIED"
)

// Allows reports whether d is one of the two "proceed" dispositions
// (AUTOMATIC or HUMAN_ONLY) — the single place callers that gate dispatch
// on "was this request authorized to proceed" should check, instead of
// comparing against one specific value. Both runtime.Runtime.execute and
// kernel/workflow.verifyAllow use this rather than a literal equality
// check, so a request correctly authorized under HUMAN_ONLY is never
// mistaken for an unauthorized one.
func (d Decision) Allows() bool {
	return d == DecisionAutomatic || d == DecisionHumanOnly
}

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
	DecisionID           uuid.UUID
	OrganizationID       uuid.UUID
	RequestID            uuid.UUID
	CorrelationID        uuid.UUID
	PrincipalID          uuid.UUID
	Action               string
	ResourceType         string
	ResourceID           string
	ProposalDigest       string
	TrustedContextDigest string
	PolicyVersion        string
	AutonomyLevel        AutonomyLevel
	Outcome              Decision
	MatchedRuleID        *string
	Reason               *string
	DecidedAt            time.Time
}
