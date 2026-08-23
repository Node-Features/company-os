package knowledge

import (
	"testing"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"
)

func strPtr(s string) *string { return &s }

func TestValidateQuery_EmptyStatuses_DefaultsToApprovedOnly(t *testing.T) {
	effective, reasons := ValidateQuery(nil, nil)
	if len(reasons) != 0 {
		t.Fatalf("reasons = %v, want none", reasons)
	}
	if len(effective) != 1 || effective[0] != knowledge.StatusApproved {
		t.Fatalf("effective = %v, want [APPROVED]", effective)
	}
}

func TestValidateQuery_ExplicitApprovedOnly_NoPurposeNeeded(t *testing.T) {
	effective, reasons := ValidateQuery([]knowledge.Status{knowledge.StatusApproved}, nil)
	if len(reasons) != 0 {
		t.Fatalf("reasons = %v, want none", reasons)
	}
	if len(effective) != 1 || effective[0] != knowledge.StatusApproved {
		t.Fatalf("effective = %v, want [APPROVED]", effective)
	}
}

func TestValidateQuery_DraftInclusiveWithoutPurpose_Rejected(t *testing.T) {
	_, reasons := ValidateQuery([]knowledge.Status{knowledge.StatusDraft}, nil)
	if len(reasons) != 1 || reasons[0] != "purpose_required_for_draft_inclusive_query" {
		t.Fatalf("reasons = %v, want [purpose_required_for_draft_inclusive_query]", reasons)
	}
}

func TestValidateQuery_MultiStatusIncludingApproved_StillNeedsPurpose(t *testing.T) {
	_, reasons := ValidateQuery([]knowledge.Status{knowledge.StatusApproved, knowledge.StatusDraft}, nil)
	if len(reasons) != 1 || reasons[0] != "purpose_required_for_draft_inclusive_query" {
		t.Fatalf("reasons = %v, want [purpose_required_for_draft_inclusive_query]", reasons)
	}
}

func TestValidateQuery_DraftInclusiveWithPurpose_Accepted(t *testing.T) {
	effective, reasons := ValidateQuery([]knowledge.Status{knowledge.StatusDraft, knowledge.StatusInReview}, strPtr("editorial review dashboard"))
	if len(reasons) != 0 {
		t.Fatalf("reasons = %v, want none", reasons)
	}
	if len(effective) != 2 {
		t.Fatalf("effective = %v, want 2 statuses echoed back", effective)
	}
}

func TestValidateQuery_InvalidStatus_Rejected(t *testing.T) {
	_, reasons := ValidateQuery([]knowledge.Status{"NOT_A_REAL_STATUS"}, strPtr("purpose"))
	if len(reasons) != 1 || reasons[0] != "invalid_status" {
		t.Fatalf("reasons = %v, want [invalid_status]", reasons)
	}
}
