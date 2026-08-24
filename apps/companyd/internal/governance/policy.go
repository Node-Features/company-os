package governance

import "github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"

// PolicyVersion identifies the hardcoded first-slice policy set persisted
// on every GovernanceDecision. Administering policy as governed, persisted
// data is future work (policy.md leaves policy lifecycle as an open
// question); first-slice plan decision #12 hardcodes it here instead.
const PolicyVersion = "first-slice-v1"

// firstSlicePolicies is default-deny: a request is ALLOW only if at least
// one rule permits it and none forbid it (docs/architecture/governance.md's
// policy evaluation step).
//
// The research_agent/finance_agent rules below are ADR-0008's illustrative
// role-to-action mapping, not a canonical department authority model —
// docs/security/agent-authority.md (ROADMAP.md Phase 2 Slice 2) is where
// that gets decided for real. transfer_funds is deliberately
// AutonomyApprovalRequired, not simply absent: "requires additional
// authority" maps precisely onto REQUIRE_APPROVAL (eligible, but needs
// Approval evidence this slice's requests never carry), which is more
// exact than a bare DENY would be. send_customer_message and any other
// action outside this list has no rule at all and relies entirely on
// default-deny — no explicit "deny" entry is written for it, matching
// governance.md: "Default is deny: absence of a matching permit never
// implies access."
var firstSlicePolicies = []policy.Rule{
	{RuleID: "workflow-actions", Effect: policy.EffectPermit, ActionPrefix: "workflow.", Autonomy: policy.AutonomyAutomatic},
	{RuleID: "capability-dispatch", Effect: policy.EffectPermit, Action: "capability.intelligence.dispatch", Autonomy: policy.AutonomyAutomatic},

	// ROADMAP.md Phase 4 Slice 1's five Research use cases
	// (docs/workflows/research-loop.md). Exact-Action entries, not a shared
	// "research." prefix, so they don't overlap the pre-existing
	// role-scoped illustrative research_agent rules below. AUTOMATIC per
	// research-loop.md's scope boundary — research.md's own open question
	// on which classes need human review is deliberately left unresolved,
	// not answered here.
	{RuleID: "research-signal-submit", Effect: policy.EffectPermit, Action: "research.signal.submit", Autonomy: policy.AutonomyAutomatic},
	{RuleID: "research-question-open", Effect: policy.EffectPermit, Action: "research.question.open", Autonomy: policy.AutonomyAutomatic},
	{RuleID: "research-evidence-record", Effect: policy.EffectPermit, Action: "research.evidence.record", Autonomy: policy.AutonomyAutomatic},
	{RuleID: "research-finding-publish", Effect: policy.EffectPermit, Action: "research.finding.publish", Autonomy: policy.AutonomyAutomatic},
	{RuleID: "research-recommendation-issue", Effect: policy.EffectPermit, Action: "research.recommendation.issue", Autonomy: policy.AutonomyAutomatic},

	// ROADMAP.md Phase 4 Slice 2's two M&E use cases
	// (docs/departments/monitoring-evaluation.md). AUTOMATIC per the same
	// reasoning as Research's rules above — monitoring-evaluation.md's own
	// open question on which outcomes need human/dual evaluation is
	// deliberately left unresolved.
	{RuleID: "me-metric-record", Effect: policy.EffectPermit, Action: "me.metric.record", Autonomy: policy.AutonomyAutomatic},
	{RuleID: "me-evaluation-run", Effect: policy.EffectPermit, Action: "me.evaluation.run", Autonomy: policy.AutonomyAutomatic},

	// ROADMAP.md Phase 4 Slice 3's four Finance use cases
	// (docs/departments/finance.md). Exact-Action entries, distinct from
	// the pre-existing illustrative finance_agent role-scoped rules below
	// (finance.read_financial_data/.create_payment_request/.transfer_funds)
	// — no overlap since matching here is exact-Action, not prefix.
	// AUTOMATIC per the same reasoning as Research's/M&E's rules above —
	// finance.md's own open question on budget authority/approval
	// thresholds is deliberately left unresolved.
	{RuleID: "finance-budget-create", Effect: policy.EffectPermit, Action: "finance.budget.create", Autonomy: policy.AutonomyAutomatic},
	{RuleID: "finance-constraint-create", Effect: policy.EffectPermit, Action: "finance.constraint.create", Autonomy: policy.AutonomyAutomatic},
	{RuleID: "finance-usage-record", Effect: policy.EffectPermit, Action: "finance.usage.record", Autonomy: policy.AutonomyAutomatic},
	{RuleID: "finance-evaluation-run", Effect: policy.EffectPermit, Action: "finance.evaluation.run", Autonomy: policy.AutonomyAutomatic},

	// ROADMAP.md Phase 4 Slice 4's Objective-creation gate
	// (docs/architecture/departments.md). Unconditional
	// AutonomyApprovalRequired — no resource-instance condition, same
	// shape as "finance-agent-transfer-funds" below ("requires additional
	// authority" maps precisely onto REQUIRE_APPROVAL). There is no
	// numeric risk/magnitude field anywhere in Request/Rule to condition
	// on, and no per-instance state analogous to Slice 2's "is this
	// Workflow READY" exists for a not-yet-created Objective — every
	// proposal genuinely requires human sign-off before an Objective is
	// created, per departments.md's Objective creation gate.
	{RuleID: "objective-propose", Effect: policy.EffectPermit, Action: "objective.propose", Autonomy: policy.AutonomyApprovalRequired},

	// ROADMAP.md Phase 5 Slice 1's KnowledgeItem ingestion/versioning use
	// case (docs/architecture/knowledge.md steps 1-4). AUTOMATIC: capturing
	// a DRAFT candidate is not approving Knowledge — knowledge.md is
	// explicit that only knowledge.review/knowledge.approve (a later
	// slice) needs human review.
	{RuleID: "knowledge-item-capture", Effect: policy.EffectPermit, Action: "knowledge.item.capture", Autonomy: policy.AutonomyAutomatic},

	// ROADMAP.md Phase 5 Slice 2's knowledge-review-request governed
	// action — requesting that a KnowledgeItem candidate be reviewed.
	// Renamed 2026-08-24 from "knowledge.approve"
	// (docs/adr/ADR-0010-authority-model-formalization.md): the *request*
	// (any principal may ask for review) and the *decide* act (only a
	// human may actually approve/reject) are structurally different
	// operations that were sharing one confusingly-named action. This rule
	// governs only the request; the decide act is protected by
	// ResolveApproval's unconditional human-decider check
	// (internal/adapters/persistence/supabase/pending_repo.go), which
	// applies to every CommandType, not a per-action policy rule.
	// Unconditional AutonomyApprovalRequired, same shape as
	// "objective-propose" above — but unlike Objective's "no numeric field
	// to condition on" reasoning, this one is a permanent architectural
	// prohibition: docs/architecture/knowledge.md is explicit that
	// deterministic automatic approval is disabled until a dedicated ADR is
	// accepted, not merely a judgment call this slice made.
	{RuleID: "knowledge-review-request", Effect: policy.EffectPermit, Action: "knowledge.review.request", Autonomy: policy.AutonomyApprovalRequired},

	{RuleID: "research-agent-read-market-data", Effect: policy.EffectPermit, Role: "research_agent", Action: "research.read_market_data", Autonomy: policy.AutonomyAutomatic},
	{RuleID: "research-agent-create-report", Effect: policy.EffectPermit, Role: "research_agent", Action: "research.create_report", Autonomy: policy.AutonomyAutomatic},

	{RuleID: "finance-agent-read-financial-data", Effect: policy.EffectPermit, Role: "finance_agent", Action: "finance.read_financial_data", Autonomy: policy.AutonomyAutomatic},
	{RuleID: "finance-agent-create-payment-request", Effect: policy.EffectPermit, Role: "finance_agent", Action: "finance.create_payment_request", Autonomy: policy.AutonomyAutomatic},
	{RuleID: "finance-agent-transfer-funds", Effect: policy.EffectPermit, Role: "finance_agent", Action: "finance.transfer_funds", Autonomy: policy.AutonomyApprovalRequired},
}

// matchRule matches on Action/ActionPrefix as before, plus Role: a rule
// with a non-empty Role only matches a request asserting that exact role.
// A role-agnostic rule (Role == "") still matches every role, so every
// rule written before ADR-0008 is unaffected by this change.
func matchRule(role policy.Role, action string) (policy.Rule, bool) {
	for _, r := range firstSlicePolicies {
		if r.Role != "" && r.Role != role {
			continue
		}
		if r.Action != "" && r.Action == action {
			return r, true
		}
		if r.ActionPrefix != "" && len(action) >= len(r.ActionPrefix) && action[:len(r.ActionPrefix)] == r.ActionPrefix {
			return r, true
		}
	}
	return policy.Rule{}, false
}
