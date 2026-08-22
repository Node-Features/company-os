package workflow

import (
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
)

func newProposal(cmd command.WorkflowCommandEnvelope, resourceID string, args map[string]any) *command.GovernedCommandProposal {
	p := &command.GovernedCommandProposal{
		ProposalID:            cmd.CommandID, // 1:1 with the command this slice; no re-proposal path yet
		SchemaVersion:         1,
		CommandID:             cmd.CommandID,
		Action:                command.ActionFor[cmd.CommandType],
		ResourceType:          "Workflow",
		ResourceID:            resourceID,
		OrganizationID:        cmd.OrganizationID,
		ExpectedVersion:       cmd.ExpectedVersion,
		Arguments:             args,
		TrustedContextDigest:  canonicalDigest(map[string]any{"org": cmd.OrganizationID, "principal": cmd.RequestingPrincipalID, "time": cmd.DeclaredTime}),
		EffectClassification:  "governed-state-change",
		ExpiresAt:             cmd.DeclaredTime.Add(commandTTL),
	}
	p.CommandDigest = canonicalDigest(cmd)
	p.ProposalDigest = canonicalDigest(struct {
		Action, ResourceType, ResourceID string
		Args                             map[string]any
		CommandDigest                    string
	}{p.Action, p.ResourceType, p.ResourceID, args, p.CommandDigest})
	return p
}

// ValidateCreateProposal is preliminary Kernel validation for
// CREATE_WORKFLOW: absent -> PLANNED. See
// docs/domain/workflow.md#first-slice-commands-and-legal-transitions.
func ValidateCreateProposal(cmd command.WorkflowCommandEnvelope, reg fixtures.Registry) (*command.GovernedCommandProposal, []string) {
	var reasons []string
	if reg.Organization().OrganizationID != cmd.OrganizationID || reg.Organization().Status != "ACTIVE" {
		reasons = append(reasons, command.ReasonOrganizationInactive)
	}
	if reg.Objective().ObjectiveID != cmd.ObjectiveID || reg.Objective().Status != "APPROVED" {
		reasons = append(reasons, command.ReasonObjectiveNotApproved)
	}
	if reg.WorkflowDefinition().DefinitionID != cmd.DefinitionID || !reg.WorkflowDefinition().Active {
		reasons = append(reasons, command.ReasonDefinitionInactive)
	}
	if !reg.Capability().Active {
		reasons = append(reasons, command.ReasonCapabilityInactive)
	}
	if len(reasons) > 0 {
		return nil, reasons
	}
	return newProposal(cmd, cmd.WorkflowID.String(), map[string]any{
		"objectiveId":  cmd.ObjectiveID,
		"definitionId": cmd.DefinitionID,
		"inputs":       cmd.Inputs,
	}), nil
}

// ValidateStartProposal is preliminary Kernel validation for
// START_WORKFLOW: PLANNED -> READY.
func ValidateStartProposal(cmd command.WorkflowCommandEnvelope, reg fixtures.Registry, current workflow.Workflow) (*command.GovernedCommandProposal, []string) {
	var reasons []string
	if !reg.WorkflowDefinition().Active || !reg.Capability().Active {
		reasons = append(reasons, command.ReasonDefinitionInactive)
	}
	if current.State != workflow.StatePlanned {
		reasons = append(reasons, command.ReasonIllegalState)
	}
	if cmd.ExpectedVersion == nil || *cmd.ExpectedVersion != current.Version {
		reasons = append(reasons, command.ReasonVersionMismatch)
	}
	if len(reasons) > 0 {
		return nil, reasons
	}
	return newProposal(cmd, cmd.WorkflowID.String(), map[string]any{
		"capabilityId":      reg.Capability().CapabilityDefinitionID,
		"capabilityVersion": reg.Capability().Version,
	}), nil
}

// ValidateResultProposal is preliminary Kernel validation shared by
// ACCEPT_WORKFLOW_RESULT and REJECT_WORKFLOW_RESULT: READY -> COMPLETED or
// READY -> FAILED, chosen by res.Outcome.
func ValidateResultProposal(cmd command.WorkflowCommandEnvelope, current workflow.Workflow, res result.Result, attemptWorkflowVersion int64) (*command.GovernedCommandProposal, []string) {
	var reasons []string
	if current.State != workflow.StateReady {
		reasons = append(reasons, command.ReasonIllegalState)
	}
	if cmd.ExpectedVersion == nil || *cmd.ExpectedVersion != current.Version {
		reasons = append(reasons, command.ReasonVersionMismatch)
	}
	if res.WorkflowID != cmd.WorkflowID || res.Accepted != nil {
		reasons = append(reasons, command.ReasonResultAlreadyDecided)
	}
	if attemptWorkflowVersion != current.Version {
		reasons = append(reasons, command.ReasonResultBindingMismatch)
	}
	switch cmd.CommandType {
	case command.AcceptWorkflowResult:
		if !res.Outcome.AcceptsResult() {
			reasons = append(reasons, command.ReasonResultOutcomeMismatch)
		}
	case command.RejectWorkflowResult:
		if !res.Outcome.RejectsResult() {
			reasons = append(reasons, command.ReasonResultOutcomeMismatch)
		}
	}
	if len(reasons) > 0 {
		return nil, reasons
	}
	return newProposal(cmd, cmd.WorkflowID.String(), map[string]any{
		"resultId": res.ResultID,
		"outcome":  res.Outcome,
	}), nil
}

// ValidateCancelProposal is preliminary Kernel validation for
// CANCEL_WORKFLOW: PLANNED -> CANCELLED or READY -> CANCELLED, requested
// only by the Workflow's initiating Principal (docs/domain/workflow.md:
// "Requested by an authorized Principal").
func ValidateCancelProposal(cmd command.WorkflowCommandEnvelope, current workflow.Workflow) (*command.GovernedCommandProposal, []string) {
	var reasons []string
	if current.State != workflow.StatePlanned && current.State != workflow.StateReady {
		reasons = append(reasons, command.ReasonIllegalState)
	}
	if cmd.ExpectedVersion == nil || *cmd.ExpectedVersion != current.Version {
		reasons = append(reasons, command.ReasonVersionMismatch)
	}
	if cmd.RequestingPrincipalID != current.InitiatingPrincipalID {
		reasons = append(reasons, command.ReasonPrincipalNotAuthorized)
	}
	if len(reasons) > 0 {
		return nil, reasons
	}
	return newProposal(cmd, cmd.WorkflowID.String(), map[string]any{
		"priorState": current.State,
	}), nil
}
