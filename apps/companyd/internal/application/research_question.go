package application

import (
	"context"
	"errors"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/departments/research"
	researchdomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/research"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// OpenResearchQuestion is the OPEN_RESEARCH_QUESTION use case
// (docs/workflows/research-loop.md).
func (a *Application) OpenResearchQuestion(ctx context.Context, req OpenResearchQuestionRequest) Result {
	orgID := a.Fixtures.Organization().OrganizationID

	sig, err := a.Research.GetSignal(ctx, orgID, req.SignalID)
	if errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Rejected, Reasons: []string{"signal_not_found"}}
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	if reasons := research.ValidateOpenQuestion(sig, req.Text); len(reasons) > 0 {
		return Result{Outcome: Rejected, Reasons: reasons}
	}

	questionID := uuid.New()
	_, govResult, ok := a.evaluateAutomaticGovernance(ctx, req.RequestID, orgID, req.PrincipalID,
		"research.question.open", "ResearchQuestion", questionID.String(), questionID.String())
	if !ok {
		return govResult
	}

	q := &researchdomain.ResearchQuestion{
		QuestionID:     questionID,
		OrganizationID: orgID,
		SignalID:       req.SignalID,
		Text:           req.Text,
		Status:         researchdomain.QuestionOpen,
		CreatedAt:      time.Now().UTC(),
	}
	if err := a.Research.CreateResearchQuestion(ctx, q); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return Result{Outcome: Accepted, ResourceID: &questionID}
}
