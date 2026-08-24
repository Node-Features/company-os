package application

import (
	"context"
	"errors"
	"testing"
	"time"

	objdomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/objective"
	researchdomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/research"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// makeEvaluationSource creates one real M&E Evaluation and returns its ID —
// a minimal SourceType: EVALUATION source for ProposeObjective tests.
func makeEvaluationSource(t *testing.T, app *Application, principalID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	_, _ = startedReadyWorkflow(t, app)
	_, resultID := submitFakeResultWithProvider(t, app, result.OutcomeSucceeded, "integration-test", "integration-test", 0, 0)
	metric := app.RecordMetric(ctx, RecordMetricRequest{RequestID: uuid.New(), PrincipalID: principalID, ResultID: resultID})
	if metric.Outcome != Accepted || metric.ResourceID == nil {
		t.Fatalf("RecordMetric outcome = %s (reasons: %v)", metric.Outcome, metric.Reasons)
	}
	evaluation := app.RunEvaluation(ctx, RunEvaluationRequest{RequestID: uuid.New(), PrincipalID: principalID, MetricIDs: []uuid.UUID{*metric.ResourceID}})
	if evaluation.Outcome != Accepted || evaluation.ResourceID == nil {
		t.Fatalf("RunEvaluation outcome = %s (reasons: %v)", evaluation.Outcome, evaluation.Reasons)
	}
	return *evaluation.ResourceID
}

// makeResourceEvaluationSource creates one real Finance ResourceEvaluation
// and returns its ID — a minimal SourceType: RESOURCE_EVALUATION source.
func makeResourceEvaluationSource(t *testing.T, app *Application, principalID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	_, _ = startedReadyWorkflow(t, app)
	_, resultID := submitFakeResultWithProvider(t, app, result.OutcomeSucceeded, "gemini", "gemini-2.5-flash", 100, 50)
	usage := app.RecordResourceUsage(ctx, RecordResourceUsageRequest{RequestID: uuid.New(), PrincipalID: principalID, ResultID: resultID})
	if usage.Outcome != Accepted || usage.ResourceID == nil {
		t.Fatalf("RecordResourceUsage outcome = %s (reasons: %v)", usage.Outcome, usage.Reasons)
	}
	evaluation := app.RunResourceEvaluation(ctx, RunResourceEvaluationRequest{RequestID: uuid.New(), PrincipalID: principalID})
	if evaluation.Outcome != Accepted || evaluation.ResourceID == nil {
		t.Fatalf("RunResourceEvaluation outcome = %s (reasons: %v)", evaluation.Outcome, evaluation.Reasons)
	}
	return *evaluation.ResourceID
}

// makeFindingSource drives Research's Signal->Question->Evidence->Finding
// chain (same shape as TestIntegration_ResearchLoop_SignalToRecommendation)
// and returns the Finding's ID — a minimal SourceType: FINDING source.
func makeFindingSource(t *testing.T, app *Application, principalID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	signal := app.SubmitSignal(ctx, SubmitSignalRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: researchdomain.SourceTypeProviderModelChange, Description: "signal for objective-proposal test",
	})
	question := app.OpenResearchQuestion(ctx, OpenResearchQuestionRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SignalID: *signal.ResourceID, Text: "question for objective-proposal test",
	})
	evidence := app.RecordEvidence(ctx, RecordEvidenceRequest{
		RequestID: uuid.New(), PrincipalID: principalID, QuestionID: *question.ResourceID,
		Source: "https://example.test/evidence", Content: "evidence for objective-proposal test", RetrievedAt: time.Now().UTC(),
	})
	finding := app.PublishFinding(ctx, PublishFindingRequest{
		RequestID: uuid.New(), PrincipalID: principalID, QuestionID: *question.ResourceID,
		Claim: "claim for objective-proposal test", EvidenceIDs: []uuid.UUID{*evidence.ResourceID},
	})
	if finding.Outcome != Accepted || finding.ResourceID == nil {
		t.Fatalf("PublishFinding outcome = %s (reasons: %v)", finding.Outcome, finding.Reasons)
	}
	return *finding.ResourceID
}

