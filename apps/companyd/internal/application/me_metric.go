package application

import (
	"context"
	"errors"
	"time"

	me "github.com/Node-Features/company-os/apps/companyd/internal/departments/monitoringevaluation"
	medomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/monitoringevaluation"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// RecordMetric is the RECORD_METRIC use case (ROADMAP.md Phase 4 Slice 2):
// computes this slice's one MetricDefinition ("did the Result succeed?")
// from an existing result.Result and persists it.
func (a *Application) RecordMetric(ctx context.Context, req RecordMetricRequest) Result {
	orgID := a.Fixtures.Organization().OrganizationID

	res, err := a.Exec.GetResult(ctx, orgID, req.ResultID)
	if errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Rejected, Reasons: []string{"result_not_found"}}
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	metricID := uuid.New()
	_, govResult, ok := a.evaluateAutomaticGovernance(ctx, req.RequestID, orgID, req.PrincipalID,
		"me.metric.record", "Metric", metricID.String(), metricID.String())
	if !ok {
		return govResult
	}

	m := &medomain.Metric{
		MetricID:       metricID,
		OrganizationID: orgID,
		SubjectID:      a.Fixtures.Capability().CapabilityDefinitionID,
		ResultID:       req.ResultID,
		Value:          me.ComputeSuccessMetric(*res),
		CreatedAt:      time.Now().UTC(),
	}
	if err := a.MonitoringEvaluation.RecordMetric(ctx, m); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return Result{Outcome: Accepted, ResourceID: &metricID}
}
