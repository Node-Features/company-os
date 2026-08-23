package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// TestIntegration_ListPendingApprovals_ReturnsAndClearsOnResolution proves
// the Approval inbox's data source end to end against the real database:
// a real REQUIRE_APPROVAL request appears in the list, and resolving it
// removes it.
func TestIntegration_ListPendingApprovals_ReturnsAndClearsOnResolution(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	producerID := uuid.New()
	requesterID := uuid.New()

	item := captureKnowledgeItem(t, app, ctx, producerID, "Approval-inbox test claim, unique per run: "+uuid.New().String())

	reqRes := app.RequestKnowledgeApproval(ctx, RequestKnowledgeApprovalRequest{
		RequestID: uuid.New(), PrincipalID: requesterID,
		KnowledgeItemID: item.KnowledgeItemID, Version: item.Version, ContentDigest: item.ContentDigest,
	})
	if reqRes.Outcome != ApprovalRequired || reqRes.ApprovalID == nil {
		t.Fatalf("RequestKnowledgeApproval outcome = %s (reasons: %v)", reqRes.Outcome, reqRes.Reasons)
	}

	before, err := app.ListPendingApprovals(ctx)
	if err != nil {
		t.Fatalf("ListPendingApprovals: %v", err)
	}
	found := false
	for _, a := range before {
		if a.ApprovalID == *reqRes.ApprovalID {
			found = true
			if a.Action != "knowledge.approve" {
				t.Fatalf("Action = %q, want knowledge.approve", a.Action)
			}
			if a.ResourceType != "KnowledgeItem" || a.ResourceID != item.KnowledgeItemID.String() {
				t.Fatalf("ResourceType/ResourceID = %s/%s, want KnowledgeItem/%s", a.ResourceType, a.ResourceID, item.KnowledgeItemID)
			}
		}
	}
	if !found {
		t.Fatalf("Approval %v not found in pending list", *reqRes.ApprovalID)
	}

	decideRes := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *reqRes.ApprovalID, Approve: true})
	if decideRes.Outcome != Accepted {
		t.Fatalf("ResolveApproval(approve) outcome = %s (reasons: %v)", decideRes.Outcome, decideRes.Reasons)
	}

	after, err := app.ListPendingApprovals(ctx)
	if err != nil {
		t.Fatalf("ListPendingApprovals: %v", err)
	}
	for _, a := range after {
		if a.ApprovalID == *reqRes.ApprovalID {
			t.Fatalf("Approval %v still pending after resolution", *reqRes.ApprovalID)
		}
	}
}
