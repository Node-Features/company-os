package application

import (
	"context"
	"time"

	findomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/finance"
	"github.com/google/uuid"
)

// CreateBudget is the CREATE_BUDGET use case (ROADMAP.md Phase 4 Slice 3):
// establishes one Budget for the one capability subject scope, caller-
// supplied LimitAmount in USD.
func (a *Application) CreateBudget(ctx context.Context, req CreateBudgetRequest) Result {
	orgID := a.Fixtures.Organization().OrganizationID
	subjectID := a.Fixtures.Capability().CapabilityDefinitionID

	if req.LimitAmount <= 0 {
		return Result{Outcome: Rejected, Reasons: []string{"limit_amount_must_be_positive"}}
	}

	budgetID := uuid.New()
	_, govResult, ok := a.evaluateAutomaticGovernance(ctx, req.RequestID, orgID, req.PrincipalID,
		"finance.budget.create", "Budget", budgetID.String(), budgetID.String())
	if !ok {
		return govResult
	}

	b := &findomain.Budget{
		BudgetID:       budgetID,
		OrganizationID: orgID,
		SubjectID:      subjectID,
		LimitAmount:    req.LimitAmount,
		Currency:       findomain.CurrencyUSD,
		CreatedAt:      time.Now().UTC(),
	}
	if err := a.Finance.CreateBudget(ctx, b); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return Result{Outcome: Accepted, ResourceID: &budgetID}
}
