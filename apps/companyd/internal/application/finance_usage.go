package application

import (
	"context"
	"errors"
	"time"

	findept "github.com/Node-Features/company-os/apps/companyd/internal/departments/finance"
	findomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/finance"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// outputInt reads an int-valued key from a Result.Output map, tolerating
// both float64 (the shape json.Unmarshal produces for a value read back
// from the database, since Output is persisted as jsonb) and int (the
// shape a value has when set in-process, before any round trip) — the
// same tolerance runtime.go's own maxOutputTokens parsing uses.
func outputInt(output map[string]any, key string) (int, bool) {
	switch v := output[key].(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	default:
		return 0, false
	}
}

// RecordResourceUsage is the RECORD_RESOURCE_USAGE use case (ROADMAP.md
// Phase 4 Slice 3): given a real completed Result, looks up the seeded
// PriceProfile for its provider, reads its token usage, computes cost, and
// persists a ResourceUsage record. Fails closed — an unknown provider or
// missing token usage is Rejected, never priced at zero (finance.md:
// "missing, stale, incompatible, or unverifiable pricing never defaults to
// zero").
func (a *Application) RecordResourceUsage(ctx context.Context, req RecordResourceUsageRequest) Result {
	orgID := a.Fixtures.Organization().OrganizationID
	subjectID := a.Fixtures.Capability().CapabilityDefinitionID

	res, err := a.Exec.GetResult(ctx, orgID, req.ResultID)
	if errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Rejected, Reasons: []string{"result_not_found"}}
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	profile, err := a.Finance.GetPriceProfile(ctx, res.ProviderAdapter)
	if errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Rejected, Reasons: []string{"no_price_profile_for_provider"}}
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	inputTokens, ok1 := outputInt(res.Output, "inputTokens")
	outputTokens, ok2 := outputInt(res.Output, "outputTokens")
	if !ok1 || !ok2 {
		return Result{Outcome: Rejected, Reasons: []string{"result_missing_token_usage"}}
	}

	cost := findept.ComputeUsageCost(profile, inputTokens, outputTokens)

	usageID := uuid.New()
	_, govResult, ok := a.evaluateAutomaticGovernance(ctx, req.RequestID, orgID, req.PrincipalID,
		"finance.usage.record", "ResourceUsage", usageID.String(), usageID.String())
	if !ok {
		return govResult
	}

	u := &findomain.ResourceUsage{
		UsageID:           usageID,
		OrganizationID:    orgID,
		ResultID:          req.ResultID,
		SubjectID:         subjectID,
		ProviderAdapter:   res.ProviderAdapter,
		ModelID:           res.ModelID,
		InputTokens:       inputTokens,
		OutputTokens:      outputTokens,
		Cost:              cost,
		Currency:          profile.Currency,
		Succeeded:         res.Outcome == result.OutcomeSucceeded,
		MeasurementMethod: findomain.MeasurementActual,
		CreatedAt:         time.Now().UTC(),
	}
	if err := a.Finance.RecordResourceUsage(ctx, u); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return Result{Outcome: Accepted, ResourceID: &usageID}
}
