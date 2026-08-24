package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// fabricatePendingApproval inserts a real GovernanceDecision, PendingCommand,
// and Approval row directly through the repositories — the same
// "bypass the external boundary to prove the mechanism" pattern this
// package's other integration tests already use (e.g.
// knowledge_integration_test.go's fabricated stale-version rows) — since no
// real Application code path can produce a self-approval or
// non-human-decider collision today (every governed use case's requester
// and the fixed Approver fixture are always distinct principals by
// construction). requestingPrincipalID and expiresAt are the two facts each
// test below varies; everything else is a minimal, otherwise-valid row.
func fabricatePendingApproval(t *testing.T, app *Application, requestingPrincipalID uuid.UUID, expiresAt *time.Time) (approvalID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	orgID := app.Fixtures.Organization().OrganizationID

	decision := policy.GovernanceDecision{
		DecisionID:           uuid.New(),
		OrganizationID:       orgID,
		RequestID:            uuid.New(),
		CorrelationID:        uuid.New(),
		PrincipalID:          requestingPrincipalID,
		Action:               "workflow.cancel",
		ResourceType:         "Workflow",
		ResourceID:           uuid.New().String(),
		ProposalDigest:       "fabricated-digest",
		TrustedContextDigest: "fabricated-ctx-digest",
		PolicyVersion:        "first-slice-v1",
		AutonomyLevel:        policy.AutonomyApprovalRequired,
		Outcome:              policy.DecisionRequireApproval,
		DecidedAt:            time.Now().UTC(),
	}
	if err := app.Repo.SaveGovernanceDecision(ctx, decision.DecisionID, decision.OrganizationID, decision.RequestID, decision.CorrelationID, decision.PrincipalID,
		decision.Action, decision.ResourceType, decision.ResourceID, decision.ProposalDigest, decision.TrustedContextDigest,
		decision.PolicyVersion, string(decision.AutonomyLevel), string(decision.Outcome), decision.MatchedRuleID, decision.Reason); err != nil {
		t.Fatalf("fabricate GovernanceDecision: %v", err)
	}

	now := time.Now().UTC()
	pc := &command.PendingCommand{
		PendingCommandID: uuid.New(),
		OrganizationID:   orgID,
		Status:           "PENDING",
		CommandType:      command.CancelWorkflow,
		ProposalDigest:   decision.ProposalDigest,
		GovernedPayload:  []byte(`{}`),
		RequestID:        decision.RequestID,
		IdempotencyKey:   uuid.New().String(),
		CorrelationID:    decision.CorrelationID,
		CreatedAt:        now,
		ExpiresAt:        expiresAt,
	}
	appr := &approval.Approval{
		ApprovalID:            uuid.New(),
		OrganizationID:        orgID,
		PendingCommandID:      pc.PendingCommandID,
		RequestingPrincipalID: requestingPrincipalID,
		Action:                decision.Action,
		ResourceType:          decision.ResourceType,
		ResourceID:            decision.ResourceID,
		ProposalDigest:        decision.ProposalDigest,
		Status:                approval.StatusPending,
		CreatedAt:             now,
	}
	pc.ApprovalID = &appr.ApprovalID
	if err := app.Pending.CreatePendingApproval(ctx, pc, decision.DecisionID, appr); err != nil {
		t.Fatalf("fabricate PendingCommand/Approval: %v", err)
	}
	return appr.ApprovalID
}

// TestIntegration_ResolveApproval_SelfApprovalDenied proves
// docs/domain/approval.md's "the requester cannot approve its own request"
// invariant is now a real, structural, unconditional check in
// ResolveApproval (docs/adr/ADR-0010-authority-model-formalization.md) —
// not merely true by accident of which fixtures happen to be wired where.
// The fabricated Approval's RequestingPrincipalID is set to exactly
// fixtures.ApproverPrincipal().PrincipalID, the Principal
// Application.ResolveApproval always uses as decider — a collision no real
// request path in this codebase can produce today.
func TestIntegration_ResolveApproval_SelfApprovalDenied(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	approverID := app.Fixtures.ApproverPrincipal().PrincipalID

	approvalID := fabricatePendingApproval(t, app, approverID, nil)

	res := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: approvalID, Approve: true})
	if res.Outcome != Denied {
		t.Fatalf("self-approval attempt outcome = %s (reasons: %v), want DENIED", res.Outcome, res.Reasons)
	}

	// The Approval must not have been burned by the illegitimate attempt —
	// a distinct, legitimate human decider can still resolve it. Calling
	// the repository directly here (not Application.ResolveApproval, which
	// always uses the same fixture) to supply a genuinely different
	// decider.
	_, _, err := app.Pending.ResolveApproval(ctx, approvalID, principal.Principal{PrincipalID: uuid.New(), Kind: principal.KindHuman}, true, nil)
	if err != nil {
		t.Fatalf("legitimate resolution after a denied self-approval attempt: %v, want success (approval must still be PENDING)", err)
	}
}

