package ports

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/research"
	"github.com/google/uuid"
)

// ResearchRepository persists the Research department's core contracts
// (docs/workflows/research-loop.md). No compare-and-swap/version machinery —
// this slice's flow is linear (Signal -> Question -> Evidence -> Finding ->
// Recommendation), not concurrently mutated the way Workflow is.
type ResearchRepository interface {
	CreateSignal(ctx context.Context, s *research.Signal) error
	GetSignal(ctx context.Context, organizationID, signalID uuid.UUID) (research.Signal, error)

	CreateResearchQuestion(ctx context.Context, q *research.ResearchQuestion) error
	GetResearchQuestion(ctx context.Context, organizationID, questionID uuid.UUID) (research.ResearchQuestion, error)

	RecordEvidence(ctx context.Context, e *research.Evidence) error
	ListEvidence(ctx context.Context, organizationID, questionID uuid.UUID) ([]research.Evidence, error)

	PublishFinding(ctx context.Context, f *research.Finding) error
	GetFinding(ctx context.Context, organizationID, findingID uuid.UUID) (research.Finding, error)
	ListFindingsByQuestion(ctx context.Context, organizationID, questionID uuid.UUID) ([]research.Finding, error)

	IssueRecommendation(ctx context.Context, r *research.Recommendation) error
	GetRecommendation(ctx context.Context, organizationID, recommendationID uuid.UUID) (research.Recommendation, error)
	ListRecommendationsByFinding(ctx context.Context, organizationID, findingID uuid.UUID) ([]research.Recommendation, error)
}
