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

// RecordEvidence is the RECORD_EVIDENCE use case
// (docs/workflows/research-loop.md).
func (a *Application) RecordEvidence(ctx context.Context, req RecordEvidenceRequest) Result {
	orgID := a.Fixtures.Organization().OrganizationID

	q, err := a.Research.GetResearchQuestion(ctx, orgID, req.QuestionID)
	if errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Rejected, Reasons: []string{"question_not_found"}}
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	if reasons := research.ValidateRecordEvidence(q, req.Source, req.Content); len(reasons) > 0 {
		return Result{Outcome: Rejected, Reasons: reasons}
	}

	evidenceID := uuid.New()
	_, govResult, ok := a.evaluateAutomaticGovernance(ctx, req.RequestID, orgID, req.PrincipalID,
		"research.evidence.record", "Evidence", evidenceID.String(), evidenceID.String())
	if !ok {
		return govResult
	}

	retrievedAt := req.RetrievedAt
	if retrievedAt.IsZero() {
		retrievedAt = time.Now().UTC()
	}
	e := &researchdomain.Evidence{
		EvidenceID:     evidenceID,
		OrganizationID: orgID,
		QuestionID:     req.QuestionID,
		Source:         req.Source,
		Content:        req.Content,
		RetrievedAt:    retrievedAt,
		CreatedAt:      time.Now().UTC(),
	}
	if err := a.Research.RecordEvidence(ctx, e); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return Result{Outcome: Accepted, ResourceID: &evidenceID}
}
