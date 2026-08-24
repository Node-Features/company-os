package application

import (
	"context"
	"errors"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/departments/research"
	knowledgedomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"
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

	// ROADMAP.md Phase 5 Slice 4: a published Finding automatically
	// proposes a KnowledgeItem candidate — end-to-end wiring from Research
	// output to Knowledge's ingestion path (Phase 5 Slice 1). This is a
	// derived side effect, never authoritative for the Finding itself: if
	// capture doesn't succeed, PublishFinding still reports Accepted (the
	// Finding is already real, committed domain data), just with a note in
	// Reasons rather than a silently swallowed failure.
	result := Result{Outcome: Accepted, ResourceID: &findingID}
	capture := a.CaptureKnowledgeCandidate(ctx, CaptureKnowledgeCandidateRequest{
		RequestID:   uuid.New(),
		PrincipalID: req.PrincipalID,
		SourceType:  knowledgedomain.SourceResearchFinding,
		SourceID:    findingID,
	})
	if capture.Outcome == Accepted && capture.ResourceID != nil {
		result.KnowledgeItemID = capture.ResourceID
	} else {
		result.Reasons = append(result.Reasons, "knowledge_capture_not_captured:"+string(capture.Outcome))
	}
	return result
}
