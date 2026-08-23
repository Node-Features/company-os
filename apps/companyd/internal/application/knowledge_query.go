package application

import (
	"context"
	"errors"

	knowledgedomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"
	kernelknowledge "github.com/Node-Features/company-os/apps/companyd/internal/kernel/knowledge"
)

const (
	defaultKnowledgeQueryLimit = 50
	maxKnowledgeQueryLimit     = 200
)

// ErrKnowledgeQueryPurposeRequired/ErrKnowledgeQueryInvalidStatus are
// QueryKnowledge's validation sentinels — not an application.Result,
// matching this codebase's existing convention that reads (GetKnowledgeItem,
// GetObjective) return (value, error), never Result/Outcome.
var (
	ErrKnowledgeQueryPurposeRequired = errors.New("purpose required for a draft-inclusive knowledge query")
	ErrKnowledgeQueryInvalidStatus   = errors.New("invalid status in knowledge query")
)

// QueryKnowledgeRequest is Phase 5 Slice 3's retrieval contract input
// (docs/architecture/knowledge.md's Retrieval contract section).
type QueryKnowledgeRequest struct {
	Statuses []knowledgedomain.Status
	Purpose  *string
	Limit    int
}

// QueryKnowledge is the retrieval contract's entry point. Deliberately
// takes no PrincipalID — matching GetKnowledgeItem/GetObjective's existing
// precedent that reads in this codebase don't carry one (no
// classification-clearance/per-Principal query authority exists yet;
// knowledge.md's own open question on that is not answered here).
func (a *Application) QueryKnowledge(ctx context.Context, req QueryKnowledgeRequest) ([]knowledgedomain.KnowledgeItem, error) {
	orgID := a.Fixtures.Organization().OrganizationID

	effective, reasons := kernelknowledge.ValidateQuery(req.Statuses, req.Purpose)
	if len(reasons) > 0 {
		if reasons[0] == "purpose_required_for_draft_inclusive_query" {
			return nil, ErrKnowledgeQueryPurposeRequired
		}
		return nil, ErrKnowledgeQueryInvalidStatus
	}

	limit := req.Limit
	if limit <= 0 || limit > maxKnowledgeQueryLimit {
		limit = defaultKnowledgeQueryLimit
	}

	return a.Knowledge.QueryItems(ctx, orgID, effective, limit)
}
