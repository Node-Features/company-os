package supabase

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/monitoringevaluation"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MonitoringEvaluationRepository implements ports.MonitoringEvaluationRepository.
type MonitoringEvaluationRepository struct{ p *Pool }

func NewMonitoringEvaluationRepository(p *Pool) *MonitoringEvaluationRepository {
	return &MonitoringEvaluationRepository{p: p}
}

func (r *MonitoringEvaluationRepository) RecordMetric(ctx context.Context, m *monitoringevaluation.Metric) error {
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO metrics (metric_id, organization_id, subject_id, result_id, value, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		m.MetricID, m.OrganizationID, m.SubjectID, m.ResultID, m.Value, m.CreatedAt)
	return err
}

func (r *MonitoringEvaluationRepository) GetMetricsByIDs(ctx context.Context, organizationID uuid.UUID, metricIDs []uuid.UUID) ([]monitoringevaluation.Metric, error) {
	if len(metricIDs) == 0 {
		return nil, nil
	}
	// []string cast to ::uuid[], not a []uuid.UUID param directly — pgx v5's
	// array codec doesn't reliably cover a type that only satisfies
	// database/sql's Scanner/Valuer (google/uuid.UUID's scalar fallback
	// path), the same reasoning findings.evidence_ids's jsonb column
	// sidesteps elsewhere in this schema.
	ids := make([]string, len(metricIDs))
	for i, id := range metricIDs {
		ids[i] = id.String()
	}
	rows, err := r.p.pool.Query(ctx, `
		SELECT metric_id, organization_id, subject_id, result_id, value, created_at
		FROM metrics WHERE organization_id=$1 AND metric_id = ANY($2::uuid[])`,
		organizationID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []monitoringevaluation.Metric
	for rows.Next() {
		var m monitoringevaluation.Metric
		if err := rows.Scan(&m.MetricID, &m.OrganizationID, &m.SubjectID, &m.ResultID, &m.Value, &m.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func (r *MonitoringEvaluationRepository) CreateEvaluation(ctx context.Context, e *monitoringevaluation.Evaluation) error {
	metricIDs, err := json.Marshal(e.MetricIDs)
	if err != nil {
		return err
	}
	_, err = r.p.pool.Exec(ctx, `
		INSERT INTO evaluations (evaluation_id, organization_id, subject_id, metric_ids, outcome, success_rate, sample_size, independence, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		e.EvaluationID, e.OrganizationID, e.SubjectID, metricIDs, e.Outcome, e.SuccessRate, e.SampleSize, e.Independence, e.CreatedAt)
	return err
}

func (r *MonitoringEvaluationRepository) GetEvaluation(ctx context.Context, organizationID, evaluationID uuid.UUID) (monitoringevaluation.Evaluation, error) {
	var e monitoringevaluation.Evaluation
	var outcome, independence string
	var metricIDs []byte
	err := r.p.pool.QueryRow(ctx, `
		SELECT evaluation_id, organization_id, subject_id, metric_ids, outcome, success_rate, sample_size, independence, created_at
		FROM evaluations WHERE organization_id=$1 AND evaluation_id=$2`,
		organizationID, evaluationID,
	).Scan(&e.EvaluationID, &e.OrganizationID, &e.SubjectID, &metricIDs, &outcome, &e.SuccessRate, &e.SampleSize, &independence, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return monitoringevaluation.Evaluation{}, ports.ErrNotFound
	}
	if err != nil {
		return monitoringevaluation.Evaluation{}, err
	}
	e.Outcome = monitoringevaluation.EvaluationOutcome(outcome)
	e.Independence = monitoringevaluation.Independence(independence)
	if err := json.Unmarshal(metricIDs, &e.MetricIDs); err != nil {
		return monitoringevaluation.Evaluation{}, err
	}
	return e, nil
}

func (r *MonitoringEvaluationRepository) UpsertPerformanceProfile(ctx context.Context, p *monitoringevaluation.PerformanceProfile) error {
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO performance_profiles (subject_id, organization_id, latest_evaluation_id, outcome, success_rate, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (subject_id) DO UPDATE SET
			latest_evaluation_id = EXCLUDED.latest_evaluation_id,
			outcome = EXCLUDED.outcome,
			success_rate = EXCLUDED.success_rate,
			updated_at = EXCLUDED.updated_at`,
		p.SubjectID, p.OrganizationID, p.LatestEvaluationID, p.Outcome, p.SuccessRate, p.UpdatedAt)
	return err
}

func (r *MonitoringEvaluationRepository) GetPerformanceProfile(ctx context.Context, organizationID, subjectID uuid.UUID) (monitoringevaluation.PerformanceProfile, error) {
	var p monitoringevaluation.PerformanceProfile
	var outcome string
	err := r.p.pool.QueryRow(ctx, `
		SELECT subject_id, organization_id, latest_evaluation_id, outcome, success_rate, updated_at
		FROM performance_profiles WHERE organization_id=$1 AND subject_id=$2`,
		organizationID, subjectID,
	).Scan(&p.SubjectID, &p.OrganizationID, &p.LatestEvaluationID, &outcome, &p.SuccessRate, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return monitoringevaluation.PerformanceProfile{}, ports.ErrNotFound
	}
	if err != nil {
		return monitoringevaluation.PerformanceProfile{}, err
	}
	p.Outcome = monitoringevaluation.EvaluationOutcome(outcome)
	return p, nil
}

var _ ports.MonitoringEvaluationRepository = (*MonitoringEvaluationRepository)(nil)
