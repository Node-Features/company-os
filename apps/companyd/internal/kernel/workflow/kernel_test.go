package workflow

import (
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/google/uuid"
)

func testReg() fixtures.Registry { return fixtures.NewRegistry() }

func allowDecision(digest string) policy.GovernanceDecision {
	return policy.GovernanceDecision{DecisionID: uuid.New(), Outcome: policy.DecisionAutomatic, ProposalDigest: digest}
}

func TestValidateCreateProposal_Legal(t *testing.T) {
	reg := testReg()
	cmd := command.WorkflowCommandEnvelope{
		CommandID: uuid.New(), CommandType: command.CreateWorkflow, OrganizationID: reg.Organization().OrganizationID,
		WorkflowID: uuid.New(), ObjectiveID: reg.Objective().ObjectiveID,
		DefinitionID: reg.WorkflowDefinition().DefinitionID, RequestingPrincipalID: reg.TriggerPrincipal().PrincipalID,
		DeclaredTime: time.Now(),
	}
	proposal, reasons := ValidateCreateProposal(cmd, reg)
	if proposal == nil {
		t.Fatalf("expected proposal, got rejection reasons: %v", reasons)
	}
	if proposal.Action != "workflow.create" {
		t.Errorf("action = %q, want workflow.create", proposal.Action)
	}
}

func TestValidateCreateProposal_InactiveOrganization(t *testing.T) {
	reg := testReg()
	cmd := command.WorkflowCommandEnvelope{
		CommandID: uuid.New(), OrganizationID: uuid.New(), // mismatched org
		WorkflowID: uuid.New(), ObjectiveID: reg.Objective().ObjectiveID,
		DefinitionID: reg.WorkflowDefinition().DefinitionID, DeclaredTime: time.Now(),
	}
	proposal, reasons := ValidateCreateProposal(cmd, reg)
	if proposal != nil {
		t.Fatalf("expected rejection, got proposal")
	}
	if len(reasons) == 0 {
		t.Fatal("expected at least one rejection reason")
	}
}

func TestFinalizeCreate_AbsentToPlanned(t *testing.T) {
	reg := testReg()
	cmd := command.WorkflowCommandEnvelope{
		CommandID: uuid.New(), OrganizationID: reg.Organization().OrganizationID,
		WorkflowID: uuid.New(), ObjectiveID: reg.Objective().ObjectiveID,
		DefinitionID: reg.WorkflowDefinition().DefinitionID, RequestingPrincipalID: reg.TriggerPrincipal().PrincipalID,
		DeclaredTime: time.Now(), CorrelationID: uuid.New(),
	}
	proposal, _ := ValidateCreateProposal(cmd, reg)
	decision := allowDecision(proposal.ProposalDigest)

	kd, reasons := FinalizeCreate(cmd, *proposal, decision, cmd.DeclaredTime)
	if kd == nil {
		t.Fatalf("expected decision, got rejection: %v", reasons)
	}
	if kd.NextState != workflow.StatePlanned || kd.NextVersion != 1 {
		t.Errorf("got state=%s version=%d, want PLANNED/1", kd.NextState, kd.NextVersion)
	}
	if len(kd.Events) != 1 {
		t.Errorf("got %d events, want 1", len(kd.Events))
	}
}

func TestFinalizeCreate_StaleGovernanceDigestRejected(t *testing.T) {
	reg := testReg()
	cmd := command.WorkflowCommandEnvelope{
		CommandID: uuid.New(), OrganizationID: reg.Organization().OrganizationID,
		WorkflowID: uuid.New(), ObjectiveID: reg.Objective().ObjectiveID,
		DefinitionID: reg.WorkflowDefinition().DefinitionID, DeclaredTime: time.Now(),
	}
	proposal, _ := ValidateCreateProposal(cmd, reg)
	decision := allowDecision("mismatched-digest")

	kd, reasons := FinalizeCreate(cmd, *proposal, decision, cmd.DeclaredTime)
	if kd != nil {
		t.Fatal("expected rejection on stale governance digest")
	}
	if reasons[0] != command.ReasonGovernanceStale {
		t.Errorf("reason = %q, want %q", reasons[0], command.ReasonGovernanceStale)
	}
}

