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

// IssueRecommendation is the ISSUE_RECOMMENDATION use case
// (docs/workflows/research-loop.md). The Recommendation this produces is a
// terminal artifact this slice — turning it into an Objective proposal is
// ROADMAP.md Phase 4 Slice 4's job, not this one's.
func (a *Application) IssueRecommendation(ctx context.Context, req IssueRecommendationRequest) Result {
	orgID := a.Fixtures.Organization().OrganizationID

	f, err := a.Research.GetFinding(ctx, orgID, req.FindingID)
	if errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Rejected, Reasons: []string{"finding_not_found"}}
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	if reasons := research.ValidateIssueRecommendation(f, req.ProposedAction); len(reasons) > 0 {
		return Result{Outcome: Rejected, Reasons: reasons}
	}

	recommendationID := uuid.New()
	_, govResult, ok := a.evaluateAutomaticGovernance(ctx, req.RequestID, orgID, req.PrincipalID,
		"research.recommendation.issue", "Recommendation", recommendationID.String(), recommendationID.String())
	if !ok {
		return govResult
	}

	rec := &researchdomain.Recommendation{
		RecommendationID: recommendationID,
		OrganizationID:   orgID,
		FindingID:        req.FindingID,
		ProposedAction:   req.ProposedAction,
		Status:           researchdomain.RecommendationProposed,
		CreatedAt:        time.Now().UTC(),
	}
	if err := a.Research.IssueRecommendation(ctx, rec); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return Result{Outcome: Accepted, ResourceID: &recommendationID}
}
