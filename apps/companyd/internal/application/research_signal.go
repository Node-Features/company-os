package application

import (
	"context"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/departments/research"
	researchdomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/research"
	"github.com/google/uuid"
)

// SubmitSignal is the SUBMIT_SIGNAL use case (docs/workflows/research-loop.md).
func (a *Application) SubmitSignal(ctx context.Context, req SubmitSignalRequest) Result {
	if reasons := research.ValidateSubmitSignal(req.SourceType, req.Description); len(reasons) > 0 {
		return Result{Outcome: Rejected, Reasons: reasons}
	}

	orgID := a.Fixtures.Organization().OrganizationID
	signalID := uuid.New()
	_, govResult, ok := a.evaluateAutomaticGovernance(ctx, req.RequestID, orgID, req.PrincipalID,
		"research.signal.submit", "Signal", signalID.String(), signalID.String())
	if !ok {
		return govResult
	}

	sig := &researchdomain.Signal{
		SignalID:               signalID,
		OrganizationID:         orgID,
		SourceType:             req.SourceType,
		Description:            req.Description,
		SubmittedByPrincipalID: req.PrincipalID,
		SubmittedAt:            time.Now().UTC(),
	}
	if err := a.Research.CreateSignal(ctx, sig); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return Result{Outcome: Accepted, ResourceID: &signalID}
}
