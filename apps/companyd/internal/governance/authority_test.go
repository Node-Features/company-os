package governance

import (
	"context"
	"testing"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
)

// agentRequest extends baseRequest (evaluate_test.go) with a Role, for the
// research_agent/finance_agent illustrative rules ADR-0008 added to
// firstSlicePolicies.
func agentRequest(role policy.Role, action string) Request {
	req := baseRequest(action)
	req.Role = role
	return req
}

// TestAuthorize_AllowedAction_Succeeds: a Research Agent reading market
// data is the original design table's allowed case.
func TestAuthorize_AllowedAction_Succeeds(t *testing.T) {
	decision, err := Evaluate(context.Background(), agentRequest("research_agent", "research.read_market_data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionAllow {
		t.Fatalf("outcome = %s, want ALLOW", decision.Outcome)
	}
}

// TestAuthorize_DeniedAction_Fails: a Research Agent has no rule granting
// finance.transfer_funds — the wrong role for this action, not merely a
// missing rule — and must not fall back to some other role's grant.
func TestAuthorize_DeniedAction_Fails(t *testing.T) {
	decision, err := Evaluate(context.Background(), agentRequest("research_agent", "finance.transfer_funds"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionDeny {
		t.Fatalf("outcome = %s, want DENY (wrong role for this action)", decision.Outcome)
	}
}

// TestAuthorize_UnknownSubject_FailsClosed: a role matching no rule at all
// — the equivalent of an unrecognized subject — must deny, not fall
// through to some implicit "unrestricted" default. Deliberately requests
// an action (research.read_market_data) that IS granted to a different
// role, to prove the denial is role-scoped, not just "no rule exists for
// this action at all" (that case is already covered by
// TestEvaluate_DefaultDenyForUnknownAction in evaluate_test.go).
func TestAuthorize_UnknownSubject_FailsClosed(t *testing.T) {
	decision, err := Evaluate(context.Background(), agentRequest("unregistered_agent", "research.read_market_data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionDeny {
		t.Fatalf("outcome = %s, want DENY (unknown role fails closed)", decision.Outcome)
	}
}

// TestAuthorize_FinanceAgent_TransferFunds_RequiresApproval demonstrates
// why this model is more precise than the binary allowed/denied table it
// was asked to match: "transfer_funds denied without additional authority"
// maps exactly onto REQUIRE_APPROVAL, a third outcome distinct from both
// ALLOW and DENY — governance.md: "not a weak allow and must never be sent
// to an executor." A caller that treats REQUIRE_APPROVAL as either ALLOW
// or DENY has misread the decision.
func TestAuthorize_FinanceAgent_TransferFunds_RequiresApproval(t *testing.T) {
	decision, err := Evaluate(context.Background(), agentRequest("finance_agent", "finance.transfer_funds"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionRequireApproval {
		t.Fatalf("outcome = %s, want REQUIRE_APPROVAL (eligible, but no approval evidence yet)", decision.Outcome)
	}
}

// The remaining two tests round out the original design table's two agent
// types with their other listed rows.

func TestAuthorize_FinanceAgent_ReadFinancialData_Succeeds(t *testing.T) {
	decision, err := Evaluate(context.Background(), agentRequest("finance_agent", "finance.read_financial_data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionAllow {
		t.Fatalf("outcome = %s, want ALLOW", decision.Outcome)
	}
}

func TestAuthorize_ResearchAgent_SendCustomerMessage_Denied(t *testing.T) {
	decision, err := Evaluate(context.Background(), agentRequest("research_agent", "customer.send_message"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionDeny {
		t.Fatalf("outcome = %s, want DENY (no rule grants this to research_agent)", decision.Outcome)
	}
}
