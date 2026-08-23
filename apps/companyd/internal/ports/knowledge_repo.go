package ports

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"
	"github.com/google/uuid"
)

// KnowledgeRepository persists KnowledgeItem versions. Rows are append-only
// per version (docs/domain/knowledge.md: "immutable version") — no
// compare-and-swap here, matching ResearchRepository/ObjectiveRepository's
// shape, not AuthoritativeStateRepository's.
type KnowledgeRepository interface {
	CaptureItem(ctx context.Context, item *knowledge.KnowledgeItem) error
	GetLatestVersion(ctx context.Context, organizationID, knowledgeItemID uuid.UUID) (knowledge.KnowledgeItem, error)
	GetLatestBySource(ctx context.Context, organizationID uuid.UUID, sourceType knowledge.SourceType, sourceID uuid.UUID) (knowledge.KnowledgeItem, error)
	FindDuplicateCandidates(ctx context.Context, organizationID uuid.UUID, contentDigest string, excludeItemID uuid.UUID) ([]knowledge.KnowledgeItem, error)
}
