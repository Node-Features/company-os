package application

import (
	"context"
	"errors"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	kernelwf "github.com/Node-Features/company-os/apps/companyd/internal/kernel/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// SubmitResult is the Runtime-result use case
// (docs/architecture/application.md#runtime-result-use-case): Runtime
// submits a proposed Result; Application branches to ACCEPT_WORKFLOW_RESULT
// or REJECT_WORKFLOW_RESULT, or — for INDETERMINATE — persists the
// observation only and leaves the Workflow at its current state.
func (a *Application) SubmitResult(ctx context.Context, req SubmitResultRequest) Result {
	reg := a.Fixtures

	if cached, ok := a.replay(ctx, reg.Organization().OrganizationID, req.IdempotencyKey); ok {
		return cached
	}

	res, err := a.Exec.GetResult(ctx, reg.Organization().OrganizationID, req.ResultID)
	if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	if res.Outcome == result.OutcomeIndeterminate {
		return Result{Outcome: Indeterminate}
	}

	current, err := a.Repo.LoadWorkflow(ctx, reg.Organization().OrganizationID, res.WorkflowID)
	if errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Rejected, Reasons: []string{command.ReasonWorkflowNotFound}}
	} else if err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	commandType := command.RejectWorkflowResult
	if res.Outcome.AcceptsResult() {
		commandType = command.AcceptWorkflowResult
	}

	cmd := command.WorkflowCommandEnvelope{
		SchemaVersion:         1,
		CommandID:             uuid.New(),
		RequestID:             req.RequestID,
		IdempotencyKey:        req.IdempotencyKey,
		CommandType:           commandType,
		OrganizationID:        current.OrganizationID,
		WorkflowID:            current.WorkflowID,
		ExpectedVersion:       &req.ExpectedVersion,
		ObjectiveID:           current.ObjectiveID,
		DefinitionID:          current.DefinitionID,
		DefinitionVersion:     current.DefinitionVersion,
		RequestingPrincipalID: reg.TriggerPrincipal().PrincipalID,
		ResultID:              &req.ResultID,
		DeclaredTime:          time.Now().UTC(),
		CorrelationID:         current.CorrelationID,
	}

	proposal, reasons := kernelwf.ValidateResultProposal(cmd, *current, *res, req.ExpectedVersion)
	if proposal == nil {
		return a.store(ctx, cmd, Result{Outcome: Rejected, Reasons: reasons})
	}

	decision, denyResult, ok := a.evaluateGovernance(ctx, cmd, *proposal, true)
	if !ok {
		return a.store(ctx, cmd, denyResult)
	}

	var kd *command.KernelDecisionEnvelope
	if commandType == command.AcceptWorkflowResult {
		kd, reasons = kernelwf.FinalizeAccept(cmd, *proposal, decision, *current, *res, cmd.DeclaredTime)
	} else {
		kd, reasons = kernelwf.FinalizeReject(cmd, *proposal, decision, *current, *res, cmd.DeclaredTime)
	}
	if kd == nil {
		return a.store(ctx, cmd, Result{Outcome: Rejected, Reasons: reasons})
	}

	next := *current
	next.Version = kd.NextVersion
	next.State = kd.NextState
	next.UpdatedAt = cmd.DeclaredTime
	if kd.NextState.IsTerminal() {
		reason := string(res.Outcome)
		next.TerminalReason = &reason
	}

	accepted := commandType == command.AcceptWorkflowResult
	decideResult := &ports.ResultDecision{ResultID: req.ResultID, Accepted: accepted}

	if err := a.Repo.CommitTransition(ctx, &next, req.ExpectedVersion, kd.Events, kd.GovernanceDecisionID, nil, nil, nil, decideResult); err != nil {
		if errors.Is(err, ports.ErrConflict) {
			return a.store(ctx, cmd, Result{Outcome: Conflict, Reasons: []string{command.ReasonVersionMismatch}})
		}
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return a.store(ctx, cmd, Result{Outcome: Accepted, Workflow: viewOf(&next)})
}