// TestIntegration_ResolveApproval_NonHumanDeciderDenied proves
// governance.md's "agents, services, providers, and models cannot serve as
// the reviewer" is enforced structurally, for any CommandType — not just
// Knowledge's. Driven directly through the repository (not
// Application.ResolveApproval, which today only ever supplies the fixed
// HUMAN-kind Approver fixture as decider) to exercise a decider Kind no
// current Application code path can produce.
func TestIntegration_ResolveApproval_NonHumanDeciderDenied(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()

	approvalID := fabricatePendingApproval(t, app, uuid.New(), nil)

	_, _, err := app.Pending.ResolveApproval(ctx, approvalID, principal.Principal{PrincipalID: uuid.New(), Kind: principal.KindService}, true, nil)
	if !errors.Is(err, ports.ErrNonHumanDecider) {
		t.Fatalf("ResolveApproval with a Service-kind decider: err = %v, want ports.ErrNonHumanDecider", err)
	}

	// Not burned: a real human decider still succeeds afterward.
	_, _, err = app.Pending.ResolveApproval(ctx, approvalID, principal.Principal{PrincipalID: uuid.New(), Kind: principal.KindHuman}, true, nil)
	if err != nil {
		t.Fatalf("legitimate human resolution after a denied non-human attempt: %v, want success", err)
	}
}

// TestIntegration_ResolveApproval_ExpiredPendingCommandRejected proves the
// approvalTTL mechanism (pipeline.go) is actually enforced: a PendingCommand
// whose ExpiresAt has already passed is rejected as expired — and, unlike
// self-approval/non-human-decider, this DOES mutate state: both rows
// transition to EXPIRED rather than being left PENDING forever, since an
// expired approval is never legitimately resolvable by anyone.
func TestIntegration_ResolveApproval_ExpiredPendingCommandRejected(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	past := time.Now().UTC().Add(-time.Hour)

	approvalID := fabricatePendingApproval(t, app, uuid.New(), &past)

	res := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: approvalID, Approve: true})
	if res.Outcome != Rejected {
		t.Fatalf("resolving an expired approval outcome = %s (reasons: %v), want REJECTED", res.Outcome, res.Reasons)
	}

	// Now genuinely EXPIRED, not PENDING — a second attempt, even by a
	// legitimate human decider, is a CONFLICT, proving the first call
	// actually transitioned the row rather than merely refusing in place.
	_, _, err := app.Pending.ResolveApproval(ctx, approvalID, principal.Principal{PrincipalID: uuid.New(), Kind: principal.KindHuman}, true, nil)
	if !errors.Is(err, ports.ErrConflict) {
		t.Fatalf("resolving an already-EXPIRED approval: err = %v, want ports.ErrConflict", err)
	}
}

// TestIntegration_ResolveApproval_ConcurrentResolutionOneWins races two
// goroutines against the same real REQUIRE_APPROVAL flow (a genuine
// CancelWorkflow on a READY Workflow, not a fabricated row) — proving the
// restructured SELECT ... FOR UPDATE transaction remains exactly as safe
// against concurrent resolution as the original single-UPDATE CAS was.
func TestIntegration_ResolveApproval_ConcurrentResolutionOneWins(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	workflowID, version := startedReadyWorkflow(t, app)

	pending := app.CancelWorkflow(ctx, CancelWorkflowRequest{
		RequestID: uuid.New(), IdempotencyKey: uuid.New().String(),
		WorkflowID: workflowID, ExpectedVersion: version,
	})
	if pending.Outcome != ApprovalRequired || pending.ApprovalID == nil {
		t.Fatalf("setup: CancelWorkflow outcome = %s (reasons: %v), want APPROVAL_REQUIRED", pending.Outcome, pending.Reasons)
	}

	const racers = 5
	var wg sync.WaitGroup
	results := make([]Result, racers)
	wg.Add(racers)
	for i := 0; i < racers; i++ {
		go func(i int) {
			defer wg.Done()
			results[i] = app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *pending.ApprovalID, Approve: true})
		}(i)
	}
	wg.Wait()

	accepted, conflicted := 0, 0
	for _, res := range results {
		switch res.Outcome {
		case Accepted:
			accepted++
		case Conflict:
			conflicted++
		default:
			t.Errorf("unexpected outcome %s (reasons: %v) among concurrent resolvers", res.Outcome, res.Reasons)
		}
	}
	if accepted != 1 || conflicted != racers-1 {
		t.Fatalf("accepted=%d conflicted=%d, want exactly 1 accepted and %d conflicted", accepted, conflicted, racers-1)
	}
}
