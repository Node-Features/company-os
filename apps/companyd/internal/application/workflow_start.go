package application

import (
	"context"
	"errors"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/Node-Features/company-os/apps/companyd/internal/governance"
	kernelwf "github.com/Node-Features/company-os/apps/companyd/internal/kernel/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// StartWorkflow is the START_WORKFLOW use case: PLANNED -> READY, causing
// exactly one ExecutionIntent.
func (a *Application) StartWorkflow(ctx context.Context, req StartWorkflowRequest) Result {
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
		CommandType:           command.StartWorkflow,
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

	proposal, reasons := kernelwf.ValidateStartProposal(cmd, reg, *current)
	if proposal == nil {
		return a.store(ctx, cmd, Result{Outcome: Rejected, Reasons: reasons})
	}

	decision, denyResult, ok := a.evaluateGovernance(ctx, cmd, *proposal, true, governanceOptions{})
	if !ok {
		return a.store(ctx, cmd, denyResult)
	}

	kd, reasons := kernelwf.FinalizeStart(cmd, *proposal, decision, *current, reg, cmd.DeclaredTime)
	if kd == nil {
		return a.store(ctx, cmd, Result{Outcome: Rejected, Reasons: reasons})
	}

	next := *current
	next.Version = kd.NextVersion
	next.State = kd.NextState
	next.UpdatedAt = cmd.DeclaredTime

	if err := a.Repo.CommitTransition(ctx, &next, req.ExpectedVersion, kd.Events, kd.GovernanceDecisionID, kd.Intent, nil, nil, nil, false); err != nil {
		if errors.Is(err, ports.ErrConflict) {
			return a.store(ctx, cmd, Result{Outcome: Conflict, Reasons: []string{command.ReasonVersionMismatch}})
		}
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	// Step 11: notify only after commit — a recoverable wake-up hint, not
	// a precondition for the commit having already succeeded.
	if kd.Intent != nil {
		a.notify(kd.Intent.IntentID)
	}

	return a.store(ctx, cmd, Result{Outcome: Accepted, Workflow: viewOf(&next)})
}

// AuthorizeDispatch is the dispatch-time governance re-check
// (docs/architecture/application.md#dispatch-time-governance): immediately
// before Runtime dispatches a capability, Application re-evaluates
// Governance against the intent's current authority/policy/version. Runtime
// passes the intent it already claimed rather than this method reloading
// it, since ClaimDueIntents already returned it — see first-slice plan.
func (a *Application) AuthorizeDispatch(ctx context.Context, intent workflow.ExecutionIntent) (policy.GovernanceDecision, error) {
	reg := a.Fixtures
	requestID := uuid.New()

	current, err := a.Repo.LoadWorkflow(ctx, intent.OrganizationID, intent.WorkflowID)
	if err != nil {
		return policy.GovernanceDecision{}, err
	}

	proposalDigest := intent.IntentID.String() + ":" + string(current.State)
	trustedContextDigest := intent.IntentID.String()

	if current.Version != intent.WorkflowVersion {
		decision := policy.GovernanceDecision{
			DecisionID:     uuid.New(),
			OrganizationID: intent.OrganizationID,
			RequestID:      requestID,
			PrincipalID:    reg.TriggerPrincipal().PrincipalID,
			Action:         fixtures.GovernanceActionDispatch,
			ResourceType:   "ExecutionIntent",
			ResourceID:     intent.IntentID.String(),
			ProposalDigest: proposalDigest,
			PolicyVersion:  governance.PolicyVersion,
			Outcome:        policy.DecisionDenied,
		}
		reason := "stale intent version"
		decision.Reason = &reason
		_ = a.Repo.SaveGovernanceDecision(ctx, decision.DecisionID, decision.OrganizationID, decision.RequestID, decision.RequestID, decision.PrincipalID,
			decision.Action, decision.ResourceType, decision.ResourceID, decision.ProposalDigest, trustedContextDigest, decision.PolicyVersion, string(decision.AutonomyLevel), string(decision.Outcome), decision.MatchedRuleID, decision.Reason)
		return decision, nil
	}

	decision, err := governance.Evaluate(ctx, governance.Request{
		RequestID:            requestID,
		CorrelationID:        requestID,
		OrganizationID:       intent.OrganizationID,
		PrincipalID:          reg.TriggerPrincipal().PrincipalID,
		EvidencePresent:      true,
		Action:               fixtures.GovernanceActionDispatch,
		ResourceType:         "ExecutionIntent",
		ResourceID:           intent.IntentID.String(),
		ProposalDigest:       proposalDigest,
		TrustedContextDigest: trustedContextDigest,
	})
	if err != nil {
		return decision, err
	}
	_ = a.Repo.SaveGovernanceDecision(ctx, decision.DecisionID, decision.OrganizationID, decision.RequestID, decision.CorrelationID, decision.PrincipalID,
		decision.Action, decision.ResourceType, decision.ResourceID, decision.ProposalDigest, decision.TrustedContextDigest,
		decision.PolicyVersion, string(decision.AutonomyLevel), string(decision.Outcome), decision.MatchedRuleID, decision.Reason)
	return decision, nil
}