func TestStartWorkflow_PlannedToReady_ProducesExactlyOneIntent(t *testing.T) {
	reg := testReg()
	current := workflow.Workflow{
		OrganizationID: reg.Organization().OrganizationID, WorkflowID: uuid.New(), Version: 1,
		DefinitionID: reg.WorkflowDefinition().DefinitionID, State: workflow.StatePlanned,
		Inputs: map[string]any{"prompt": "hi"},
	}
	expected := current.Version
	cmd := command.WorkflowCommandEnvelope{
		CommandID: uuid.New(), CommandType: command.StartWorkflow, OrganizationID: current.OrganizationID, WorkflowID: current.WorkflowID,
		ExpectedVersion: &expected, DeclaredTime: time.Now(),
	}
	proposal, reasons := ValidateStartProposal(cmd, reg, current)
	if proposal == nil {
		t.Fatalf("expected proposal, got: %v", reasons)
	}
	decision := allowDecision(proposal.ProposalDigest)
	kd, reasons := FinalizeStart(cmd, *proposal, decision, current, reg, cmd.DeclaredTime)
	if kd == nil {
		t.Fatalf("expected decision, got rejection: %v", reasons)
	}
	if kd.NextState != workflow.StateReady || kd.NextVersion != 2 {
		t.Errorf("got state=%s version=%d, want READY/2", kd.NextState, kd.NextVersion)
	}
	if kd.Intent == nil {
		t.Fatal("expected exactly one ExecutionIntent, got nil")
	}
}

func TestStartWorkflow_WrongVersionRejected(t *testing.T) {
	reg := testReg()
	current := workflow.Workflow{
		OrganizationID: reg.Organization().OrganizationID, WorkflowID: uuid.New(), Version: 3,
		DefinitionID: reg.WorkflowDefinition().DefinitionID, State: workflow.StatePlanned,
	}
	staleExpected := int64(1)
	cmd := command.WorkflowCommandEnvelope{
		CommandID: uuid.New(), OrganizationID: current.OrganizationID, WorkflowID: current.WorkflowID,
		ExpectedVersion: &staleExpected, DeclaredTime: time.Now(),
	}
	proposal, reasons := ValidateStartProposal(cmd, reg, current)
	if proposal != nil {
		t.Fatal("expected rejection on version mismatch")
	}
	found := false
	for _, r := range reasons {
		if r == command.ReasonVersionMismatch {
			found = true
		}
	}
	if !found {
		t.Errorf("reasons = %v, want to include %q", reasons, command.ReasonVersionMismatch)
	}
}

func TestAcceptWorkflowResult_ReadyToCompleted(t *testing.T) {
	reg := testReg()
	current := workflow.Workflow{
		OrganizationID: reg.Organization().OrganizationID, WorkflowID: uuid.New(), Version: 2,
		State: workflow.StateReady,
	}
	expected := current.Version
	res := result.Result{ResultID: uuid.New(), WorkflowID: current.WorkflowID, Outcome: result.OutcomeSucceeded}
	cmd := command.WorkflowCommandEnvelope{
		CommandID: uuid.New(), CommandType: command.AcceptWorkflowResult, OrganizationID: current.OrganizationID,
		WorkflowID: current.WorkflowID, ExpectedVersion: &expected, ResultID: &res.ResultID, DeclaredTime: time.Now(),
	}
	proposal, reasons := ValidateResultProposal(cmd, current, res, current.Version)
	if proposal == nil {
		t.Fatalf("expected proposal, got: %v", reasons)
	}
	decision := allowDecision(proposal.ProposalDigest)
	kd, reasons := FinalizeAccept(cmd, *proposal, decision, current, res, cmd.DeclaredTime)
	if kd == nil {
		t.Fatalf("expected decision, got rejection: %v", reasons)
	}
	if kd.NextState != workflow.StateCompleted {
		t.Errorf("state = %s, want COMPLETED", kd.NextState)
	}
}

