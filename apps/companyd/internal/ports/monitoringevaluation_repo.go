package ports

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/monitoringevaluation"
	"github.com/google/uuid"
)

// MonitoringEvaluationRepository persists M&E's core contracts
// (ROADMAP.md Phase 4 Slice 2).
type MonitoringEvaluationRepository interface {
	RecordMetric(ctx context.Context, m *monitoringevaluation.Metric) error
	GetMetricsByIDs(ctx context.Context, organizationID uuid.UUID, metricIDs []uuid.UUID) ([]monitoringevaluation.Metric, error)

	CreateEvaluation(ctx context.Context, e *monitoringevaluation.Evaluation) error
	GetEvaluation(ctx context.Context, organizationID, evaluationID uuid.UUID) (monitoringevaluation.Evaluation, error)

	UpsertPerformanceProfile(ctx context.Context, p *monitoringevaluation.PerformanceProfile) error
	GetPerformanceProfile(ctx context.Context, organizationID, subjectID uuid.UUID) (monitoringevaluation.PerformanceProfile, error)
}
