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

// CreateWorkflow is the CREATE_WORKFLOW use case: absent -> PLANNED, per
// docs/architecture/application.md's 12-step state-changing sequence.
func (a *Application) CreateWorkflow(ctx context.Context, req CreateWorkflowRequest) Result {
	reg := a.Fixtures

	if cached, ok := a.replay(ctx, reg.Organization().OrganizationID, req.IdempotencyKey); ok {
		return cached
	}

	cmd := command.WorkflowCommandEnvelope{
		SchemaVersion:         1,
		CommandID:             uuid.New(),
		RequestID:             req.RequestID,
		IdempotencyKey:        req.IdempotencyKey,
		CommandType:           command.CreateWorkflow,
		OrganizationID:        reg.Organization().OrganizationID,
		WorkflowID:            uuid.New(),
		ObjectiveID:           reg.Objective().ObjectiveID,
		DefinitionID:          reg.WorkflowDefinition().DefinitionID,
		DefinitionVersion:     reg.WorkflowDefinition().Version,
		RequestingPrincipalID: reg.TriggerPrincipal().PrincipalID,
		Inputs:                map[string]any{"prompt": "Say hello to CompanyOS in one short sentence.", "maxOutputTokens": 128},
		DeclaredTime:          time.Now().UTC(),
		CorrelationID:         req.RequestID,
	}

	// Step 2: verify the explicit aggregate absence CREATE_WORKFLOW requires.
	if _, err := a.Repo.LoadWorkflow(ctx, cmd.OrganizationID, cmd.WorkflowID); err == nil {
		return a.store(ctx, cmd, Result{Outcome: Conflict, Reasons: []string{command.ReasonWorkflowAlreadyExists}})
	} else if !errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	proposal, reasons := kernelwf.ValidateCreateProposal(cmd, reg)
	if proposal == nil {
		return a.store(ctx, cmd, Result{Outcome: Rejected, Reasons: reasons})
	}

	decision, denyResult, ok := a.evaluateGovernance(ctx, cmd, *proposal, true, governanceOptions{})
	if !ok {
		return a.store(ctx, cmd, denyResult)
	}

	kd, reasons := kernelwf.FinalizeCreate(cmd, *proposal, decision, cmd.DeclaredTime)
	if kd == nil {
		return a.store(ctx, cmd, Result{Outcome: Rejected, Reasons: reasons})
	}

	w := &workflow.Workflow{
		OrganizationID:        cmd.OrganizationID,
		WorkflowID:            cmd.WorkflowID,
		Version:               kd.NextVersion,
		DefinitionID:          cmd.DefinitionID,
		DefinitionVersion:     cmd.DefinitionVersion,
		ObjectiveID:           cmd.ObjectiveID,
		State:                 kd.NextState,
		InitiatingPrincipalID: cmd.RequestingPrincipalID,
		CorrelationID:         cmd.CorrelationID,
		Inputs:                cmd.Inputs,
		CreatedAt:             cmd.DeclaredTime,
		UpdatedAt:             cmd.DeclaredTime,
	}

	if err := a.Repo.CreateWorkflow(ctx, w, kd.Events, kd.GovernanceDecisionID); err != nil {
		if errors.Is(err, ports.ErrConflict) {
			return a.store(ctx, cmd, Result{Outcome: Conflict, Reasons: []string{command.ReasonWorkflowAlreadyExists}})
		}
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	return a.store(ctx, cmd, Result{Outcome: Accepted, Workflow: viewOf(w)})
}
