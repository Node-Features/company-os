package workflow

import (
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/google/uuid"
)

// commandTTL bounds how long a GovernedCommandProposal remains valid for
// final Kernel decision after preliminary validation (command.md's
// "proposed effect classification and expiry").
const commandTTL = 5 * time.Minute

// Kernel functions never read wall-clock time, network state, or
// environment variables implicitly (docs/architecture/kernel.md) — every
// function here takes declaredTime as an explicit parameter and does no I/O.

// FinalizeCreate is final Kernel validation for CREATE_WORKFLOW: re-verifies
// the Governance decision is ALLOW against the unchanged proposal, then
// produces the accepted absent -> PLANNED transition.
func FinalizeCreate(cmd command.WorkflowCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string) {
	if reasons := verifyAllow(proposal, decision); reasons != nil {
		return nil, reasons
	}
	evt := event.DomainEvent{
		EventID:        uuid.New(),
		OrganizationID: cmd.OrganizationID,
		EventType:      event.TypeWorkflowCreated,
		SchemaVersion:  1,
		SubjectType:    "Workflow",
		SubjectID:      cmd.WorkflowID,
		SubjectVersion: 1,
		OccurredAt:     declaredTime,
		CorrelationID:  cmd.CorrelationID,
		CausationID:    cmd.CausationID,
		WorkflowID:     &cmd.WorkflowID,
		ObjectiveID:    &cmd.ObjectiveID,
		PrincipalID:    &cmd.RequestingPrincipalID,
		Payload:        map[string]any{"definitionId": cmd.DefinitionID, "definitionVersion": cmd.DefinitionVersion},
	}
	return &command.KernelDecisionEnvelope{
		DecisionID:           uuid.New(),
		SchemaVersion:        1,
		ProposalID:           proposal.ProposalID,
		ProposalDigest:       proposal.ProposalDigest,
		OrganizationID:       cmd.OrganizationID,
		AggregateID:          cmd.WorkflowID,
		PriorVersion:         nil,
		Accepted:             true,
		NextState:            workflow.StatePlanned,
		NextVersion:          1,
		Events:               []event.DomainEvent{evt},
		Intent:               nil,
		GovernanceDecisionID: decision.DecisionID,
		DecidedAt:            declaredTime,
		CorrelationID:        cmd.CorrelationID,
		CausationID:          cmd.CausationID,
	}, nil
}

// FinalizeStart is final Kernel validation for START_WORKFLOW: PLANNED ->
// READY, producing exactly one ExecutionIntent for the fixture capability.
func FinalizeStart(cmd command.WorkflowCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, current workflow.Workflow, reg fixtures.Registry, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string) {
	if reasons := verifyAllow(proposal, decision); reasons != nil {
		return nil, reasons
	}
	nextVersion := current.Version + 1
	evt := event.DomainEvent{
		EventID:        uuid.New(),
		OrganizationID: cmd.OrganizationID,
		EventType:      event.TypeWorkflowStarted,
		SchemaVersion:  1,
		SubjectType:    "Workflow",
		SubjectID:      cmd.WorkflowID,
		SubjectVersion: nextVersion,
		OccurredAt:     declaredTime,
		CorrelationID:  cmd.CorrelationID,
		CausationID:    cmd.CausationID,
		WorkflowID:     &cmd.WorkflowID,
		PrincipalID:    &cmd.RequestingPrincipalID,
		Payload:        map[string]any{},
	}
	intent := &workflow.ExecutionIntent{
		IntentID:                    uuid.New(),
		OrganizationID:              cmd.OrganizationID,
		WorkflowID:                  cmd.WorkflowID,
		WorkflowVersion:             nextVersion,
		CapabilityDefinitionID:      reg.Capability().CapabilityDefinitionID,
		CapabilityDefinitionVersion: reg.Capability().Version,
		GovernanceDecisionID:        decision.DecisionID,
		IdempotencyKey:              cmd.IdempotencyKey,
		DueAt:                       declaredTime,
		Inputs:                      current.Inputs,
	}
	return &command.KernelDecisionEnvelope{
		DecisionID:           uuid.New(),
		SchemaVersion:        1,
		ProposalID:           proposal.ProposalID,
		ProposalDigest:       proposal.ProposalDigest,
		OrganizationID:       cmd.OrganizationID,
		AggregateID:          cmd.WorkflowID,
		PriorVersion:         &current.Version,
		Accepted:             true,
		NextState:            workflow.StateReady,
		NextVersion:          nextVersion,
		Events:               []event.DomainEvent{evt},
		Intent:               intent,
		GovernanceDecisionID: decision.DecisionID,
		DecidedAt:            declaredTime,
		CorrelationID:        cmd.CorrelationID,
		CausationID:          cmd.CausationID,
	}, nil
}

