package governance

import (
	"context"
	"testing"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
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
			if decision.Outcome != policy.DecisionAutomatic {
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
	if decision.Outcome != policy.DecisionAutomatic {
		t.Errorf("outcome = %s, want ALLOW", decision.Outcome)
	}
}

func TestEvaluate_DefaultDenyForUnknownAction(t *testing.T) {
	decision, err := Evaluate(context.Background(), baseRequest("finance.approve.budget"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionDenied {
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
	if decision.Outcome != policy.DecisionDenied {
		t.Errorf("outcome = %s, want DENY (missing evidence)", decision.Outcome)
	}
}

// TestEvaluate_DeniesResourceOwnerMismatch is Phase 3 Slice 1's first
// concrete, HTTP-reachable DENY policy (ROADMAP.md): a Principal other
// than a resource's trusted owner (e.g. a Workflow's
// InitiatingPrincipalID) is denied, overriding the blanket "workflow.*"
// permit rule — this is a real Authority check (governance.md step 3),
// not a policy-rule match, so it fires before matchRule is even reached.
func TestEvaluate_DeniesResourceOwnerMismatch(t *testing.T) {
	req := baseRequest("workflow.cancel")
	owner := uuid.New() // different from req.PrincipalID
	req.ResourceOwnerPrincipalID = &owner

	decision, err := Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionDenied {
		t.Fatalf("outcome = %s, want DENY (requester does not own the resource)", decision.Outcome)
	}
}

// TestEvaluate_AllowsResourceOwnerMatch proves the ownership check does not
// block the legitimate case: the resource's own owner may still proceed to
// the normal policy-rule evaluation and reach ALLOW.
func TestEvaluate_AllowsResourceOwnerMatch(t *testing.T) {
	req := baseRequest("workflow.cancel")
	owner := req.PrincipalID
	req.ResourceOwnerPrincipalID = &owner

	decision, err := Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionAutomatic {
		t.Fatalf("outcome = %s, want ALLOW (requester owns the resource)", decision.Outcome)
	}
}

// TestEvaluate_AdditionalAutonomyRequirement_EscalatesToApprovalRequired is
// ROADMAP.md Phase 3 Slice 2's composition step (governance.md step 5:
// "human_only > approval_required > automatic"): a request whose matched
// Rule alone would ALLOW (workflow.cancel is under the blanket "workflow."
// permit) is escalated to REQUIRE_APPROVAL when the caller supplies a
// resource-instance requirement — without any ApprovalEvidence, it cannot
// proceed to ALLOW.
func TestEvaluate_AdditionalAutonomyRequirement_EscalatesToApprovalRequired(t *testing.T) {
	req := baseRequest("workflow.cancel")
	lvl := policy.AutonomyApprovalRequired
	req.AdditionalAutonomyRequirement = &lvl

	decision, err := Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionRequireApproval {
		t.Fatalf("outcome = %s, want REQUIRE_APPROVAL (escalated by resource-instance requirement)", decision.Outcome)
	}
	if decision.AutonomyLevel != policy.AutonomyApprovalRequired {
		t.Fatalf("autonomy level = %s, want APPROVAL_REQUIRED (composed, most-restrictive-wins)", decision.AutonomyLevel)
	}
}

// TestEvaluate_MatchingApprovalEvidence_UnlocksAllow is governance.md step
// 7's literal wording: "unless valid unconsumed approval evidence exactly
// matches the request."
func TestEvaluate_MatchingApprovalEvidence_UnlocksAllow(t *testing.T) {
	req := baseRequest("workflow.cancel")
	lvl := policy.AutonomyApprovalRequired
	req.AdditionalAutonomyRequirement = &lvl
	req.ApprovalEvidence = &ApprovalEvidence{ApprovalID: uuid.New(), ProposalDigest: req.ProposalDigest}

	decision, err := Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionAutomatic {
		t.Fatalf("outcome = %s, want ALLOW (matching approval evidence)", decision.Outcome)
	}
}

// TestEvaluate_MismatchedApprovalEvidence_StaysRequireApproval proves the
// evidence match is exact, not merely "some approval exists" — a digest
// for a different (or stale) request must not unlock ALLOW.
func TestEvaluate_MismatchedApprovalEvidence_StaysRequireApproval(t *testing.T) {
	req := baseRequest("workflow.cancel")
	lvl := policy.AutonomyApprovalRequired
	req.AdditionalAutonomyRequirement = &lvl
	req.ApprovalEvidence = &ApprovalEvidence{ApprovalID: uuid.New(), ProposalDigest: "a-different-digest"}

	decision, err := Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionRequireApproval {
		t.Fatalf("outcome = %s, want REQUIRE_APPROVAL (evidence digest does not match this request)", decision.Outcome)
	}
}

// TestEvaluate_AdditionalAutonomyRequirement_NeverWeakens proves
// composition is one-directional: a resource-instance requirement can only
// escalate, never relax, what the matched Rule alone would already
// require (human_only from the Rule table stays human_only even if the
// caller's additional requirement is merely approval_required).
func TestEvaluate_AdditionalAutonomyRequirement_NeverWeakens(t *testing.T) {
	req := agentRequest("finance_agent", "finance.transfer_funds") // Rule alone: APPROVAL_REQUIRED
	automatic := policy.AutonomyAutomatic
	req.AdditionalAutonomyRequirement = &automatic // weaker; must not downgrade the outcome

	decision, err := Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionRequireApproval {
		t.Fatalf("outcome = %s, want REQUIRE_APPROVAL (a weaker additional requirement must not relax the Rule's own)", decision.Outcome)
	}
}

// TestEvaluate_HumanOnly_HumanRequesterAllowed proves HUMAN_ONLY autonomy
// is real, not dead code (docs/adr/ADR-0010-authority-model-formalization.md
// — the prior three-value Decision model made this branch unconditionally
// DENY regardless of requester, since Request carried no principal-kind
// signal at all). No policy rule sets AutonomyHumanOnly yet, so this uses
// AdditionalAutonomyRequirement to synthesize it, same technique
// TestEvaluate_AdditionalAutonomyRequirement_EscalatesToApprovalRequired
// already uses for APPROVAL_REQUIRED.
func TestEvaluate_HumanOnly_HumanRequesterAllowed(t *testing.T) {
	req := baseRequest("workflow.cancel") // Rule alone: AUTOMATIC
	lvl := policy.AutonomyHumanOnly
	req.AdditionalAutonomyRequirement = &lvl
	req.RequestingPrincipalKind = principal.KindHuman

	decision, err := Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionHumanOnly {
		t.Fatalf("outcome = %s, want HUMAN_ONLY (eligible human requester)", decision.Outcome)
	}
	if decision.AutonomyLevel != policy.AutonomyHumanOnly {
		t.Fatalf("autonomy level = %s, want HUMAN_ONLY", decision.AutonomyLevel)
	}
}

// TestEvaluate_HumanOnly_NonHumanRequesterDenied is governance.md step 6's
// other half: "an agent or service request is DENY."
func TestEvaluate_HumanOnly_NonHumanRequesterDenied(t *testing.T) {
	req := baseRequest("workflow.cancel")
	lvl := policy.AutonomyHumanOnly
	req.AdditionalAutonomyRequirement = &lvl
	req.RequestingPrincipalKind = principal.KindService

	decision, err := Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionDenied {
		t.Fatalf("outcome = %s, want DENIED (service requester under human_only)", decision.Outcome)
	}
}

// TestEvaluate_HumanOnly_UnsetKindDenied proves a zero-value
// RequestingPrincipalKind fails closed rather than being silently treated
// as human — the same "a missing input never implies eligibility"
// discipline every other check in Evaluate already follows (evidence,
// resource ownership, exclusion).
func TestEvaluate_HumanOnly_UnsetKindDenied(t *testing.T) {
	req := baseRequest("workflow.cancel")
	lvl := policy.AutonomyHumanOnly
	req.AdditionalAutonomyRequirement = &lvl
	// req.RequestingPrincipalKind left at its zero value deliberately.

	decision, err := Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Outcome != policy.DecisionDenied {
		t.Fatalf("outcome = %s, want DENIED (unset requester kind must fail closed)", decision.Outcome)
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