// makeRecommendationSource extends makeFindingSource one step further —
// a minimal SourceType: RECOMMENDATION source.
func makeRecommendationSource(t *testing.T, app *Application, principalID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	findingID := makeFindingSource(t, app, principalID)
	recommendation := app.IssueRecommendation(ctx, IssueRecommendationRequest{
		RequestID: uuid.New(), PrincipalID: principalID, FindingID: findingID,
		ProposedAction: "proposed action for objective-proposal test",
	})
	if recommendation.Outcome != Accepted || recommendation.ResourceID == nil {
		t.Fatalf("IssueRecommendation outcome = %s (reasons: %v)", recommendation.Outcome, recommendation.Reasons)
	}
	return *recommendation.ResourceID
}

// TestIntegration_ProposeObjective_RequiresApprovalThenCreatesOnResolve
// proves ROADMAP.md Phase 4 Slice 4's full gate end to end against the real
// database, using an EVALUATION source: ProposeObjective deterministically
// returns APPROVAL_REQUIRED (governance/policy.go's unconditional
// AutonomyApprovalRequired "objective-propose" rule), and only
// Application.ResolveApproval's replay actually creates the Objective.
func TestIntegration_ProposeObjective_RequiresApprovalThenCreatesOnResolve(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	sourceID := makeEvaluationSource(t, app, principalID)

	propose := app.ProposeObjective(ctx, ProposeObjectiveRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: objdomain.SourceEvaluation, SourceID: sourceID,
		Title: "Improve bounded-text-generation success rate", Intent: "Raise success rate above the PASS threshold",
	})
	if propose.Outcome != ApprovalRequired || propose.ApprovalID == nil {
		t.Fatalf("ProposeObjective outcome = %s (reasons: %v), want APPROVAL_REQUIRED with an ApprovalID", propose.Outcome, propose.Reasons)
	}

	resolve := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *propose.ApprovalID, Approve: true, DecidingPrincipal: app.Fixtures.ApproverPrincipal()})
	if resolve.Outcome != Accepted || resolve.ResourceID == nil {
		t.Fatalf("ResolveApproval(approve) outcome = %s (reasons: %v)", resolve.Outcome, resolve.Reasons)
	}
	objectiveID := *resolve.ResourceID

	obj, err := app.GetObjective(ctx, objectiveID)
	if err != nil {
		t.Fatalf("GetObjective: %v", err)
	}
	if obj.Status != objdomain.StatusProposed {
		t.Fatalf("Objective.Status = %s, want PROPOSED", obj.Status)
	}
	if obj.SourceType != objdomain.SourceEvaluation || obj.SourceID != sourceID {
		t.Fatalf("Objective.SourceType/SourceID = %s/%s, want EVALUATION/%s", obj.SourceType, obj.SourceID, sourceID)
	}
	if obj.ApprovalID == nil || *obj.ApprovalID != *propose.ApprovalID {
		t.Fatalf("Objective.ApprovalID = %v, want %s", obj.ApprovalID, *propose.ApprovalID)
	}

	// Double-resolve of the same, now-consumed Approval must conflict, not
	// silently succeed or create a second Objective — same guard
	// TestIntegration_ResolveApproval_UnknownApprovalConflict proves for
	// CANCEL_WORKFLOW's approval path.
	again := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *propose.ApprovalID, Approve: true, DecidingPrincipal: app.Fixtures.ApproverPrincipal()})
	if again.Outcome != Conflict {
		t.Fatalf("second ResolveApproval outcome = %s, want CONFLICT", again.Outcome)
	}
}

