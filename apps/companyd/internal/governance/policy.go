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
var firstSlicePolicies = []policy.Rule{
	{RuleID: "workflow-actions", Effect: policy.EffectPermit, ActionPrefix: "workflow.", Autonomy: policy.AutonomyAutomatic},
	{RuleID: "capability-dispatch", Effect: policy.EffectPermit, Action: "capability.intelligence.dispatch", Autonomy: policy.AutonomyAutomatic},
}

func matchRule(action string) (policy.Rule, bool) {
	for _, r := range firstSlicePolicies {
		if r.Action != "" && r.Action == action {
			return r, true
		}
		if r.ActionPrefix != "" && len(action) >= len(r.ActionPrefix) && action[:len(r.ActionPrefix)] == r.ActionPrefix {
			return r, true
		}
	}
	return policy.Rule{}, false
}
