package application

import (
	"context"
	"errors"
	"time"

	findept "github.com/Node-Features/company-os/apps/companyd/internal/departments/finance"
	findomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/finance"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// RunResourceEvaluation is the RUN_RESOURCE_EVALUATION use case
// (ROADMAP.md Phase 4 Slice 3): sums the subject's ResourceUsage into a
// total cost and effective cost per successful result — failed/retried
// attempts are included, per finance.md's "must not count only successful
// calls" invariant — records budget variance if a Budget exists, and reads
// M&E's PerformanceProfile (Phase 4 Slice 2) for the same subject as
// informational cross-department evidence (finance.md's "Boundary with
// M&E": "Finance consumes M&E outcome evidence without rewriting it").
// This is the first real cross-department read in the codebase — Finance
// does not require it to exist; its absence never blocks this evaluation.
func (a *Application) RunResourceEvaluation(ctx context.Context, req RunResourceEvaluationRequest) Result {
	orgID := a.Fixtures.Organization().OrganizationID
	subjectID := a.Fixtures.Capability().CapabilityDefinitionID

	usages, err := a.Finance.GetResourceUsageBySubject(ctx, orgID, subjectID)
	if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}
	if len(usages) == 0 {
		return Result{Outcome: Rejected, Reasons: []string{"no_resource_usage_recorded"}}
	}

	var totalCost float64
	var successfulCount int
	for _, u := range usages {
		totalCost += u.Cost
		if u.Succeeded {
			successfulCount++
		}
	}

	effectiveCost := findept.ComputeEffectiveCostPerSuccess(totalCost, successfulCount)
	status := findomain.EvaluationComputed
	if effectiveCost == nil {
		status = findomain.EvaluationInconclusive
	}

	var budgetLimit, budgetVariance *float64
	budget, err := a.Finance.GetBudgetBySubject(ctx, orgID, subjectID)
	if err == nil {
		limit := budget.LimitAmount
		variance := totalCost - budget.LimitAmount
		budgetLimit, budgetVariance = &limit, &variance
	} else if !errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	var mePerformanceOutcome *string
	var meSuccessRate *float64
	profile, err := a.MonitoringEvaluation.GetPerformanceProfile(ctx, orgID, subjectID)
	if err == nil {
		outcome := string(profile.Outcome)
		rate := profile.SuccessRate
		mePerformanceOutcome, meSuccessRate = &outcome, &rate
	} else if !errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	evaluationID := uuid.New()
	_, govResult, ok := a.evaluateAutomaticGovernance(ctx, req.RequestID, orgID, req.PrincipalID,
		"finance.evaluation.run", "ResourceEvaluation", evaluationID.String(), evaluationID.String())
	if !ok {
		return govResult
	}

	e := &findomain.ResourceEvaluation{
		EvaluationID:                     evaluationID,
		OrganizationID:                   orgID,
		SubjectID:                        subjectID,
		TotalCost:                        totalCost,
		Currency:                         findomain.CurrencyUSD,
		TotalCount:                       len(usages),
		SuccessfulCount:                  successfulCount,
		EffectiveCostPerSuccessfulResult: effectiveCost,
		BudgetLimitAmount:                budgetLimit,
		BudgetVariance:                   budgetVariance,
		MEPerformanceOutcome:             mePerformanceOutcome,
		MESuccessRate:                    meSuccessRate,
		Status:                           status,
		CreatedAt:                        time.Now().UTC(),
	}
	if err := a.Finance.CreateResourceEvaluation(ctx, e); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return Result{Outcome: Accepted, ResourceID: &evaluationID}
}

// GetResourceEvaluation is the read projection served by
// GET /v1/finance/evaluations/{evaluationId}.
func (a *Application) GetResourceEvaluation(ctx context.Context, evaluationID uuid.UUID) (findomain.ResourceEvaluation, error) {
	orgID := a.Fixtures.Organization().OrganizationID
	return a.Finance.GetResourceEvaluation(ctx, orgID, evaluationID)
}
