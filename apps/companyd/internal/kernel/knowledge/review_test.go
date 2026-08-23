package knowledge

import (
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"
	"github.com/google/uuid"
)

func testApprovalEnvelope() command.KnowledgeApprovalCommandEnvelope {
	return command.KnowledgeApprovalCommandEnvelope{
		SchemaVersion:         1,
		CommandID:             uuid.New(),
		RequestID:             uuid.New(),
		IdempotencyKey:        "idem-1",
		CommandType:           command.ApproveKnowledgeItem,
		OrganizationID:        uuid.New(),
		KnowledgeItemID:       uuid.New(),
		Version:               1,
		ContentDigest:         "digest-a",
		RequestingPrincipalID: uuid.New(),
		DeclaredTime:          time.Now().UTC(),
		CorrelationID:         uuid.New(),
	}
}

func TestValidateApprovalRequest_NotFoundRejected(t *testing.T) {
	proposal, reasons := ValidateApprovalRequest(false, knowledge.StatusDraft, 1, "digest-a", 1, "digest-a", testApprovalEnvelope())
	if proposal != nil {
		t.Fatalf("proposal = %+v, want nil", proposal)
	}
	if len(reasons) != 1 || reasons[0] != "knowledge_item_not_found" {
		t.Fatalf("reasons = %v, want [knowledge_item_not_found]", reasons)
	}
}

func TestValidateApprovalRequest_NotDraftRejected(t *testing.T) {
	proposal, reasons := ValidateApprovalRequest(true, knowledge.StatusApproved, 1, "digest-a", 1, "digest-a", testApprovalEnvelope())
	if proposal != nil {
		t.Fatalf("proposal = %+v, want nil", proposal)
	}
	if len(reasons) != 1 || reasons[0] != "knowledge_item_not_in_draft_status" {
		t.Fatalf("reasons = %v, want [knowledge_item_not_in_draft_status]", reasons)
	}
}

func TestValidateApprovalRequest_VersionMismatchRejected(t *testing.T) {
	proposal, reasons := ValidateApprovalRequest(true, knowledge.StatusDraft, 2, "digest-a", 1, "digest-a", testApprovalEnvelope())
	if proposal != nil {
		t.Fatalf("proposal = %+v, want nil", proposal)
	}
	if len(reasons) != 1 || reasons[0] != "knowledge_item_stale_version_or_digest" {
		t.Fatalf("reasons = %v, want [knowledge_item_stale_version_or_digest]", reasons)
	}
}

func TestValidateApprovalRequest_DigestMismatchRejected(t *testing.T) {
	proposal, reasons := ValidateApprovalRequest(true, knowledge.StatusDraft, 1, "digest-a", 1, "digest-b", testApprovalEnvelope())
	if proposal != nil {
		t.Fatalf("proposal = %+v, want nil", proposal)
	}
	if len(reasons) != 1 || reasons[0] != "knowledge_item_stale_version_or_digest" {
		t.Fatalf("reasons = %v, want [knowledge_item_stale_version_or_digest]", reasons)
	}
}

func TestValidateApprovalRequest_HappyPath_StableDigest(t *testing.T) {
	cmd := testApprovalEnvelope()
	proposal, reasons := ValidateApprovalRequest(true, knowledge.StatusDraft, 1, "digest-a", 1, "digest-a", cmd)
	if proposal == nil {
		t.Fatalf("proposal = nil, reasons = %v, want a proposal", reasons)
	}
	if proposal.Action != "knowledge.approve" {
		t.Fatalf("Action = %q, want knowledge.approve", proposal.Action)
	}
	if proposal.ResourceType != "KnowledgeItem" || proposal.ResourceID != cmd.KnowledgeItemID.String() {
		t.Fatalf("ResourceType/ResourceID = %s/%s, want KnowledgeItem/%s", proposal.ResourceType, proposal.ResourceID, cmd.KnowledgeItemID)
	}
	if proposal.ProposalDigest == "" || proposal.CommandDigest == "" || proposal.TrustedContextDigest == "" {
		t.Fatalf("digest fields not populated: %+v", proposal)
	}

	proposal2, _ := ValidateApprovalRequest(true, knowledge.StatusDraft, 1, "digest-a", 1, "digest-a", cmd)
	if proposal2.ProposalDigest != proposal.ProposalDigest {
		t.Fatalf("ProposalDigest not stable across identical replay: %s != %s", proposal2.ProposalDigest, proposal.ProposalDigest)
	}
}
