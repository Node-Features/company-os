package supabase

import (
	"context"
	"errors"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// KnowledgeRepository implements ports.KnowledgeRepository.
type KnowledgeRepository struct{ p *Pool }

func NewKnowledgeRepository(p *Pool) *KnowledgeRepository {
	return &KnowledgeRepository{p: p}
}

// knowledgeItemColumns is the shared SELECT list every read query and
// scanKnowledgeItem agree on, in order.
const knowledgeItemColumns = `
	knowledge_item_id, organization_id, version, claim, content_digest, classification,
	source_type, source_id, produced_by_principal_id, produced_by_method, status,
	duplicate_of_item_id, created_at, reviewer_principal_id, governance_decision_id,
	approval_id, reviewed_at`

func (r *KnowledgeRepository) CaptureItem(ctx context.Context, item *knowledge.KnowledgeItem) error {
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO knowledge_items (knowledge_item_id, organization_id, version, claim, content_digest,
		                              classification, source_type, source_id, produced_by_principal_id,
		                              produced_by_method, status, duplicate_of_item_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		item.KnowledgeItemID, item.OrganizationID, item.Version, item.Claim, item.ContentDigest,
		item.Classification, item.SourceType, item.SourceID, item.ProducedByPrincipalID,
		item.ProducedByMethod, item.Status, item.DuplicateOfItemID, item.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ports.ErrConflict
		}
		return err
	}
	return nil
}

func (r *KnowledgeRepository) GetLatestVersion(ctx context.Context, organizationID, knowledgeItemID uuid.UUID) (knowledge.KnowledgeItem, error) {
	return r.get(ctx, `
		SELECT `+knowledgeItemColumns+`
		FROM knowledge_items WHERE organization_id=$1 AND knowledge_item_id=$2
		ORDER BY version DESC LIMIT 1`,
		organizationID, knowledgeItemID)
}

func (r *KnowledgeRepository) GetLatestBySource(ctx context.Context, organizationID uuid.UUID, sourceType knowledge.SourceType, sourceID uuid.UUID) (knowledge.KnowledgeItem, error) {
	return r.get(ctx, `
		SELECT `+knowledgeItemColumns+`
		FROM knowledge_items WHERE organization_id=$1 AND source_type=$2 AND source_id=$3
		ORDER BY version DESC LIMIT 1`,
		organizationID, sourceType, sourceID)
}

// FindDuplicateCandidates returns the latest version of every other
// KnowledgeItem in this organization sharing the exact content digest — a
// review signal only (knowledge.md: "not automatic merges").
func (r *KnowledgeRepository) FindDuplicateCandidates(ctx context.Context, organizationID uuid.UUID, contentDigest string, excludeItemID uuid.UUID) ([]knowledge.KnowledgeItem, error) {
	rows, err := r.p.pool.Query(ctx, `
		SELECT DISTINCT ON (knowledge_item_id) `+knowledgeItemColumns+`
		FROM knowledge_items
		WHERE organization_id=$1 AND content_digest=$2 AND knowledge_item_id <> $3
		ORDER BY knowledge_item_id, version DESC`,
		organizationID, contentDigest, excludeItemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []knowledge.KnowledgeItem
	for rows.Next() {
		item, err := scanKnowledgeItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// TransitionStatus is knowledge.approve's guarded compare-and-swap — see
// ports.KnowledgeRepository's doc comment. Mirrors ObjectiveRepository.
// CreateObjective's transaction shape exactly, as an UPDATE instead of an
// INSERT.
func (r *KnowledgeRepository) TransitionStatus(ctx context.Context, organizationID, knowledgeItemID uuid.UUID, version int,
	expectedCurrentStatus, newStatus knowledge.Status, reviewerPrincipalID, governanceDecisionID uuid.UUID,
	closePendingCommand, consumeApproval *uuid.UUID) error {
	tx, err := r.p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
		UPDATE knowledge_items
		SET status=$1, reviewer_principal_id=$2, governance_decision_id=$3, approval_id=$4, reviewed_at=now()
		WHERE organization_id=$5 AND knowledge_item_id=$6 AND version=$7 AND status=$8`,
		newStatus, reviewerPrincipalID, governanceDecisionID, consumeApproval,
		organizationID, knowledgeItemID, version, expectedCurrentStatus)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrConflict
	}

	if closePendingCommand != nil {
		if _, err := tx.Exec(ctx, `UPDATE pending_commands SET status='RESOLVED', closed_at=now() WHERE pending_command_id=$1`, *closePendingCommand); err != nil {
			return err
		}
	}
	if consumeApproval != nil {
		if _, err := tx.Exec(ctx, `UPDATE approvals SET status='CONSUMED', decided_at=now() WHERE approval_id=$1`, *consumeApproval); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanKnowledgeItem(row rowScanner) (knowledge.KnowledgeItem, error) {
	var item knowledge.KnowledgeItem
	var classification, sourceType, producedByMethod, status string
	err := row.Scan(&item.KnowledgeItemID, &item.OrganizationID, &item.Version, &item.Claim, &item.ContentDigest,
		&classification, &sourceType, &item.SourceID, &item.ProducedByPrincipalID, &producedByMethod, &status,
		&item.DuplicateOfItemID, &item.CreatedAt, &item.ReviewerPrincipalID, &item.GovernanceDecisionID,
		&item.ApprovalID, &item.ReviewedAt)
	if err != nil {
		return knowledge.KnowledgeItem{}, err
	}
	item.Classification = knowledge.Classification(classification)
	item.SourceType = knowledge.SourceType(sourceType)
	item.ProducedByMethod = knowledge.ProducedByMethod(producedByMethod)
	item.Status = knowledge.Status(status)
	return item, nil
}

func (r *KnowledgeRepository) get(ctx context.Context, query string, args ...any) (knowledge.KnowledgeItem, error) {
	item, err := scanKnowledgeItem(r.p.pool.QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return knowledge.KnowledgeItem{}, ports.ErrNotFound
	}
	if err != nil {
		return knowledge.KnowledgeItem{}, err
	}
	return item, nil
}

var _ ports.KnowledgeRepository = (*KnowledgeRepository)(nil)
