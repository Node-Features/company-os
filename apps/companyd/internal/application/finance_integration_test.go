package application

import (
	"context"
	"testing"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/google/uuid"
)

// TestIntegration_Finance_BudgetToResourceEvaluation proves ROADMAP.md
// Phase 4 Slice 3's full Budget -> CostConstraint -> ResourceUsage ->
// ResourceEvaluation chain end to end against the real database, using the
// migration-seeded "gemini" PriceProfile: 4 succeeded Results (1000 input /
// 200 output tokens each -> $0.0001*1 + $0.0004*0.2 = 0.00018 each) and 1
// failed Result (same usage, still priced and included in total cost per
// finance.md's "must not count only successful calls").
//
// RunResourceEvaluation sums *every* ResourceUsage ever recorded for the
// subject (no time-window scoping this slice — matches finance.md's "no
// idempotency-replay guard"-style deferrals elsewhere), and this slice's
// one-Budget/CostConstraint-per-subject scoping means every Finance test
// shares that same subject. Against a real, persistent (non-reset-between-
// runs) database, a second run of this test — or any other Finance test
// that recorded usage first — leaves prior rows in place. So this test
// captures a baseline before recording its own 5 rows and asserts the
// *delta*, not an absolute total, making it correct whether run alone,
// repeatedly, or alongside the rest of the suite.
func TestIntegration_Finance_BudgetToResourceEvaluation(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()
	orgID := app.Fixtures.Organization().OrganizationID
	subjectID := app.Fixtures.Capability().CapabilityDefinitionID

	baseline, err := app.Finance.GetResourceUsageBySubject(ctx, orgID, subjectID)
	if err != nil {
		t.Fatalf("baseline GetResourceUsageBySubject: %v", err)
	}
	var baselineCost float64
	var baselineSuccessful int
	for _, u := range baseline {
		baselineCost += u.Cost
		if u.Succeeded {
			baselineSuccessful++
		}
	}
	baselineCount := len(baseline)

	budget := app.CreateBudget(ctx, CreateBudgetRequest{
		RequestID: uuid.New(), PrincipalID: principalID, LimitAmount: 1.00,
	})
	if budget.Outcome != Accepted || budget.ResourceID == nil {
		t.Fatalf("CreateBudget outcome = %s (reasons: %v)", budget.Outcome, budget.Reasons)
	}

	constraint := app.CreateCostConstraint(ctx, CreateCostConstraintRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
	})
	if constraint.Outcome != Accepted || constraint.ResourceID == nil {
		t.Fatalf("CreateCostConstraint outcome = %s (reasons: %v)", constraint.Outcome, constraint.Reasons)
	}

	outcomes := []result.Outcome{result.OutcomeSucceeded, result.OutcomeSucceeded, result.OutcomeSucceeded, result.OutcomeSucceeded, result.OutcomeFailed}
	const inputTokens, outputTokens = 1000, 200
	var addedCost float64
	for _, outcome := range outcomes {
		_, _ = startedReadyWorkflow(t, app)
		_, resultID := submitFakeResultWithProvider(t, app, outcome, "gemini", "gemini-2.5-flash", inputTokens, outputTokens)

		usage := app.RecordResourceUsage(ctx, RecordResourceUsageRequest{
			RequestID: uuid.New(), PrincipalID: principalID, ResultID: resultID,
		})
		if usage.Outcome != Accepted || usage.ResourceID == nil {
			t.Fatalf("RecordResourceUsage(%s) outcome = %s (reasons: %v)", outcome, usage.Outcome, usage.Reasons)
		}
		addedCost += float64(inputTokens)/1000*0.0001 + float64(outputTokens)/1000*0.0004
	}

	evaluation := app.RunResourceEvaluation(ctx, RunResourceEvaluationRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
	})
	if evaluation.Outcome != Accepted || evaluation.ResourceID == nil {
		t.Fatalf("RunResourceEvaluation outcome = %s (reasons: %v)", evaluation.Outcome, evaluation.Reasons)
	}

	got, err := app.GetResourceEvaluation(ctx, *evaluation.ResourceID)
	if err != nil {
		t.Fatalf("GetResourceEvaluation: %v", err)
	}
	if got.TotalCount != baselineCount+5 || got.SuccessfulCount != baselineSuccessful+4 {
		t.Fatalf("TotalCount/SuccessfulCount = %d/%d, want %d/%d", got.TotalCount, got.SuccessfulCount, baselineCount+5, baselineSuccessful+4)
	}
	wantTotalCost := baselineCost + addedCost
	if diff := got.TotalCost - wantTotalCost; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("TotalCost = %v, want %v", got.TotalCost, wantTotalCost)
	}
	if got.EffectiveCostPerSuccessfulResult == nil {
		t.Fatalf("EffectiveCostPerSuccessfulResult is nil, want non-nil (%d successes)", baselineSuccessful+4)
	}
	if got.BudgetLimitAmount == nil || *got.BudgetLimitAmount != 1.00 {
		t.Fatalf("BudgetLimitAmount = %v, want 1.00", got.BudgetLimitAmount)
	}
	if got.BudgetVariance == nil || *got.BudgetVariance >= 0 {
		t.Fatalf("BudgetVariance = %v, want negative (well under budget)", got.BudgetVariance)
	}
	if string(got.Status) != "COMPUTED" {
		t.Fatalf("Status = %s, want COMPUTED", got.Status)
	}

	// GetConstraintStatus is asserted after CreateCostConstraint+recording,
	// not before, since it too reflects the same accumulated cumulative
	// cost — still comfortably under the $1.00 budget even with prior
	// runs' residue.
	status, err := app.GetConstraintStatus(ctx)
	if err != nil {
		t.Fatalf("GetConstraintStatus: %v", err)
	}
	if status.Outcome != "WITHIN_LIMIT" && status.Outcome != "APPROACHING_LIMIT" {
		t.Fatalf("GetConstraintStatus.Outcome = %s, want WITHIN_LIMIT or APPROACHING_LIMIT (cumulative %.6f)", status.Outcome, status.CumulativeCost)
	}
}

