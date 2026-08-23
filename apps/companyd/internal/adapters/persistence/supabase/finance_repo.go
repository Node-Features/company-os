package supabase

import (
	"context"
	"errors"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/finance"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// FinanceRepository implements ports.FinanceRepository.
type FinanceRepository struct{ p *Pool }

func NewFinanceRepository(p *Pool) *FinanceRepository {
	return &FinanceRepository{p: p}
}

func (r *FinanceRepository) CreateBudget(ctx context.Context, b *finance.Budget) error {
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO budgets (budget_id, organization_id, subject_id, limit_amount, currency, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		b.BudgetID, b.OrganizationID, b.SubjectID, b.LimitAmount, b.Currency, b.CreatedAt)
	return err
}

func (r *FinanceRepository) GetBudgetBySubject(ctx context.Context, organizationID, subjectID uuid.UUID) (finance.Budget, error) {
	var b finance.Budget
	var currency string
	err := r.p.pool.QueryRow(ctx, `
		SELECT budget_id, organization_id, subject_id, limit_amount, currency, created_at
		FROM budgets WHERE organization_id=$1 AND subject_id=$2
		ORDER BY created_at DESC LIMIT 1`,
		organizationID, subjectID,
	).Scan(&b.BudgetID, &b.OrganizationID, &b.SubjectID, &b.LimitAmount, &currency, &b.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return finance.Budget{}, ports.ErrNotFound
	}
	if err != nil {
		return finance.Budget{}, err
	}
	b.Currency = finance.Currency(currency)
	return b, nil
}

func (r *FinanceRepository) CreateCostConstraint(ctx context.Context, c *finance.CostConstraint) error {
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO cost_constraints (constraint_id, organization_id, budget_id, subject_id, max_cost, currency, enforcement_mode, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		c.ConstraintID, c.OrganizationID, c.BudgetID, c.SubjectID, c.MaxCost, c.Currency, c.EnforcementMode, c.CreatedAt)
	return err
}

func (r *FinanceRepository) GetCostConstraintBySubject(ctx context.Context, organizationID, subjectID uuid.UUID) (finance.CostConstraint, error) {
	var c finance.CostConstraint
	var currency, mode string
	err := r.p.pool.QueryRow(ctx, `
		SELECT constraint_id, organization_id, budget_id, subject_id, max_cost, currency, enforcement_mode, created_at
		FROM cost_constraints WHERE organization_id=$1 AND subject_id=$2
		ORDER BY created_at DESC LIMIT 1`,
		organizationID, subjectID,
	).Scan(&c.ConstraintID, &c.OrganizationID, &c.BudgetID, &c.SubjectID, &c.MaxCost, &currency, &mode, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return finance.CostConstraint{}, ports.ErrNotFound
	}
	if err != nil {
		return finance.CostConstraint{}, err
	}
	c.Currency = finance.Currency(currency)
	c.EnforcementMode = finance.EnforcementMode(mode)
	return c, nil
}

func (r *FinanceRepository) GetPriceProfile(ctx context.Context, providerAdapter string) (finance.PriceProfile, error) {
	var p finance.PriceProfile
	var currency string
	err := r.p.pool.QueryRow(ctx, `
		SELECT price_profile_id, provider_adapter, model_id, currency, input_price_per_k_tokens, output_price_per_k_tokens, effective_at
		FROM price_profiles WHERE provider_adapter=$1`,
		providerAdapter,
	).Scan(&p.PriceProfileID, &p.ProviderAdapter, &p.ModelID, &currency, &p.InputPricePerKTokens, &p.OutputPricePerKTokens, &p.EffectiveAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return finance.PriceProfile{}, ports.ErrNotFound
	}
	if err != nil {
		return finance.PriceProfile{}, err
	}
	p.Currency = finance.Currency(currency)
	return p, nil
}

func (r *FinanceRepository) RecordResourceUsage(ctx context.Context, u *finance.ResourceUsage) error {
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO resource_usage (usage_id, organization_id, result_id, subject_id, provider_adapter, model_id, input_tokens, output_tokens, cost, currency, succeeded, measurement_method, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		u.UsageID, u.OrganizationID, u.ResultID, u.SubjectID, u.ProviderAdapter, u.ModelID, u.InputTokens, u.OutputTokens, u.Cost, u.Currency, u.Succeeded, u.MeasurementMethod, u.CreatedAt)
	return err
}

func (r *FinanceRepository) GetResourceUsageBySubject(ctx context.Context, organizationID, subjectID uuid.UUID) ([]finance.ResourceUsage, error) {
	rows, err := r.p.pool.Query(ctx, `
		SELECT usage_id, organization_id, result_id, subject_id, provider_adapter, model_id, input_tokens, output_tokens, cost, currency, succeeded, measurement_method, created_at
		FROM resource_usage WHERE organization_id=$1 AND subject_id=$2`,
		organizationID, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []finance.ResourceUsage
	for rows.Next() {
		var u finance.ResourceUsage
		var currency, method string
		if err := rows.Scan(&u.UsageID, &u.OrganizationID, &u.ResultID, &u.SubjectID, &u.ProviderAdapter, &u.ModelID, &u.InputTokens, &u.OutputTokens, &u.Cost, &currency, &u.Succeeded, &method, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.Currency = finance.Currency(currency)
		u.MeasurementMethod = finance.MeasurementMethod(method)
		list = append(list, u)
	}
	return list, rows.Err()
}

func (r *FinanceRepository) CreateResourceEvaluation(ctx context.Context, e *finance.ResourceEvaluation) error {
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO resource_evaluations (evaluation_id, organization_id, subject_id, total_cost, currency, total_count, successful_count, effective_cost_per_successful_result, budget_limit_amount, budget_variance, me_performance_outcome, me_success_rate, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		e.EvaluationID, e.OrganizationID, e.SubjectID, e.TotalCost, e.Currency, e.TotalCount, e.SuccessfulCount, e.EffectiveCostPerSuccessfulResult, e.BudgetLimitAmount, e.BudgetVariance, e.MEPerformanceOutcome, e.MESuccessRate, e.Status, e.CreatedAt)
	return err
}

func (r *FinanceRepository) GetResourceEvaluation(ctx context.Context, organizationID, evaluationID uuid.UUID) (finance.ResourceEvaluation, error) {
	var e finance.ResourceEvaluation
	var currency, status string
	err := r.p.pool.QueryRow(ctx, `
		SELECT evaluation_id, organization_id, subject_id, total_cost, currency, total_count, successful_count, effective_cost_per_successful_result, budget_limit_amount, budget_variance, me_performance_outcome, me_success_rate, status, created_at
		FROM resource_evaluations WHERE organization_id=$1 AND evaluation_id=$2`,
		organizationID, evaluationID,
	).Scan(&e.EvaluationID, &e.OrganizationID, &e.SubjectID, &e.TotalCost, &currency, &e.TotalCount, &e.SuccessfulCount, &e.EffectiveCostPerSuccessfulResult, &e.BudgetLimitAmount, &e.BudgetVariance, &e.MEPerformanceOutcome, &e.MESuccessRate, &status, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return finance.ResourceEvaluation{}, ports.ErrNotFound
	}
	if err != nil {
		return finance.ResourceEvaluation{}, err
	}
	e.Currency = finance.Currency(currency)
	e.Status = finance.EvaluationStatus(status)
	return e, nil
}

var _ ports.FinanceRepository = (*FinanceRepository)(nil)
