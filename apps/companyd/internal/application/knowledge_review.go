package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	knowledgedomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"
	"github.com/Node-Features/company-os/apps/companyd/internal/governance"
	kernelknowledge "github.com/Node-Features/company-os/apps/companyd/internal/kernel/knowledge"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// RequestKnowledgeApprovalRequest is the REQUEST_KNOWLEDGE_APPROVAL use
// case's input — the client asserts which exact version/content it believes
// is current (docs/architecture/knowledge.md: "exact item-version/content
// digest" verification), never a bare "approve the latest" request.
type RequestKnowledgeApprovalRequest struct {
	RequestID       uuid.UUID
	PrincipalID     uuid.UUID
	KnowledgeItemID uuid.UUID
	Version         int
	ContentDigest   string
}

// RequestKnowledgeApproval is Phase 5 Slice 2's entry point into
// knowledge.approve. governance/policy.go's "knowledge-approve" rule is
// unconditional AutonomyApprovalRequired
// (docs/architecture/knowledge.md: "Deterministic automatic approval is
// disabled... until a dedicated ADR is accepted"), so every call here
// deterministically returns APPROVAL_REQUIRED — unless the requester is the
// candidate's own producer, in which case it's a real, reachable DENIED
// (separation of duties).
func (a *Application) RequestKnowledgeApproval(ctx context.Context, req RequestKnowledgeApprovalRequest) Result {
	orgID := a.Fixtures.Organization().OrganizationID

	item, err := a.Knowledge.GetLatestVersion(ctx, orgID, req.KnowledgeItemID)
	found := true
	if errors.Is(err, ports.ErrNotFound) {
		found = false
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	cmd := command.KnowledgeApprovalCommandEnvelope{
		SchemaVersion:         1,
		CommandID:             uuid.New(),
		RequestID:             req.RequestID,
		IdempotencyKey:        req.RequestID.String(),
		CommandType:           command.ApproveKnowledgeItem,
		OrganizationID:        orgID,
		KnowledgeItemID:       req.KnowledgeItemID,
		Version:               req.Version,
		ContentDigest:         req.ContentDigest,
		RequestingPrincipalID: req.PrincipalID,
		DeclaredTime:          time.Now().UTC(),
		CorrelationID:         req.RequestID,
	}

	proposal, reasons := kernelknowledge.ValidateApprovalRequest(found, item.Status, item.Version, item.ContentDigest, req.Version, req.ContentDigest, cmd)
	if proposal == nil {
		return Result{Outcome: Rejected, Reasons: reasons}
	}

	decision, gateResult, ok := a.evaluateKnowledgeApprovalGovernance(ctx, cmd, *proposal, item.ProducedByPrincipalID, nil)
	if !ok {
		return gateResult
	}

	// Structurally unreachable under this slice's always-ApprovalRequired
	// policy (documented above), implemented for completeness the same way
	// ProposeObjective's identical branch is.
	return a.transitionKnowledgeItem(ctx, cmd, knowledgedomain.StatusApproved, req.PrincipalID, decision.DecisionID, nil, nil)
}

// resumeApproveKnowledgeItem replays the exact original APPROVE_KNOWLEDGE_ITEM
// envelope now that Approval evidence exists — same pattern
// resumeProposeObjective (objective_propose.go) uses. Re-running
// ValidateApprovalRequest against the freshly reloaded item catches a
// version/content change since the original request (knowledge.md: "a stale
// item version... requires a new Governance evaluation") before Governance
// is even consulted again.
func (a *Application) resumeApproveKnowledgeItem(ctx context.Context, pc *command.PendingCommand, appr *approval.Approval) Result {
	var cmd command.KnowledgeApprovalCommandEnvelope
	if err := json.Unmarshal(pc.GovernedPayload, &cmd); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{"corrupt governed payload: " + err.Error()}}
	}

	item, err := a.Knowledge.GetLatestVersion(ctx, cmd.OrganizationID, cmd.KnowledgeItemID)
	found := true
	if errors.Is(err, ports.ErrNotFound) {
		found = false
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	proposal, reasons := kernelknowledge.ValidateApprovalRequest(found, item.Status, item.Version, item.ContentDigest, cmd.Version, cmd.ContentDigest, cmd)
	if proposal == nil {
		return Result{Outcome: Rejected, Reasons: reasons}
	}

	decision, gateResult, ok := a.evaluateKnowledgeApprovalGovernance(ctx, cmd, *proposal, item.ProducedByPrincipalID, &governance.ApprovalEvidence{
		ApprovalID:     appr.ApprovalID,
		ProposalDigest: proposal.ProposalDigest,
	})
	if !ok {
		return gateResult
	}

	return a.transitionKnowledgeItem(ctx, cmd, knowledgedomain.StatusApproved, reviewerOf(appr, cmd.RequestingPrincipalID), decision.DecisionID, &pc.PendingCommandID, &appr.ApprovalID)
}

