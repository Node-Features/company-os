package workflow

import (
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
)

// requiredPriorState is the exact 5-row legal-transition table in
// docs/domain/workflow.md#first-slice-commands-and-legal-transitions,
// shared by proposal.go and decision.go for every command. A nil entry
// means "no Workflow may exist" (CREATE_WORKFLOW only).
// CANCEL_WORKFLOW is deliberately absent from both maps below: it is the
// one command legal from two prior states (PLANNED or READY), which this
// single-prior-state table cannot express — see ValidateCancelProposal and
// FinalizeCancel, which encode its two-row legality directly instead.
var requiredPriorState = map[command.CommandType]*workflow.State{
	command.CreateWorkflow:       nil,
	command.StartWorkflow:        statePtr(workflow.StatePlanned),
	command.AcceptWorkflowResult: statePtr(workflow.StateReady),
	command.RejectWorkflowResult: statePtr(workflow.StateReady),
}

var nextState = map[command.CommandType]workflow.State{
	command.StartWorkflow:        workflow.StateReady,
	command.AcceptWorkflowResult: workflow.StateCompleted,
	command.RejectWorkflowResult: workflow.StateFailed,
}

func statePtr(s workflow.State) *workflow.State { return &s }
