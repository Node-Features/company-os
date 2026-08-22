package application

import (
	"context"
	"errors"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	kernelwf "github.com/Node-Features/company-os/apps/companyd/internal/kernel/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// CancelWorkflow is the CANCEL_WORKFLOW use case: PLANNED -> CANCELLED or
// READY -> CANCELLED, requested only by the Workflow's initiating
// Principal. See docs/domain/workflow.md#first-slice-commands-and-legal-transitions.
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

	decision, denyResult, ok := a.evaluateGovernance(ctx, cmd, *proposal, true)
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
