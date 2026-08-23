// This file is M&E's Kernel-equivalent (docs/workflows patterns established
// by Phase 4 Slice 1's internal/departments/research): pure functions only.
// Orchestration (Governance evaluation, persistence) lives in
// internal/application/me_*.go.
package monitoringevaluation

import (
	domain "github.com/Node-Features/company-os/apps/companyd/internal/domain/monitoringevaluation"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
)

// ComputeSuccessMetric is this slice's one MetricDefinition: did the Result
// succeed? 1.0/0.0 — a binary success signal, not a graded quality score.
func ComputeSuccessMetric(res result.Result) float64 {
	if res.Outcome == result.OutcomeSucceeded {
		return 1.0
	}
	return 0.0
}

// minimumSampleSize is the fewer-than-this-many-samples -> INCONCLUSIVE
// threshold (monitoring-evaluation.md: "insufficient samples... produce
// INCONCLUSIVE, with the precise evidence gap preserved"). An
// implementation-detail default, not a resolved answer to the doc's own
// open question on minimum sample requirements.
const minimumSampleSize = 3

// passThreshold is the observed-success-rate cutoff for PASS vs FAIL. Also
// an implementation-detail default, not doctrine.
const passThreshold = 0.8

// ClassifyEvaluation computes an outcome from a set of Metric values (each
// 0.0 or 1.0, per ComputeSuccessMetric). Never assumes success from
// insufficient evidence.
func ClassifyEvaluation(values []float64) (outcome domain.EvaluationOutcome, successRate float64) {
	if len(values) < minimumSampleSize {
		return domain.EvaluationInconclusive, 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	successRate = sum / float64(len(values))
	if successRate >= passThreshold {
		return domain.EvaluationPass, successRate
	}
	return domain.EvaluationFail, successRate
}
