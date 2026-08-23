// This file is Finance's Kernel-equivalent (pattern established by Phase 4
// Slice 1's internal/departments/research and Slice 2's
// internal/departments/monitoringevaluation): pure functions only.
// Orchestration (Governance evaluation, persistence) lives in
// internal/application/finance_*.go.
package finance

import domain "github.com/Node-Features/company-os/apps/companyd/internal/domain/finance"

// ComputeUsageCost prices a Result's token usage against a PriceProfile.
func ComputeUsageCost(p domain.PriceProfile, inputTokens, outputTokens int) float64 {
	return float64(inputTokens)/1000*p.InputPricePerKTokens + float64(outputTokens)/1000*p.OutputPricePerKTokens
}

// warningThresholdRatio is the cumulative-cost-as-fraction-of-MaxCost
// cutoff for APPROACHING_LIMIT vs WITHIN_LIMIT. An implementation-detail
// default, not doctrine — same convention as M&E's passThreshold.
const warningThresholdRatio = 0.8

// EvaluateConstraintOutcome computes resource.md's enforcement outcome for
// one subject's cumulative cost against its CostConstraint. A nil
// constraint (none created yet for the subject) is NOT_APPLICABLE, not an
// error — this slice never blocks usage recording on constraint absence.
func EvaluateConstraintOutcome(cumulativeCost float64, constraint *domain.CostConstraint) domain.ConstraintOutcome {
	if constraint == nil {
		return domain.NotApplicable
	}
	switch {
	case cumulativeCost > constraint.MaxCost:
		return domain.LimitExceeded
	case cumulativeCost >= constraint.MaxCost*warningThresholdRatio:
		return domain.ApproachingLimit
	default:
		return domain.WithinLimit
	}
}

// ComputeEffectiveCostPerSuccess returns totalCost divided by
// successfulCount, or nil when there are zero successes — resource.md:
// missing or indeterminate usage/outcome evidence produces explicit
// uncertainty, never a fabricated zero or infinite value.
func ComputeEffectiveCostPerSuccess(totalCost float64, successfulCount int) *float64 {
	if successfulCount == 0 {
		return nil
	}
	v := totalCost / float64(successfulCount)
	return &v
}