// rejectApproveKnowledgeItem transitions the item to REJECTED — a real
// state transition (docs/architecture/knowledge.md: Governance DENY leaves
// the candidate unchanged "or permits a separate rejected-review transition
// according to the review request" — this is that transition, driven by a
// human reject disposition, not a Governance DENY). Deliberately does not
// re-run governance.Evaluate: the Approval-table compare-and-swap in
// ResolveApproval already is the authorization check for "is this a
// legitimate resolution" — re-running Governance is what the approve path
// needs (it unlocks ALLOW via matching ApprovalEvidence), not something a
// reject needs.
func (a *Application) rejectApproveKnowledgeItem(ctx context.Context, pc *command.PendingCommand, appr *approval.Approval) Result {
	var cmd command.KnowledgeApprovalCommandEnvelope
	if err := json.Unmarshal(pc.GovernedPayload, &cmd); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{"corrupt governed payload: " + err.Error()}}
	}

	res := a.transitionKnowledgeItem(ctx, cmd, knowledgedomain.StatusRejected, reviewerOf(appr, cmd.RequestingPrincipalID), pc.GovernanceDecisionID, &pc.PendingCommandID, &appr.ApprovalID)
	if res.Outcome != Accepted {
		return res
	}
	return Result{Outcome: Rejected, Reasons: []string{command.ReasonApprovalRejected}, ResourceID: res.ResourceID}
}

// reviewerOf is the actual decider — always fixtures.Registry.ApproverPrincipal()
// today (appr.DecidedByPrincipalID, populated by ResolveApproval on a
// successful resolution) — distinct from whichever Principal requested the
// review. Falls back to the requester only defensively; ResolveApproval's
// repository always sets DecidedByPrincipalID on success.
func reviewerOf(appr *approval.Approval, fallback uuid.UUID) uuid.UUID {
	if appr.DecidedByPrincipalID != nil {
		return *appr.DecidedByPrincipalID
	}
	return fallback
}

func (a *Application) transitionKnowledgeItem(ctx context.Context, cmd command.KnowledgeApprovalCommandEnvelope, newStatus knowledgedomain.Status, reviewerPrincipalID, governanceDecisionID uuid.UUID, closePendingCommand, consumeApproval *uuid.UUID) Result {
	err := a.Knowledge.TransitionStatus(ctx, cmd.OrganizationID, cmd.KnowledgeItemID, cmd.Version,
		knowledgedomain.StatusDraft, newStatus, reviewerPrincipalID, governanceDecisionID, closePendingCommand, consumeApproval)
	if errors.Is(err, ports.ErrConflict) {
		return Result{Outcome: Conflict, Reasons: []string{"knowledge_item_stale_version_or_digest"}}
	}
	if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}
	id := cmd.KnowledgeItemID
	return Result{Outcome: Accepted, ResourceID: &id}
}