// TestIntegration_ProposeObjective_RejectLeavesNoObjective proves the
// reject path (a RESOURCE_EVALUATION source this time) never creates an
// Objective — departments.md: "REQUIRE_APPROVAL becomes ESCALATED, not
// retryable" for the exact proposed version; nothing here retries it.
func TestIntegration_ProposeObjective_RejectLeavesNoObjective(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	sourceID := makeResourceEvaluationSource(t, app, principalID)

	propose := app.ProposeObjective(ctx, ProposeObjectiveRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: objdomain.SourceResourceEvaluation, SourceID: sourceID,
		Title: "Reduce cost per successful result", Intent: "Bring effective cost under budget",
	})
	if propose.Outcome != ApprovalRequired || propose.ApprovalID == nil {
		t.Fatalf("ProposeObjective outcome = %s (reasons: %v), want APPROVAL_REQUIRED", propose.Outcome, propose.Reasons)
	}

	resolve := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *propose.ApprovalID, Approve: false, DecidingPrincipal: app.Fixtures.ApproverPrincipal()})
	if resolve.Outcome != Rejected {
		t.Fatalf("ResolveApproval(reject) outcome = %s, want REJECTED", resolve.Outcome)
	}

	orgID := app.Fixtures.Organization().OrganizationID
	_, err := app.Objective.GetObjectiveBySource(ctx, orgID, objdomain.SourceResourceEvaluation, sourceID)
	if !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("GetObjectiveBySource after reject: err = %v, want ErrNotFound (no Objective should exist)", err)
	}
}

// TestIntegration_ProposeObjective_DuplicateSourceRejected proves
// departments.md's "duplicate feedback... cannot create another...
// Objective": once a source (a FINDING here) already has a proposed
// Objective, proposing from it again is REJECTED before a second Approval
// is even created.
func TestIntegration_ProposeObjective_DuplicateSourceRejected(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	sourceID := makeFindingSource(t, app, principalID)

	first := app.ProposeObjective(ctx, ProposeObjectiveRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: objdomain.SourceFinding, SourceID: sourceID,
		Title: "Act on finding", Intent: "First proposal",
	})
	if first.Outcome != ApprovalRequired || first.ApprovalID == nil {
		t.Fatalf("first ProposeObjective outcome = %s (reasons: %v)", first.Outcome, first.Reasons)
	}
	resolve := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *first.ApprovalID, Approve: true, DecidingPrincipal: app.Fixtures.ApproverPrincipal()})
	if resolve.Outcome != Accepted {
		t.Fatalf("ResolveApproval(approve) outcome = %s (reasons: %v)", resolve.Outcome, resolve.Reasons)
	}

	second := app.ProposeObjective(ctx, ProposeObjectiveRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: objdomain.SourceFinding, SourceID: sourceID,
		Title: "Act on finding again", Intent: "Duplicate proposal",
	})
	if second.Outcome != Rejected {
		t.Fatalf("second ProposeObjective outcome = %s (reasons: %v), want REJECTED", second.Outcome, second.Reasons)
	}
}

// TestIntegration_ProposeObjective_RecommendationSourceAndUnknownSourceRejected
// exercises the fourth SourceType (RECOMMENDATION) and proves an
// unrecognized SourceID for a known SourceType fails closed rather than
// silently proceeding.
func TestIntegration_ProposeObjective_RecommendationSourceAndUnknownSourceRejected(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	sourceID := makeRecommendationSource(t, app, principalID)
	propose := app.ProposeObjective(ctx, ProposeObjectiveRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: objdomain.SourceRecommendation, SourceID: sourceID,
		Title: "Adopt recommendation", Intent: "Act on the issued recommendation",
	})
	if propose.Outcome != ApprovalRequired || propose.ApprovalID == nil {
		t.Fatalf("ProposeObjective(RECOMMENDATION) outcome = %s (reasons: %v)", propose.Outcome, propose.Reasons)
	}

	unknown := app.ProposeObjective(ctx, ProposeObjectiveRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: objdomain.SourceRecommendation, SourceID: uuid.New(),
		Title: "Adopt a nonexistent recommendation", Intent: "Should fail closed",
	})
	if unknown.Outcome != Rejected {
		t.Fatalf("ProposeObjective with an unknown RecommendationID outcome = %s, want REJECTED", unknown.Outcome)
	}
}
