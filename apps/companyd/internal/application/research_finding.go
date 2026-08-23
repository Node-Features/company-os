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

// PublishFinding is the PUBLISH_FINDING use case
// (docs/workflows/research-loop.md).
func (a *Application) PublishFinding(ctx context.Context, req PublishFindingRequest) Result {
	orgID := a.Fixtures.Organization().OrganizationID

	q, err := a.Research.GetResearchQuestion(ctx, orgID, req.QuestionID)
	if errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Rejected, Reasons: []string{"question_not_found"}}
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	if reasons := research.ValidatePublishFinding(q, req.Claim, req.EvidenceIDs); len(reasons) > 0 {
		return Result{Outcome: Rejected, Reasons: reasons}
	}

	// A Finding cannot exceed what its evidence supports (research.md
	// invariant): the one part of that this layer can mechanically check is
	// that every cited EvidenceID actually exists and belongs to this
	// question — not that the claim is correctly derived from it.
	existing, err := a.Research.ListEvidence(ctx, orgID, req.QuestionID)
	if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}
	known := make(map[uuid.UUID]bool, len(existing))
	for _, e := range existing {
		known[e.EvidenceID] = true
	}
	for _, id := range req.EvidenceIDs {
		if !known[id] {
			return Result{Outcome: Rejected, Reasons: []string{"evidence_not_found_for_question"}}
		}
	}

	findingID := uuid.New()
	_, govResult, ok := a.evaluateAutomaticGovernance(ctx, req.RequestID, orgID, req.PrincipalID,
		"research.finding.publish", "Finding", findingID.String(), findingID.String())
	if !ok {
		return govResult
	}

	f := &researchdomain.Finding{
		FindingID:      findingID,
		OrganizationID: orgID,
		QuestionID:     req.QuestionID,
		Claim:          req.Claim,
		EvidenceIDs:    req.EvidenceIDs,
		Status:         researchdomain.FindingPublished,
		CreatedAt:      time.Now().UTC(),
	}
	if err := a.Research.PublishFinding(ctx, f); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return Result{Outcome: Accepted, ResourceID: &findingID}
}
