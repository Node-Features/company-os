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

// CreateCostConstraint is the CREATE_COST_CONSTRAINT use case (ROADMAP.md
// Phase 4 Slice 3): looks up the subject's Budget and creates one ADVISORY
// CostConstraint with MaxCost = Budget.LimitAmount. ADVISORY only, always —
// hard enforcement would mean gating Runtime dispatch, out of scope this
// slice.
func (a *Application) CreateCostConstraint(ctx context.Context, req CreateCostConstraintRequest) Result {
	orgID := a.Fixtures.Organization().OrganizationID
	subjectID := a.Fixtures.Capability().CapabilityDefinitionID

	budget, err := a.Finance.GetBudgetBySubject(ctx, orgID, subjectID)
	if errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Rejected, Reasons: []string{"budget_not_found_for_subject"}}
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	constraintID := uuid.New()
	_, govResult, ok := a.evaluateAutomaticGovernance(ctx, req.RequestID, orgID, req.PrincipalID,
		"finance.constraint.create", "CostConstraint", constraintID.String(), constraintID.String())
	if !ok {
		return govResult
	}

	c := &findomain.CostConstraint{
		ConstraintID:    constraintID,
		OrganizationID:  orgID,
		BudgetID:        budget.BudgetID,
		SubjectID:       subjectID,
		MaxCost:         budget.LimitAmount,
		Currency:        budget.Currency,
		EnforcementMode: findomain.EnforcementAdvisory,
		CreatedAt:       time.Now().UTC(),
	}
	if err := a.Finance.CreateCostConstraint(ctx, c); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return Result{Outcome: Accepted, ResourceID: &constraintID}
}

// ConstraintStatusView is the read projection served by
// GET /v1/finance/constraint-status.
type ConstraintStatusView struct {
	SubjectID      string
	CumulativeCost float64
	MaxCost        *float64
	Outcome        string
}

// GetConstraintStatus computes resource.md's enforcement outcome live from
// the subject's persisted ResourceUsage and its most recent
// CostConstraint (NOT_APPLICABLE if none exists yet). Not persisted as its
// own record type — it isn't one of finance.md's five named contracts.
func (a *Application) GetConstraintStatus(ctx context.Context) (ConstraintStatusView, error) {
	orgID := a.Fixtures.Organization().OrganizationID
	subjectID := a.Fixtures.Capability().CapabilityDefinitionID

	usages, err := a.Finance.GetResourceUsageBySubject(ctx, orgID, subjectID)
	if err != nil {
		return ConstraintStatusView{}, err
	}
	var cumulativeCost float64
	for _, u := range usages {
		cumulativeCost += u.Cost
	}

	constraint, err := a.Finance.GetCostConstraintBySubject(ctx, orgID, subjectID)
	view := ConstraintStatusView{SubjectID: subjectID.String(), CumulativeCost: cumulativeCost}
	if errors.Is(err, ports.ErrNotFound) {
		view.Outcome = string(findept.EvaluateConstraintOutcome(cumulativeCost, nil))
		return view, nil
	}
	if err != nil {
		return ConstraintStatusView{}, err
	}
	view.MaxCost = &constraint.MaxCost
	view.Outcome = string(findept.EvaluateConstraintOutcome(cumulativeCost, &constraint))
	return view, nil
}
