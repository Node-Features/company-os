package governance

import (
	"context"
	"testing"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/google/uuid"
)

func baseRequest(action string) Request {
	return Request{
		RequestID: uuid.New(), CorrelationID: uuid.New(), OrganizationID: uuid.New(), PrincipalID: uuid.New(),
		EvidencePresent: true, Action: action, ResourceType: "Workflow", ResourceID: uuid.New().String(),
		ProposalDigest: "digest", TrustedContextDigest: "ctx-digest",
	}
}

func TestEvaluate_AllowsWorkflowActions(t *testing.T) {
	for _, action := range []string{"workflow.create", "workflow.start", "workflow.result.accept", "workflow.result.reject"} {
		t.Run(action, func(t *testing.T) {
			decision, err := Evaluate(context.Background(), baseRequest(action))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if decision.Outcome != policy.DecisionAllow {
				t.Errorf("outcome = %s, want ALLOW", decision.Outcome)
			}
		})
	}
}

func TestEvaluate_AllowsCapabilityDispatch(t *testing.T) {
	decision, err := Evaluate(context.Background(), baseRequest("capability.intelligence.dispatch"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionAllow {
		t.Errorf("outcome = %s, want ALLOW", decision.Outcome)
	}
}

func TestEvaluate_DefaultDenyForUnknownAction(t *testing.T) {
	decision, err := Evaluate(context.Background(), baseRequest("finance.approve.budget"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionDeny {
		t.Errorf("outcome = %s, want DENY (default-deny)", decision.Outcome)
	}
}

func TestEvaluate_DenyWithoutEvidence(t *testing.T) {
	req := baseRequest("workflow.create")
	req.EvidencePresent = false
	decision, err := Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionDeny {
		t.Errorf("outcome = %s, want DENY (missing evidence)", decision.Outcome)
	}
}

func TestEvaluate_PersistsPolicyVersionAndDigests(t *testing.T) {
	req := baseRequest("workflow.create")
	decision, err := Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.PolicyVersion != PolicyVersion {
		t.Errorf("policy version = %q, want %q", decision.PolicyVersion, PolicyVersion)
	}
	if decision.ProposalDigest != req.ProposalDigest {
		t.Errorf("proposal digest not carried through to decision")
	}
}