func TestRejectWorkflowResult_FailedTimedOutPartialAllRejectToFailed(t *testing.T) {
	for _, outcome := range []result.Outcome{result.OutcomeFailed, result.OutcomeTimedOut, result.OutcomePartial} {
		t.Run(string(outcome), func(t *testing.T) {
			reg := testReg()
			_ = reg
			current := workflow.Workflow{OrganizationID: uuid.New(), WorkflowID: uuid.New(), Version: 2, State: workflow.StateReady}
			expected := current.Version
			res := result.Result{ResultID: uuid.New(), WorkflowID: current.WorkflowID, Outcome: outcome}
			cmd := command.WorkflowCommandEnvelope{
				CommandID: uuid.New(), CommandType: command.RejectWorkflowResult, OrganizationID: current.OrganizationID,
				WorkflowID: current.WorkflowID, ExpectedVersion: &expected, ResultID: &res.ResultID, DeclaredTime: time.Now(),
			}
			proposal, reasons := ValidateResultProposal(cmd, current, res, current.Version)
			if proposal == nil {
				t.Fatalf("expected proposal, got: %v", reasons)
			}
			decision := allowDecision(proposal.ProposalDigest)
			kd, reasons := FinalizeReject(cmd, *proposal, decision, current, res, cmd.DeclaredTime)
			if kd == nil {
				t.Fatalf("expected decision, got rejection: %v", reasons)
			}
			if kd.NextState != workflow.StateFailed {
				t.Errorf("state = %s, want FAILED", kd.NextState)
			}
		})
	}
}

func TestValidateResultProposal_IndeterminateNeitherAcceptsNorRejects(t *testing.T) {
	current := workflow.Workflow{OrganizationID: uuid.New(), WorkflowID: uuid.New(), Version: 2, State: workflow.StateReady}
	expected := current.Version
	res := result.Result{ResultID: uuid.New(), WorkflowID: current.WorkflowID, Outcome: result.OutcomeIndeterminate}

	for _, ct := range []command.CommandType{command.AcceptWorkflowResult, command.RejectWorkflowResult} {
		cmd := command.WorkflowCommandEnvelope{
			CommandID: uuid.New(), CommandType: ct, OrganizationID: current.OrganizationID,
			WorkflowID: current.WorkflowID, ExpectedVersion: &expected, ResultID: &res.ResultID, DeclaredTime: time.Now(),
		}
		proposal, reasons := ValidateResultProposal(cmd, current, res, current.Version)
		if proposal != nil {
			t.Errorf("%s: expected rejection for INDETERMINATE result, got proposal", ct)
		}
		if len(reasons) == 0 {
			t.Errorf("%s: expected rejection reasons", ct)
		}
	}
}

func TestCancelWorkflow_PlannedOrReadyToCancelled(t *testing.T) {
	for _, prior := range []workflow.State{workflow.StatePlanned, workflow.StateReady} {
		t.Run(string(prior), func(t *testing.T) {
			principalID := uuid.New()
			current := workflow.Workflow{
				OrganizationID: uuid.New(), WorkflowID: uuid.New(), Version: 2, State: prior,
				InitiatingPrincipalID: principalID,
			}
			expected := current.Version
			cmd := command.WorkflowCommandEnvelope{
				CommandID: uuid.New(), CommandType: command.CancelWorkflow, OrganizationID: current.OrganizationID,
				WorkflowID: current.WorkflowID, ExpectedVersion: &expected, RequestingPrincipalID: principalID,
				DeclaredTime: time.Now(),
			}
			proposal, reasons := ValidateCancelProposal(cmd, current)
			if proposal == nil {
				t.Fatalf("expected proposal, got: %v", reasons)
			}
			decision := allowDecision(proposal.ProposalDigest)
			kd, reasons := FinalizeCancel(cmd, *proposal, decision, current, cmd.DeclaredTime)
			if kd == nil {
				t.Fatalf("expected decision, got rejection: %v", reasons)
			}
			if kd.NextState != workflow.StateCancelled || kd.NextVersion != 3 {
				t.Errorf("got state=%s version=%d, want CANCELLED/3", kd.NextState, kd.NextVersion)
			}
			if len(kd.Events) != 1 || kd.Events[0].EventType != event.TypeWorkflowCancelled {
				t.Errorf("expected one WORKFLOW_CANCELLED event, got %+v", kd.Events)
			}
		})
	}
}

