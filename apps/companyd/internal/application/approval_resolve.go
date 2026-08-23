package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/governance"
	kernelwf "github.com/Node-Features/company-os/apps/companyd/internal/kernel/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
)

// ResolveApproval is the human-decision use case for a PENDING Approval
// (docs/domain/approval.md's lifecycle step 3). The deciding Principal is
// always fixtures.Registry.ApproverPrincipal() (Kind HUMAN) — never
// client-asserted, same as every other Principal reference this slice.
// docs/domain/approval.md's "only a human, not the requester, may approve"
// invariant holds by construction rather than a runtime check: the
// deciding Principal is always the Approver fixture and the requesting
// Principal on every CancelWorkflowRequest is always the (distinct)
// Trigger fixture, so they can never collide given how those two request
// types are wired this slice — there is no client input path that could
// make them the same Principal.
//
// On reject, the PendingCommand is closed (by the repository, in the same
// transaction as the Approval status write) and nothing else happens. On
// approve, resumeCancelWorkflow immediately attempts to complete the
// original governed transition — docs/domain/approval.md frames approval
// and resumption as sequential steps of one resolution, not two separate
// client-triggered actions.
func (a *Application) ResolveApproval(ctx context.Context, req ResolveApprovalRequest) Result {
	pc, appr, err := a.Pending.ResolveApproval(ctx, req.ApprovalID, a.Fixtures.ApproverPrincipal().PrincipalID, req.Approve, req.Reason)
	if errors.Is(err, ports.ErrConflict) {
		return Result{Outcome: Conflict, Reasons: []string{command.ReasonApprovalAlreadyResolved}}
	}
	if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	if !req.Approve {
		return Result{Outcome: Rejected, Reasons: []string{command.ReasonApprovalRejected}}
	}

	switch pc.CommandType {
	case command.CancelWorkflow:
		return a.resumeCancelWorkflow(ctx, pc, appr)
	case command.ProposeObjective:
		return a.resumeProposeObjective(ctx, pc, appr)
	default:
		return Result{Outcome: Unavailable, Reasons: []string{"unsupported command type for resumption: " + string(pc.CommandType)}}
	}
}

// resumeCancelWorkflow replays the exact original CANCEL_WORKFLOW command
// envelope now that Approval evidence exists. It deliberately does not
// call a.store: pc.GovernedPayload's IdempotencyKey was already correctly
// recorded as APPROVAL_REQUIRED at the original request time (that WAS the
// accurate outcome for that exact request then) — overwriting it here
// would violate idempotent-replay semantics for the original request. This
// resumption is a causally separate event, guarded against double
// execution by the approvals-table compare-and-swap in
// PendingCommandRepository.ResolveApproval, not by the idempotency store.
func (a *Application) resumeCancelWorkflow(ctx context.Context, pc *command.PendingCommand, appr *approval.Approval) Result {
	var cmd command.WorkflowCommandEnvelope
	if err := json.Unmarshal(pc.GovernedPayload, &cmd); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{"corrupt governed payload: " + err.Error()}}
	}

	current, err := a.Repo.LoadWorkflow(ctx, cmd.OrganizationID, cmd.WorkflowID)
	if errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Rejected, Reasons: []string{command.ReasonWorkflowNotFound}}
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	// Replaying the byte-for-byte original cmd against the freshly loaded
	// current Workflow reproduces the byte-for-byte original proposal
	// (kernelwf.newProposal is a pure function of cmd + current.State)
	// whenever nothing material changed — which is exactly when this
	// should be allowed to proceed. If the Workflow's version or state did
	// change since the original request, this rejects here with a normal
	// REJECTED/ILLEGAL_STATE or VERSION_MISMATCH reason, before Governance
	// is even consulted — no special-case digest logic needed.
	proposal, reasons := kernelwf.ValidateCancelProposal(cmd, *current)
	if proposal == nil {
		return Result{Outcome: Rejected, Reasons: reasons}
	}

	decision, denyResult, ok := a.evaluateGovernance(ctx, cmd, *proposal, true, governanceOptions{
		ResourceOwnerPrincipalID:      &current.InitiatingPrincipalID,
		AdditionalAutonomyRequirement: cancelAutonomyRequirement(current.State),
		ApprovalEvidence: &governance.ApprovalEvidence{
			ApprovalID:     appr.ApprovalID,
			ProposalDigest: proposal.ProposalDigest,
		},
	})
	if !ok {
		return denyResult
	}

	// declaredTime for FinalizeCancel's event timestamp is deliberately
	// "now", not cmd.DeclaredTime — the cancellation genuinely takes effect
	// now, even though cmd itself (used above for digest-sensitive
	// proposal validation) stays unmutated.
	kd, reasons := kernelwf.FinalizeCancel(cmd, *proposal, decision, *current, time.Now().UTC())
	if kd == nil {
		return Result{Outcome: Rejected, Reasons: reasons}
	}

	next := *current
	next.Version = kd.NextVersion
	next.State = kd.NextState
	next.UpdatedAt = kd.DecidedAt
	reason := "CANCELLED"
	next.TerminalReason = &reason
	closeOutstandingIntent := current.State == workflow.StateReady

	if err := a.Repo.CommitTransition(ctx, &next, current.Version, kd.Events, kd.GovernanceDecisionID, nil, &pc.PendingCommandID, &appr.ApprovalID, nil, closeOutstandingIntent); err != nil {
		if errors.Is(err, ports.ErrConflict) {
			return Result{Outcome: Conflict, Reasons: []string{command.ReasonVersionMismatch}}
		}
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return Result{Outcome: Accepted, Workflow: viewOf(&next)}
}
