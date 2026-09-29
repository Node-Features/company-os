package application

import (
	"context"
	"errors"
	"testing"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
	objdomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/objective"
	"github.com/google/uuid"
)

// TestIntegration_ListApprovals_DefaultsToPending proves the endpoint's
// inbox default: no statuses given means PENDING-only, per
// Application.ListApprovals's doc comment — a fresh org's one proposed,
// unresolved Objective is exactly what comes back.
func TestIntegration_ListApprovals_DefaultsToPending(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	sourceID := makeEvaluationSource(t, app, principalID)
	propose := app.ProposeObjective(ctx, ProposeObjectiveRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: objdomain.SourceEvaluation, SourceID: sourceID,
		Title: "Pending objective", Intent: "left unresolved for ListApprovals test",
	})
	if propose.Outcome != ApprovalRequired || propose.ApprovalID == nil {
		t.Fatalf("ProposeObjective outcome = %s (reasons: %v)", propose.Outcome, propose.Reasons)
	}

	got, err := app.ListApprovals(ctx, nil, 0, 0)
	if err != nil {
		t.Fatalf("ListApprovals: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListApprovals() (default PENDING) returned %d approvals, want 1", len(got))
	}
	if got[0].ApprovalID != *propose.ApprovalID {
		t.Fatalf("ListApprovals()[0].ApprovalID = %s, want %s", got[0].ApprovalID, *propose.ApprovalID)
	}
	if got[0].Status != approval.StatusPending {
		t.Fatalf("ListApprovals()[0].Status = %s, want PENDING", got[0].Status)
	}
}

// TestIntegration_ListApprovals_StatusFilterAndOrdering proves statuses
// actually filter (a resolved Approval no longer appears in the PENDING
// default, but does appear when explicitly requested) and that results
// come back oldest-first (an inbox, not a history view — the opposite
// ordering from ListObjectives' newest-first).
func TestIntegration_ListApprovals_StatusFilterAndOrdering(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	// First: propose-and-approve (ends CONSUMED once CreateObjective
	// consumes it — see approval_resolve.go's resumeProposeObjective).
	consumedSourceID := makeEvaluationSource(t, app, principalID)
	consumedPropose := app.ProposeObjective(ctx, ProposeObjectiveRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: objdomain.SourceEvaluation, SourceID: consumedSourceID,
		Title: "Approved objective", Intent: "will be approved",
	})
	if consumedPropose.Outcome != ApprovalRequired || consumedPropose.ApprovalID == nil {
		t.Fatalf("ProposeObjective(consumed) outcome = %s (reasons: %v)", consumedPropose.Outcome, consumedPropose.Reasons)
	}
	resolve := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *consumedPropose.ApprovalID, Approve: true, DecidingPrincipal: app.Fixtures.ApproverPrincipal()})
	if resolve.Outcome != Accepted {
		t.Fatalf("ResolveApproval(approve) outcome = %s (reasons: %v)", resolve.Outcome, resolve.Reasons)
	}

	// Second: propose-and-reject (stays REJECTED, never consumed).
	rejectedSourceID := makeResourceEvaluationSource(t, app, principalID)
	rejectedPropose := app.ProposeObjective(ctx, ProposeObjectiveRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: objdomain.SourceResourceEvaluation, SourceID: rejectedSourceID,
		Title: "Declined objective", Intent: "will be rejected",
	})
	if rejectedPropose.Outcome != ApprovalRequired || rejectedPropose.ApprovalID == nil {
		t.Fatalf("ProposeObjective(rejected) outcome = %s (reasons: %v)", rejectedPropose.Outcome, rejectedPropose.Reasons)
	}
	reject := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *rejectedPropose.ApprovalID, Approve: false, DecidingPrincipal: app.Fixtures.ApproverPrincipal()})
	if reject.Outcome != Rejected {
		t.Fatalf("ResolveApproval(reject) outcome = %s, want REJECTED", reject.Outcome)
	}

	// Third: left pending, proposed after the other two so ordering has
	// something to prove.
	pendingSourceID := makeFindingSource(t, app, principalID)
	pendingPropose := app.ProposeObjective(ctx, ProposeObjectiveRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: objdomain.SourceFinding, SourceID: pendingSourceID,
		Title: "Still-pending objective", Intent: "left unresolved",
	})
	if pendingPropose.Outcome != ApprovalRequired || pendingPropose.ApprovalID == nil {
		t.Fatalf("ProposeObjective(pending) outcome = %s (reasons: %v)", pendingPropose.Outcome, pendingPropose.Reasons)
	}

	// Default (PENDING-only) must show only the still-pending one — the
	// resolved two must not leak into the inbox view.
	pendingOnly, err := app.ListApprovals(ctx, nil, 0, 0)
	if err != nil {
		t.Fatalf("ListApprovals(default): %v", err)
	}
	if len(pendingOnly) != 1 || pendingOnly[0].ApprovalID != *pendingPropose.ApprovalID {
		t.Fatalf("ListApprovals(default) = %+v, want exactly the still-pending Approval %s", pendingOnly, *pendingPropose.ApprovalID)
	}

	// Explicit CONSUMED+REJECTED must show the other two, oldest first
	// (the approved-then-consumed one was proposed before the rejected
	// one).
	resolved, err := app.ListApprovals(ctx, []approval.Status{approval.StatusConsumed, approval.StatusRejected}, 0, 0)
	if err != nil {
		t.Fatalf("ListApprovals(CONSUMED,REJECTED): %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf("ListApprovals(CONSUMED,REJECTED) returned %d approvals, want 2", len(resolved))
	}
	if resolved[0].ApprovalID != *consumedPropose.ApprovalID || resolved[1].ApprovalID != *rejectedPropose.ApprovalID {
		t.Fatalf("ListApprovals(CONSUMED,REJECTED) order = [%s, %s], want oldest-first [%s, %s]",
			resolved[0].ApprovalID, resolved[1].ApprovalID, *consumedPropose.ApprovalID, *rejectedPropose.ApprovalID)
	}
	if resolved[0].Status != approval.StatusConsumed {
		t.Fatalf("resolved[0].Status = %s, want CONSUMED", resolved[0].Status)
	}
	if resolved[1].Status != approval.StatusRejected {
		t.Fatalf("resolved[1].Status = %s, want REJECTED", resolved[1].Status)
	}
}

// TestListApprovals_InvalidStatusRejected needs no real database — same
// reasoning as TestListObjectives_InvalidStatusRejected: the validation
// loop runs before touching a.Fixtures or a.Pending.
func TestListApprovals_InvalidStatusRejected(t *testing.T) {
	app := &Application{}
	_, err := app.ListApprovals(context.Background(), []approval.Status{"NOT_A_REAL_STATUS"}, 0, 0)
	if !errors.Is(err, ErrInvalidApprovalStatus) {
		t.Fatalf("ListApprovals with an invalid status: err = %v, want ErrInvalidApprovalStatus", err)
	}
}