// FinalizeAccept is final Kernel validation for ACCEPT_WORKFLOW_RESULT:
// READY -> COMPLETED.
func FinalizeAccept(cmd command.WorkflowCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, current workflow.Workflow, res result.Result, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string) {
	return finalizeResult(cmd, proposal, decision, current, res, event.TypeWorkflowCompleted, workflow.StateCompleted, declaredTime)
}

// FinalizeReject is final Kernel validation for REJECT_WORKFLOW_RESULT:
// READY -> FAILED.
func FinalizeReject(cmd command.WorkflowCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, current workflow.Workflow, res result.Result, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string) {
	return finalizeResult(cmd, proposal, decision, current, res, event.TypeWorkflowFailed, workflow.StateFailed, declaredTime)
}

func finalizeResult(cmd command.WorkflowCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, current workflow.Workflow, res result.Result, eventType string, next workflow.State, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string) {
	if reasons := verifyAllow(proposal, decision); reasons != nil {
		return nil, reasons
	}
	nextVersion := current.Version + 1
	evt := event.DomainEvent{
		EventID:        uuid.New(),
		OrganizationID: cmd.OrganizationID,
		EventType:      eventType,
		SchemaVersion:  1,
		SubjectType:    "Workflow",
		SubjectID:      cmd.WorkflowID,
		SubjectVersion: nextVersion,
		OccurredAt:     declaredTime,
		CorrelationID:  cmd.CorrelationID,
		CausationID:    cmd.CausationID,
		WorkflowID:     &cmd.WorkflowID,
		PrincipalID:    &cmd.RequestingPrincipalID,
		Payload:        map[string]any{"resultId": res.ResultID, "outcome": res.Outcome},
	}
	return &command.KernelDecisionEnvelope{
		DecisionID:           uuid.New(),
		SchemaVersion:        1,
		ProposalID:           proposal.ProposalID,
		ProposalDigest:       proposal.ProposalDigest,
		OrganizationID:       cmd.OrganizationID,
		AggregateID:          cmd.WorkflowID,
		PriorVersion:         &current.Version,
		Accepted:             true,
		NextState:            next,
		NextVersion:          nextVersion,
		Events:               []event.DomainEvent{evt},
		Intent:               nil,
		GovernanceDecisionID: decision.DecisionID,
		DecidedAt:            declaredTime,
		CorrelationID:        cmd.CorrelationID,
		CausationID:          cmd.CausationID,
	}, nil
}

// FinalizeCancel is final Kernel validation for CANCEL_WORKFLOW: PLANNED ->
// CANCELLED or READY -> CANCELLED. If current was READY, the caller is
// responsible for recording durable cancellation of the outstanding
// ExecutionIntent per docs/domain/execution.md's invariants — this
// function does not assume the in-flight attempt stopped immediately.
func FinalizeCancel(cmd command.WorkflowCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, current workflow.Workflow, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string) {
	if reasons := verifyAllow(proposal, decision); reasons != nil {
		return nil, reasons
	}
	nextVersion := current.Version + 1
	evt := event.DomainEvent{
		EventID:        uuid.New(),
		OrganizationID: cmd.OrganizationID,
		EventType:      event.TypeWorkflowCancelled,
		SchemaVersion:  1,
		SubjectType:    "Workflow",
		SubjectID:      cmd.WorkflowID,
		SubjectVersion: nextVersion,
		OccurredAt:     declaredTime,
		CorrelationID:  cmd.CorrelationID,
		CausationID:    cmd.CausationID,
		WorkflowID:     &cmd.WorkflowID,
		PrincipalID:    &cmd.RequestingPrincipalID,
		Payload:        map[string]any{"priorState": current.State},
	}
	return &command.KernelDecisionEnvelope{
		DecisionID:           uuid.New(),
		SchemaVersion:        1,
		ProposalID:           proposal.ProposalID,
		ProposalDigest:       proposal.ProposalDigest,
		OrganizationID:       cmd.OrganizationID,
		AggregateID:          cmd.WorkflowID,
		PriorVersion:         &current.Version,
		Accepted:             true,
		NextState:            workflow.StateCancelled,
		NextVersion:          nextVersion,
		Events:               []event.DomainEvent{evt},
		Intent:               nil,
		GovernanceDecisionID: decision.DecisionID,
		DecidedAt:            declaredTime,
		CorrelationID:        cmd.CorrelationID,
		CausationID:          cmd.CausationID,
	}, nil
}

func verifyAllow(proposal command.GovernedCommandProposal, decision policy.GovernanceDecision) []string {
	if decision.Outcome != policy.DecisionAllow {
		return []string{command.ReasonGovernanceDenied}
	}
	if decision.ProposalDigest != proposal.ProposalDigest {
		return []string{command.ReasonGovernanceStale}
	}
	return nil
}
