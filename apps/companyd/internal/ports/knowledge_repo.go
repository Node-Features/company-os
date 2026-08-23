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

	// TransitionStatus is knowledge.approve's (Phase 5 Slice 2) guarded
	// compare-and-swap on one immutable version row: WHERE
	// knowledge_item_id=knowledgeItemID AND version=version AND
	// status=expectedCurrentStatus. Zero rows affected (already decided, or
	// changed since the caller last loaded it) returns ports.ErrConflict —
	// the first CAS-style write in KnowledgeRepository (Slice 1's methods
	// were pure create/read), same convention as
	// PendingCommandRepository.ResolveApproval. closePendingCommand/
	// consumeApproval mirror ObjectiveRepository.CreateObjective's
	// parameters of the same name.
	TransitionStatus(ctx context.Context, organizationID, knowledgeItemID uuid.UUID, version int,
		expectedCurrentStatus, newStatus knowledge.Status, reviewerPrincipalID, governanceDecisionID uuid.UUID,
		closePendingCommand, consumeApproval *uuid.UUID) error
}