func TestValidateCancelProposal_TerminalStateRejected(t *testing.T) {
	for _, terminal := range []workflow.State{workflow.StateCompleted, workflow.StateFailed, workflow.StateCancelled} {
		t.Run(string(terminal), func(t *testing.T) {
			principalID := uuid.New()
			current := workflow.Workflow{
				OrganizationID: uuid.New(), WorkflowID: uuid.New(), Version: 3, State: terminal,
				InitiatingPrincipalID: principalID,
			}
			expected := current.Version
			cmd := command.WorkflowCommandEnvelope{
				CommandID: uuid.New(), CommandType: command.CancelWorkflow, OrganizationID: current.OrganizationID,
				WorkflowID: current.WorkflowID, ExpectedVersion: &expected, RequestingPrincipalID: principalID,
				DeclaredTime: time.Now(),
			}
			proposal, reasons := ValidateCancelProposal(cmd, current)
			if proposal != nil {
				t.Fatalf("expected rejection from terminal state %s, got proposal", terminal)
			}
			found := false
			for _, r := range reasons {
				if r == command.ReasonIllegalState {
					found = true
				}
			}
			if !found {
				t.Errorf("reasons = %v, want to include %q", reasons, command.ReasonIllegalState)
			}
		})
	}
}

// TestValidateCancelProposal_PrincipalMismatchStillProducesProposal locks
// in that the Kernel no longer rejects a non-initiating Principal itself —
// that check moved to Governance (governance.evaluateGovernance's
// ResourceOwnerPrincipalID, see governance/evaluate_test.go's
// TestEvaluate_DeniesResourceOwnerMismatch) because ownership is an
// authorization decision, not a state-transition legality one. The Kernel
// only cares that state and version are legal.
func TestValidateCancelProposal_PrincipalMismatchStillProducesProposal(t *testing.T) {
	current := workflow.Workflow{
		OrganizationID: uuid.New(), WorkflowID: uuid.New(), Version: 1, State: workflow.StatePlanned,
		InitiatingPrincipalID: uuid.New(),
	}
	expected := current.Version
	cmd := command.WorkflowCommandEnvelope{
		CommandID: uuid.New(), CommandType: command.CancelWorkflow, OrganizationID: current.OrganizationID,
		WorkflowID: current.WorkflowID, ExpectedVersion: &expected,
		RequestingPrincipalID: uuid.New(), // a different Principal than InitiatingPrincipalID
		DeclaredTime:          time.Now(),
	}
	proposal, reasons := ValidateCancelProposal(cmd, current)
	if proposal == nil {
		t.Fatalf("expected a proposal despite the Principal mismatch (Kernel validates legality only), got rejection: %v", reasons)
	}
}

func TestValidateCancelProposal_WrongVersionRejected(t *testing.T) {
	principalID := uuid.New()
	current := workflow.Workflow{
		OrganizationID: uuid.New(), WorkflowID: uuid.New(), Version: 3, State: workflow.StateReady,
		InitiatingPrincipalID: principalID,
	}
	staleExpected := int64(1)
	cmd := command.WorkflowCommandEnvelope{
		CommandID: uuid.New(), CommandType: command.CancelWorkflow, OrganizationID: current.OrganizationID,
		WorkflowID: current.WorkflowID, ExpectedVersion: &staleExpected, RequestingPrincipalID: principalID,
		DeclaredTime: time.Now(),
	}
	proposal, reasons := ValidateCancelProposal(cmd, current)
	if proposal != nil {
		t.Fatal("expected rejection on version mismatch")
	}
	found := false
	for _, r := range reasons {
		if r == command.ReasonVersionMismatch {
			found = true
		}
	}
	if !found {
		t.Errorf("reasons = %v, want to include %q", reasons, command.ReasonVersionMismatch)
	}
}
