package application

import (
	"context"
	"errors"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	kernelwf "github.com/Node-Features/company-os/apps/companyd/internal/kernel/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// cancelAutonomyRequirement is ROADMAP.md Phase 3 Slice 2's concrete,
// HTTP-reachable REQUIRE_APPROVAL trigger: cancelling a Workflow that's
// already READY (dispatched, possibly mid-provider-call) needs a human's
// sign-off; cancelling a PLANNED one (nothing dispatched yet) stays
// automatic. Shared by the original request (CancelWorkflow) and the
// resumed one (resumeCancelWorkflow) so both compute the identical
// requirement from the identical fact (current Workflow state).
func cancelAutonomyRequirement(state workflow.State) *policy.AutonomyLevel {
	if state != workflow.StateReady {
		return nil
	}
	lvl := policy.AutonomyApprovalRequired
	return &lvl
}

// CancelWorkflow is the CANCEL_WORKFLOW use case: PLANNED -> CANCELLED or
// READY -> CANCELLED, requested only by the Workflow's initiating
// Principal. The Kernel (kernelwf.ValidateCancelProposal) validates only
// state/version legality; ownership is Governance's job
// (docs/architecture/governance.md: "Governance determines whether a
// specific principal may execute a specific action on a specific
// resource") — the current Workflow's InitiatingPrincipalID is passed to
// evaluateGovernance as the resource owner, so a non-initiator's request
// is DENIED before FinalizeCancel ever runs, per governance.md's
// invariant that no governed action reaches an executor without a
// current persisted ALLOW decision (ROADMAP.md Phase 3 Slice 1).
// Cancelling a READY Workflow additionally requires human approval
// (cancelAutonomyRequirement, ROADMAP.md Phase 3 Slice 2) — the caller
// gets ApprovalRequired with an ApprovalID to resolve via
// Application.ResolveApproval rather than an immediate CANCELLED. See
// docs/domain/workflow.md#first-slice-commands-and-legal-transitions.
func (a *Application) CancelWorkflow(ctx context.Context, req CancelWorkflowRequest) Result {
	reg := a.Fixtures

	if cached, ok := a.replay(ctx, reg.Organization().OrganizationID, req.IdempotencyKey); ok {
		return cached
	}

	current, err := a.Repo.LoadWorkflow(ctx, reg.Organization().OrganizationID, req.WorkflowID)
	if errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Rejected, Reasons: []string{command.ReasonWorkflowNotFound}}
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	cmd := command.WorkflowCommandEnvelope{
		SchemaVersion:         1,
		CommandID:             uuid.New(),
		RequestID:             req.RequestID,
		IdempotencyKey:        req.IdempotencyKey,
		CommandType:           command.CancelWorkflow,
		OrganizationID:        current.OrganizationID,
		WorkflowID:            current.WorkflowID,
		ExpectedVersion:       &req.ExpectedVersion,
		ObjectiveID:           current.ObjectiveID,
		DefinitionID:          current.DefinitionID,
		DefinitionVersion:     current.DefinitionVersion,
		RequestingPrincipalID: reg.TriggerPrincipal().PrincipalID,
		Inputs:                current.Inputs,
		DeclaredTime:          time.Now().UTC(),
		CorrelationID:         current.CorrelationID,
	}

	proposal, reasons := kernelwf.ValidateCancelProposal(cmd, *current)
	if proposal == nil {
		return a.store(ctx, cmd, Result{Outcome: Rejected, Reasons: reasons})
	}

	decision, denyResult, ok := a.evaluateGovernance(ctx, cmd, *proposal, true, governanceOptions{
		ResourceOwnerPrincipalID:      &current.InitiatingPrincipalID,
		AdditionalAutonomyRequirement: cancelAutonomyRequirement(current.State),
	})
	if !ok {
		return a.store(ctx, cmd, denyResult)
	}

	kd, reasons := kernelwf.FinalizeCancel(cmd, *proposal, decision, *current, cmd.DeclaredTime)
	if kd == nil {
		return a.store(ctx, cmd, Result{Outcome: Rejected, Reasons: reasons})
	}

	next := *current
	next.Version = kd.NextVersion
	next.State = kd.NextState
	next.UpdatedAt = cmd.DeclaredTime
	reason := "CANCELLED"
	next.TerminalReason = &reason

	// A READY Workflow has an outstanding ExecutionIntent; PLANNED never
	// does, since START_WORKFLOW is what produces one.
	closeOutstandingIntent := current.State == workflow.StateReady

	if err := a.Repo.CommitTransition(ctx, &next, req.ExpectedVersion, kd.Events, kd.GovernanceDecisionID, nil, nil, nil, nil, closeOutstandingIntent); err != nil {
		if errors.Is(err, ports.ErrConflict) {
			return a.store(ctx, cmd, Result{Outcome: Conflict, Reasons: []string{command.ReasonVersionMismatch}})
		}
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return a.store(ctx, cmd, Result{Outcome: Accepted, Workflow: viewOf(&next)})
}
