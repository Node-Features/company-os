package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	objdomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/objective"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/governance"
	kernelobj "github.com/Node-Features/company-os/apps/companyd/internal/kernel/objective"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// errUnknownSourceType guards against a SourceType this Application build
// doesn't recognize — never reachable through the HTTP handler (which
// parses SourceType from a fixed set of accepted strings), but keeps
// validateSourceExists total rather than panicking on an unexpected value.
var errUnknownSourceType = errors.New("unknown source type")

// validateSourceExists confirms the cited Finding/Recommendation/
// Evaluation/ResourceEvaluation is real by loading it through the already-
// injected per-department repository — the same pattern IssueRecommendation
// (research_recommendation.go) already uses to load its Finding via
// a.Research.GetFinding. No new cross-department coupling: each branch
// calls straight into that department's own existing repository.
func (a *Application) validateSourceExists(ctx context.Context, orgID uuid.UUID, sourceType objdomain.SourceType, sourceID uuid.UUID) error {
	switch sourceType {
	case objdomain.SourceFinding:
		_, err := a.Research.GetFinding(ctx, orgID, sourceID)
		return err
	case objdomain.SourceRecommendation:
		_, err := a.Research.GetRecommendation(ctx, orgID, sourceID)
		return err
	case objdomain.SourceEvaluation:
		_, err := a.MonitoringEvaluation.GetEvaluation(ctx, orgID, sourceID)
		return err
	case objdomain.SourceResourceEvaluation:
		_, err := a.Finance.GetResourceEvaluation(ctx, orgID, sourceID)
		return err
	default:
		return fmt.Errorf("%w: %s", errUnknownSourceType, sourceType)
	}
}

// ProposeObjective is the PROPOSE_OBJECTIVE use case (ROADMAP.md Phase 4
// Slice 4, docs/architecture/departments.md's Objective creation gate): a
// Finding, Recommendation, Evaluation, or ResourceEvaluation may propose an
// Objective, never create one directly. governance/policy.go's
// "objective-propose" rule is unconditional AutonomyApprovalRequired, so
// every call here deterministically returns APPROVAL_REQUIRED — the actual
// Objective is only created by resumeProposeObjective, once
// Application.ResolveApproval supplies matching ApprovalEvidence.
func (a *Application) ProposeObjective(ctx context.Context, req ProposeObjectiveRequest) Result {
	orgID := a.Fixtures.Organization().OrganizationID

	if err := a.validateSourceExists(ctx, orgID, req.SourceType, req.SourceID); err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return Result{Outcome: Rejected, Reasons: []string{command.ReasonObjectiveSourceNotFound}}
		}
		if errors.Is(err, errUnknownSourceType) {
			return Result{Outcome: Invalid, Reasons: []string{err.Error()}}
		}
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	_, err := a.Objective.GetObjectiveBySource(ctx, orgID, req.SourceType, req.SourceID)
	alreadyProposed := err == nil
	if err != nil && !errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	cmd := command.ObjectiveProposalCommandEnvelope{
		SchemaVersion:           1,
		CommandID:               uuid.New(),
		RequestID:               req.RequestID,
		IdempotencyKey:          req.RequestID.String(),
		CommandType:             command.ProposeObjective,
		OrganizationID:          orgID,
		ObjectiveID:             uuid.New(),
		SourceType:              req.SourceType,
		SourceID:                req.SourceID,
		Title:                   req.Title,
		Intent:                  req.Intent,
		RequestingPrincipalID:   req.PrincipalID,
		RequestingPrincipalKind: req.PrincipalKind,
		DeclaredTime:            time.Now().UTC(),
		CorrelationID:           req.RequestID,
	}

	proposal, reasons := kernelobj.ValidateProposal(cmd, alreadyProposed)
	if proposal == nil {
		return Result{Outcome: Rejected, Reasons: reasons}
	}

	decision, gateResult, ok := a.evaluateObjectiveProposalGovernance(ctx, cmd, *proposal, nil)
	if !ok {
		return gateResult
	}

	// Structurally unreachable under this slice's always-ApprovalRequired
	// policy (documented above), implemented for completeness the same way
	// Governance's DENY branch is always available even when a rule set
	// never routes a real request into it.
	return a.createObjective(ctx, cmd, decision, nil, nil)
}

// resumeProposeObjective replays the exact original PROPOSE_OBJECTIVE
// envelope now that Approval evidence exists — same pattern
// resumeCancelWorkflow (approval_resolve.go) uses for Workflow commands:
// unmutated replay reproduces the identical ProposalDigest whenever
// nothing material changed, which is exactly when this should proceed.
func (a *Application) resumeProposeObjective(ctx context.Context, pc *command.PendingCommand, appr *approval.Approval) Result {
	var cmd command.ObjectiveProposalCommandEnvelope
	if err := json.Unmarshal(pc.GovernedPayload, &cmd); err != nil {
		return Result{Outcome: Unavailable, Reasons: []string{"corrupt governed payload: " + err.Error()}}
	}

	_, err := a.Objective.GetObjectiveBySource(ctx, cmd.OrganizationID, cmd.SourceType, cmd.SourceID)
	alreadyProposed := err == nil
	if err != nil && !errors.Is(err, ports.ErrNotFound) {
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}

	proposal, reasons := kernelobj.ValidateProposal(cmd, alreadyProposed)
	if proposal == nil {
		return Result{Outcome: Rejected, Reasons: reasons}
	}

	decision, gateResult, ok := a.evaluateObjectiveProposalGovernance(ctx, cmd, *proposal, &governance.ApprovalEvidence{
		ApprovalID:     appr.ApprovalID,
		ProposalDigest: proposal.ProposalDigest,
	})
	if !ok {
		return gateResult
	}

	return a.createObjective(ctx, cmd, decision, &pc.PendingCommandID, &appr.ApprovalID)
}

func (a *Application) createObjective(ctx context.Context, cmd command.ObjectiveProposalCommandEnvelope, decision policy.GovernanceDecision, closePendingCommand, consumeApproval *uuid.UUID) Result {
	obj := &objdomain.Objective{
		ObjectiveID:          cmd.ObjectiveID,
		OrganizationID:       cmd.OrganizationID,
		Title:                cmd.Title,
		Intent:               cmd.Intent,
		Status:               objdomain.StatusProposed,
		OwnerPrincipalID:     cmd.RequestingPrincipalID,
		SourceType:           cmd.SourceType,
		SourceID:             cmd.SourceID,
		GovernanceDecisionID: decision.DecisionID,
		ApprovalID:           consumeApproval,
		CreatedAt:            time.Now().UTC(),
	}
	if err := a.Objective.CreateObjective(ctx, obj, closePendingCommand, consumeApproval); err != nil {
		if errors.Is(err, ports.ErrConflict) {
			return Result{Outcome: Conflict, Reasons: []string{command.ReasonObjectiveAlreadyProposed}}
		}
		return Result{Outcome: Unavailable, Reasons: []string{err.Error()}}
	}
	objectiveID := cmd.ObjectiveID
	return Result{Outcome: Accepted, ResourceID: &objectiveID}
}

// GetObjective is the read projection served by GET /v1/objectives/{objectiveId}.
func (a *Application) GetObjective(ctx context.Context, objectiveID uuid.UUID) (objdomain.Objective, error) {
	orgID := a.Fixtures.Organization().OrganizationID
	return a.Objective.GetObjective(ctx, orgID, objectiveID)
}
