package knowledge

import "github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"

var validStatuses = map[knowledge.Status]bool{
	knowledge.StatusDraft:      true,
	knowledge.StatusInReview:   true,
	knowledge.StatusApproved:   true,
	knowledge.StatusRejected:   true,
	knowledge.StatusSuperseded: true,
	knowledge.StatusExpired:    true,
}

// ValidateQuery is the retrieval contract's Kernel legality check
// (docs/architecture/knowledge.md's Retrieval contract section): no
// statuses requested defaults to APPROVED-only, no purpose needed. Any
// other status set requires a non-empty purpose — "draft-inclusive queries
// require an explicit purpose and label every draft." The label itself is
// just the real Status field on each returned KnowledgeItem — no separate
// field is invented here.
func ValidateQuery(statuses []knowledge.Status, purpose *string) (effective []knowledge.Status, reasons []string) {
	if len(statuses) == 0 {
		return []knowledge.Status{knowledge.StatusApproved}, nil
	}
	for _, s := range statuses {
		if !validStatuses[s] {
			return nil, []string{"invalid_status"}
		}
	}
	draftInclusive := len(statuses) != 1 || statuses[0] != knowledge.StatusApproved
	if draftInclusive && (purpose == nil || *purpose == "") {
		return nil, []string{"purpose_required_for_draft_inclusive_query"}
	}
	return statuses, nil
}
