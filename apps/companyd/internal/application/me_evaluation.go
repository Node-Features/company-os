package application

import (
	"context"
	"time"

	me "github.com/Node-Features/company-os/apps/companyd/internal/departments/monitoringevaluation"
	medomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/monitoringevaluation"
	"github.com/google/uuid"
)

// RunEvaluation is the RUN_EVALUATION use case (ROADMAP.md Phase 4 Slice 2):
// classifies a set of Metrics into PASS/FAIL/INCONCLUSIVE and upserts the
// subject's PerformanceProfile with the result.
func (a *Application) RunEvaluation(ctx context.Context, req RunEvaluationRequest) Result {
	orgID := a.Fixtures.Organization().OrganizationID

	if len(req.MetricIDs) == 0 {
		return Result{Outcome: Rejected, Reasons: []string{"no_metrics_given"}}
	}

	metrics, err := a.MonitoringEvaluation.GetMetricsByIDs(ctx, orgID, req.MetricIDs)
	if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}
	if len(metrics) != len(req.MetricIDs) {
		return Result{Outcome: Rejected, Reasons: []string{"one_or_more_metrics_not_found"}}
	}

	values := make([]float64, len(metrics))
	var subjectID uuid.UUID
	for i, m := range metrics {
		values[i] = m.Value
		subjectID = m.SubjectID // every Metric this slice shares the one CapabilityDefinition subject
	}
	outcome, successRate := me.ClassifyEvaluation(values)

	evaluationID := uuid.New()
	_, govResult, ok := a.evaluateAutomaticGovernance(ctx, req.RequestID, orgID, req.PrincipalID,
		"me.evaluation.run", "Evaluation", evaluationID.String(), evaluationID.String())
	if !ok {
		return govResult
	}

	now := time.Now().UTC()
	eval := &medomain.Evaluation{
		EvaluationID:   evaluationID,
		OrganizationID: orgID,
		SubjectID:      subjectID,
		MetricIDs:      req.MetricIDs,
		Outcome:        outcome,
		SuccessRate:    successRate,
		SampleSize:     len(metrics),
		Independence:   medomain.IndependenceSelfReported,
		CreatedAt:      now,
	}
	if err := a.MonitoringEvaluation.CreateEvaluation(ctx, eval); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	profile := &medomain.PerformanceProfile{
		SubjectID:          subjectID,
		OrganizationID:     orgID,
		LatestEvaluationID: evaluationID,
		Outcome:            outcome,
		SuccessRate:        successRate,
		UpdatedAt:          now,
	}
	if err := a.MonitoringEvaluation.UpsertPerformanceProfile(ctx, profile); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return Result{Outcome: Accepted, ResourceID: &evaluationID}
}

// GetPerformanceProfile is the read projection served by
// GET /v1/me/subjects/{subjectId}/performance-profile.
func (a *Application) GetPerformanceProfile(ctx context.Context, subjectID uuid.UUID) (medomain.PerformanceProfile, error) {
	orgID := a.Fixtures.Organization().OrganizationID
	return a.MonitoringEvaluation.GetPerformanceProfile(ctx, orgID, subjectID)
}
