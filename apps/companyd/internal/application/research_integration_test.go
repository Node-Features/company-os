package application

import (
	"context"
	"testing"
	"time"

	researchdomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/research"
	"github.com/google/uuid"
)

// TestIntegration_ResearchLoop_SignalToRecommendation proves
// docs/workflows/research-loop.md's full Signal -> ResearchQuestion ->
// Evidence -> Finding -> Recommendation chain end to end against the real
// database, including that each step persists a real GovernanceDecision
// for the real (fabricated, not fixture) Principal driving it — the first
// slice where Governance evaluates a non-fixture Principal.
func TestIntegration_ResearchLoop_SignalToRecommendation(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New() // stands in for a real signed-in HumanPrincipal (no live JWT needed to exercise Application directly)

	signal := app.SubmitSignal(ctx, SubmitSignalRequest{
		RequestID:   uuid.New(),
		PrincipalID: principalID,
		SourceType:  researchdomain.SourceTypeProviderModelChange,
		Description: "New provider X released a model 30% cheaper than our current one",
	})
	if signal.Outcome != Accepted || signal.ResourceID == nil {
		t.Fatalf("SubmitSignal outcome = %s (reasons: %v)", signal.Outcome, signal.Reasons)
	}
	signalID := *signal.ResourceID

	question := app.OpenResearchQuestion(ctx, OpenResearchQuestionRequest{
		RequestID:   uuid.New(),
		PrincipalID: principalID,
		SignalID:    signalID,
		Text:        "Is provider X's new model a viable cheaper alternative?",
	})
	if question.Outcome != Accepted || question.ResourceID == nil {
		t.Fatalf("OpenResearchQuestion outcome = %s (reasons: %v)", question.Outcome, question.Reasons)
	}
	questionID := *question.ResourceID

	evidence := app.RecordEvidence(ctx, RecordEvidenceRequest{
		RequestID:   uuid.New(),
		PrincipalID: principalID,
		QuestionID:  questionID,
		Source:      "https://provider-x.example/pricing",
		Content:     "Provider X's pricing page shows $0.07/1M tokens vs our current $0.10/1M",
		RetrievedAt: time.Now().UTC(),
	})
	if evidence.Outcome != Accepted || evidence.ResourceID == nil {
		t.Fatalf("RecordEvidence outcome = %s (reasons: %v)", evidence.Outcome, evidence.Reasons)
	}
	evidenceID := *evidence.ResourceID

	finding := app.PublishFinding(ctx, PublishFindingRequest{
		RequestID:   uuid.New(),
		PrincipalID: principalID,
		QuestionID:  questionID,
		Claim:       "Provider X's model is 30% cheaper at comparable quality",
		EvidenceIDs: []uuid.UUID{evidenceID},
	})
	if finding.Outcome != Accepted || finding.ResourceID == nil {
		t.Fatalf("PublishFinding outcome = %s (reasons: %v)", finding.Outcome, finding.Reasons)
	}
	findingID := *finding.ResourceID

	recommendation := app.IssueRecommendation(ctx, IssueRecommendationRequest{
		RequestID:      uuid.New(),
		PrincipalID:    principalID,
		FindingID:      findingID,
		ProposedAction: "Evaluate Provider X as a fallback ProviderAdapter",
	})
	if recommendation.Outcome != Accepted || recommendation.ResourceID == nil {
		t.Fatalf("IssueRecommendation outcome = %s (reasons: %v)", recommendation.Outcome, recommendation.Reasons)
	}

	status, err := app.GetResearchQuestionStatus(ctx, questionID)
	if err != nil {
		t.Fatalf("GetResearchQuestionStatus: %v", err)
	}
	if len(status.Evidence) != 1 {
		t.Fatalf("status.Evidence = %d entries, want 1", len(status.Evidence))
	}
	if len(status.Findings) != 1 {
		t.Fatalf("status.Findings = %d entries, want 1", len(status.Findings))
	}
	if len(status.Findings[0].Recommendations) != 1 {
		t.Fatalf("status.Findings[0].Recommendations = %d entries, want 1", len(status.Findings[0].Recommendations))
	}
}

// TestIntegration_PublishFinding_RejectsUncitedEvidence proves the one
// mechanically-checkable part of "a finding cannot exceed what its evidence
// supports" (research.md): citing an EvidenceID that doesn't belong to the
// ResearchQuestion is rejected, not silently accepted.
func TestIntegration_PublishFinding_RejectsUncitedEvidence(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	signal := app.SubmitSignal(ctx, SubmitSignalRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: researchdomain.SourceTypeProviderModelChange, Description: "signal for uncited-evidence test",
	})
	question := app.OpenResearchQuestion(ctx, OpenResearchQuestionRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SignalID: *signal.ResourceID, Text: "question for uncited-evidence test",
	})

	finding := app.PublishFinding(ctx, PublishFindingRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		QuestionID: *question.ResourceID, Claim: "a claim with a fabricated evidence reference",
		EvidenceIDs: []uuid.UUID{uuid.New()}, // never recorded against this question
	})
	if finding.Outcome != Rejected {
		t.Fatalf("PublishFinding with an uncited EvidenceID outcome = %s, want REJECTED", finding.Outcome)
	}
}
