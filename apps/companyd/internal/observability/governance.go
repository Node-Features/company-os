package observability

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
)

// RecordGovernanceDecision logs and records a metric for one already-
// persisted Governance decision. Called from every governance-evaluation
// call site (Workflow, department-automatic, Objective proposal, Knowledge
// approval) right after SaveGovernanceDecision commits — never before,
// per observability.md's non-authority invariant. Sharing this helper
// across call sites is a side-effect-only convenience (the same kind of
// sharing those call sites already do via ports.AuthoritativeStateRepository.
// SaveGovernanceDecision itself), not the domain-control-flow
// generalization this codebase otherwise deliberately avoids across its
// governance wrappers.
func RecordGovernanceDecision(ctx context.Context, metrics ports.MetricsRecorder, decision policy.GovernanceDecision) {
	enriched := WithExecutionContext(ctx, ExecutionContext{
		CorrelationID:        decision.CorrelationID,
		OrganizationID:       decision.OrganizationID,
		PrincipalID:          decision.PrincipalID,
		GovernanceDecisionID: decision.DecisionID,
		GovernanceOutcome:    string(decision.Outcome),
	})

	args := []any{
		"action", decision.Action,
		"resource_type", decision.ResourceType,
		"resource_id", decision.ResourceID,
		"autonomy_level", string(decision.AutonomyLevel),
	}
	if decision.Reason != nil {
		args = append(args, "reason", *decision.Reason)
	}
	Logger(enriched).Info("governance decision", args...)

	if metrics == nil {
		return
	}
	metrics.IncrCounter("governance_decisions_total", map[string]string{
		"outcome": string(decision.Outcome),
		"action":  decision.Action,
	})
}
