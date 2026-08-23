package monitoringevaluation

import (
	"testing"

	domain "github.com/Node-Features/company-os/apps/companyd/internal/domain/monitoringevaluation"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
)

func TestComputeSuccessMetric(t *testing.T) {
	if v := ComputeSuccessMetric(result.Result{Outcome: result.OutcomeSucceeded}); v != 1.0 {
		t.Fatalf("SUCCEEDED metric = %v, want 1.0", v)
	}
	for _, outcome := range []result.Outcome{result.OutcomeFailed, result.OutcomeTimedOut, result.OutcomePartial, result.OutcomeCancelled, result.OutcomeIndeterminate} {
		if v := ComputeSuccessMetric(result.Result{Outcome: outcome}); v != 0.0 {
			t.Fatalf("%s metric = %v, want 0.0", outcome, v)
		}
	}
}

func TestClassifyEvaluation_InsufficientSamples(t *testing.T) {
	for _, values := range [][]float64{{}, {1.0}, {1.0, 1.0}} {
		outcome, rate := ClassifyEvaluation(values)
		if outcome != domain.EvaluationInconclusive {
			t.Fatalf("ClassifyEvaluation(%v) outcome = %s, want INCONCLUSIVE", values, outcome)
		}
		if rate != 0 {
			t.Fatalf("ClassifyEvaluation(%v) successRate = %v, want 0 for INCONCLUSIVE", values, rate)
		}
	}
}

func TestClassifyEvaluation_Pass(t *testing.T) {
	// 4/5 = 0.8, exactly at the pass threshold.
	outcome, rate := ClassifyEvaluation([]float64{1, 1, 1, 1, 0})
	if outcome != domain.EvaluationPass {
		t.Fatalf("outcome = %s, want PASS", outcome)
	}
	if rate != 0.8 {
		t.Fatalf("successRate = %v, want 0.8", rate)
	}
}

func TestClassifyEvaluation_Fail(t *testing.T) {
	// 2/5 = 0.4, below the pass threshold, but still >= 3 samples.
	outcome, rate := ClassifyEvaluation([]float64{1, 1, 0, 0, 0})
	if outcome != domain.EvaluationFail {
		t.Fatalf("outcome = %s, want FAIL", outcome)
	}
	if rate != 0.4 {
		t.Fatalf("successRate = %v, want 0.4", rate)
	}
}
