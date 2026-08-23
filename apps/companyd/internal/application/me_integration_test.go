package application

import (
	"context"
	"testing"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/google/uuid"
)

// TestIntegration_MonitoringEvaluation_ResultToPerformanceProfile proves
// ROADMAP.md Phase 4 Slice 2's full Result -> Metric -> Evaluation ->
// PerformanceProfile chain end to end against the real database: 4
// succeeded Results and 1 failed Result (4/5 = 0.8, exactly the PASS
// threshold) each become a Metric, RunEvaluation classifies them PASS, and
// the subject's PerformanceProfile reflects it.
func TestIntegration_MonitoringEvaluation_ResultToPerformanceProfile(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New() // stands in for a real signed-in HumanPrincipal

	outcomes := []result.Outcome{result.OutcomeSucceeded, result.OutcomeSucceeded, result.OutcomeSucceeded, result.OutcomeSucceeded, result.OutcomeFailed}
	metricIDs := make([]uuid.UUID, 0, len(outcomes))

	for _, outcome := range outcomes {
		_, _ = startedReadyWorkflow(t, app)
		_, resultID := submitFakeResult(t, app, outcome)

		metric := app.RecordMetric(ctx, RecordMetricRequest{
			RequestID:   uuid.New(),
			PrincipalID: principalID,
			ResultID:    resultID,
		})
		if metric.Outcome != Accepted || metric.ResourceID == nil {
			t.Fatalf("RecordMetric(%s) outcome = %s (reasons: %v)", outcome, metric.Outcome, metric.Reasons)
		}
		metricIDs = append(metricIDs, *metric.ResourceID)
	}

	evaluation := app.RunEvaluation(ctx, RunEvaluationRequest{
		RequestID:   uuid.New(),
		PrincipalID: principalID,
		MetricIDs:   metricIDs,
	})
	if evaluation.Outcome != Accepted || evaluation.ResourceID == nil {
		t.Fatalf("RunEvaluation outcome = %s (reasons: %v)", evaluation.Outcome, evaluation.Reasons)
	}

	subjectID := app.Fixtures.Capability().CapabilityDefinitionID
	profile, err := app.GetPerformanceProfile(ctx, subjectID)
	if err != nil {
		t.Fatalf("GetPerformanceProfile: %v", err)
	}
	if profile.LatestEvaluationID != *evaluation.ResourceID {
		t.Fatalf("profile.LatestEvaluationID = %s, want %s", profile.LatestEvaluationID, *evaluation.ResourceID)
	}
	if profile.SuccessRate != 0.8 {
		t.Fatalf("profile.SuccessRate = %v, want 0.8", profile.SuccessRate)
	}
	if string(profile.Outcome) != "PASS" {
		t.Fatalf("profile.Outcome = %s, want PASS", profile.Outcome)
	}
}

// TestIntegration_RunEvaluation_InsufficientSamplesInconclusive proves the
// INCONCLUSIVE floor against the real database: fewer than the minimum
// sample size never gets scored as PASS or FAIL.
func TestIntegration_RunEvaluation_InsufficientSamplesInconclusive(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	_, _ = startedReadyWorkflow(t, app)
	_, resultID := submitFakeResult(t, app, result.OutcomeSucceeded)

	metric := app.RecordMetric(ctx, RecordMetricRequest{
		RequestID: uuid.New(), PrincipalID: principalID, ResultID: resultID,
	})
	if metric.Outcome != Accepted || metric.ResourceID == nil {
		t.Fatalf("RecordMetric outcome = %s (reasons: %v)", metric.Outcome, metric.Reasons)
	}

	evaluation := app.RunEvaluation(ctx, RunEvaluationRequest{
		RequestID: uuid.New(), PrincipalID: principalID, MetricIDs: []uuid.UUID{*metric.ResourceID},
	})
	if evaluation.Outcome != Accepted || evaluation.ResourceID == nil {
		t.Fatalf("RunEvaluation outcome = %s (reasons: %v)", evaluation.Outcome, evaluation.Reasons)
	}

	got, err := app.MonitoringEvaluation.GetEvaluation(ctx, app.Fixtures.Organization().OrganizationID, *evaluation.ResourceID)
	if err != nil {
		t.Fatalf("GetEvaluation: %v", err)
	}
	if string(got.Outcome) != "INCONCLUSIVE" {
		t.Fatalf("Evaluation.Outcome = %s, want INCONCLUSIVE (only 1 sample)", got.Outcome)
	}
}
