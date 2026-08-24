package policy

import "testing"

// TestDecision_Allows is the single place runtime.Runtime.execute and
// kernel/workflow.verifyAllow's "may this proceed" gate is defined —
// AUTOMATIC and HUMAN_ONLY both proceed, REQUIRE_APPROVAL and DENIED do
// not (docs/adr/ADR-0010-authority-model-formalization.md).
func TestDecision_Allows(t *testing.T) {
	cases := []struct {
		decision Decision
		want     bool
	}{
		{DecisionAutomatic, true},
		{DecisionHumanOnly, true},
		{DecisionRequireApproval, false},
		{DecisionDenied, false},
		{Decision("ALLOW"), false}, // legacy historical spelling is not treated as a live proceed value
	}
	for _, c := range cases {
		if got := c.decision.Allows(); got != c.want {
			t.Errorf("Decision(%q).Allows() = %v, want %v", c.decision, got, c.want)
		}
	}
}