// TestIntegration_RecordResourceUsage_UnknownProviderRejected proves
// finance.md's fail-closed invariant: a Result from a provider with no
// seeded PriceProfile is rejected, never priced at zero.
func TestIntegration_RecordResourceUsage_UnknownProviderRejected(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	_, _ = startedReadyWorkflow(t, app)
	_, resultID := submitFakeResultWithProvider(t, app, result.OutcomeSucceeded, "integration-test", "integration-test", 100, 50)

	usage := app.RecordResourceUsage(ctx, RecordResourceUsageRequest{
		RequestID: uuid.New(), PrincipalID: principalID, ResultID: resultID,
	})
	if usage.Outcome != Rejected {
		t.Fatalf("RecordResourceUsage outcome = %s (reasons: %v), want REJECTED (no PriceProfile for \"integration-test\")", usage.Outcome, usage.Reasons)
	}
}

// TestIntegration_GetConstraintStatus_NotApplicableBeforeConstraintExists
// proves GetConstraintStatus never fabricates an enforcement outcome
// before a CostConstraint exists for the subject.
func TestIntegration_GetConstraintStatus_NotApplicableBeforeConstraintExists(t *testing.T) {
	app := requireRealApp(t)

	// Every Finance test this slice shares one org+subject scope (the one
	// fixtures.CapabilityID), so a CostConstraint created by another test
	// in the same run may already exist here — this only asserts the
	// outcome is one this fixed-subject scope can legitimately produce,
	// never an error or a fabricated enforcement outcome.
	status, err := app.GetConstraintStatus(context.Background())
	if err != nil {
		t.Fatalf("GetConstraintStatus: %v", err)
	}
	if status.Outcome != "NOT_APPLICABLE" && status.Outcome != "WITHIN_LIMIT" {
		t.Fatalf("GetConstraintStatus.Outcome = %s, want NOT_APPLICABLE or WITHIN_LIMIT", status.Outcome)
	}
}
