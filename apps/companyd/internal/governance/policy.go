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
