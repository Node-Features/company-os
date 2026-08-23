package application

import (
	"context"
	"errors"
	"time"

	knowledgedomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"
	kernelknowledge "github.com/Node-Features/company-os/apps/companyd/internal/kernel/knowledge"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// CaptureKnowledgeCandidate is Phase 5 Slice 1's ingestion use case
// (docs/architecture/knowledge.md steps 1-4): capture a candidate from a
// supported source, normalize it into a new immutable DRAFT KnowledgeItem
// version without overwriting prior versions, and flag exact-content
// duplicates elsewhere as a review signal. It never sets any status but
// DRAFT — knowledge.review/knowledge.approve (a later slice) is a separate
// governed action this one does not implement.
func (a *Application) CaptureKnowledgeCandidate(ctx context.Context, req CaptureKnowledgeCandidateRequest) Result {
	orgID := a.Fixtures.Organization().OrganizationID

	var claim string
	switch req.SourceType {
	case knowledgedomain.SourceResearchFinding:
		f, err := a.Research.GetFinding(ctx, orgID, req.SourceID)
		if errors.Is(err, ports.ErrNotFound) {
			return Result{Outcome: Rejected, Reasons: []string{"source_not_found"}}
		} else if err != nil {
			return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
		}
		claim = f.Claim
	default:
		return Result{Outcome: Rejected, Reasons: []string{"unsupported_source_type"}}
	}

	contentDigest := kernelknowledge.ContentDigest(claim)

	latest, err := a.Knowledge.GetLatestBySource(ctx, orgID, req.SourceType, req.SourceID)
	hasLatest := true
	if errors.Is(err, ports.ErrNotFound) {
		hasLatest = false
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	nextVersion, reasons := kernelknowledge.ValidateCapture(hasLatest, latest.Version, latest.ContentDigest, contentDigest)
	if len(reasons) > 0 {
		return Result{Outcome: Rejected, Reasons: reasons}
	}

	itemID := latest.KnowledgeItemID
	if !hasLatest {
		itemID = uuid.New()
	}

	_, govResult, ok := a.evaluateAutomaticGovernance(ctx, req.RequestID, orgID, req.PrincipalID,
		"knowledge.item.capture", "KnowledgeItem", itemID.String(), contentDigest)
	if !ok {
		return govResult
	}

	var duplicateOf *uuid.UUID
	dups, err := a.Knowledge.FindDuplicateCandidates(ctx, orgID, contentDigest, itemID)
	if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}
	if len(dups) > 0 {
		d := dups[0].KnowledgeItemID
		duplicateOf = &d
	}

	item := &knowledgedomain.KnowledgeItem{
		KnowledgeItemID:       itemID,
		OrganizationID:        orgID,
		Version:               nextVersion,
		Claim:                 claim,
		ContentDigest:         contentDigest,
		Classification:        knowledgedomain.ClassificationInternal,
		SourceType:            req.SourceType,
		SourceID:              req.SourceID,
		ProducedByPrincipalID: req.PrincipalID,
		ProducedByMethod:      knowledgedomain.MethodSourceVerbatim,
		Status:                knowledgedomain.StatusDraft,
		DuplicateOfItemID:     duplicateOf,
		CreatedAt:             time.Now().UTC(),
	}
	if err := a.Knowledge.CaptureItem(ctx, item); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return Result{Outcome: Accepted, ResourceID: &itemID}
}

// GetKnowledgeItem returns the latest version of one KnowledgeItem.
func (a *Application) GetKnowledgeItem(ctx context.Context, knowledgeItemID uuid.UUID) (knowledgedomain.KnowledgeItem, error) {
	orgID := a.Fixtures.Organization().OrganizationID
	return a.Knowledge.GetLatestVersion(ctx, orgID, knowledgeItemID)
}
