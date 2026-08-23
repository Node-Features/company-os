package application

import (
	"context"
	"testing"
	"time"

	knowledgedomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"
	researchdomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/research"
	"github.com/google/uuid"
)

// publishFindingWithClaim drives a real Signal -> ResearchQuestion ->
// Finding chain (same shape as research_integration_test.go) and returns
// the published Finding's ID, so Knowledge's ingestion tests have a real
// source to capture from.
func publishFindingWithClaim(t *testing.T, app *Application, ctx context.Context, principalID uuid.UUID, claim string) uuid.UUID {
	t.Helper()
	signal := app.SubmitSignal(ctx, SubmitSignalRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: researchdomain.SourceTypeProviderModelChange, Description: "signal for knowledge ingestion test",
	})
	if signal.Outcome != Accepted || signal.ResourceID == nil {
		t.Fatalf("SubmitSignal outcome = %s (reasons: %v)", signal.Outcome, signal.Reasons)
	}
	question := app.OpenResearchQuestion(ctx, OpenResearchQuestionRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SignalID: *signal.ResourceID, Text: "question for knowledge ingestion test",
	})
	if question.Outcome != Accepted || question.ResourceID == nil {
		t.Fatalf("OpenResearchQuestion outcome = %s (reasons: %v)", question.Outcome, question.Reasons)
	}
	evidence := app.RecordEvidence(ctx, RecordEvidenceRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		QuestionID:  *question.ResourceID,
		Source:      "https://example.test/knowledge-ingestion-test",
		Content:     "evidence content for knowledge ingestion test",
		RetrievedAt: time.Now().UTC(),
	})
	if evidence.Outcome != Accepted || evidence.ResourceID == nil {
		t.Fatalf("RecordEvidence outcome = %s (reasons: %v)", evidence.Outcome, evidence.Reasons)
	}

	finding := app.PublishFinding(ctx, PublishFindingRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		QuestionID: *question.ResourceID, Claim: claim, EvidenceIDs: []uuid.UUID{*evidence.ResourceID},
	})
	if finding.Outcome != Accepted || finding.ResourceID == nil {
		t.Fatalf("PublishFinding outcome = %s (reasons: %v)", finding.Outcome, finding.Reasons)
	}
	return *finding.ResourceID
}

// TestIntegration_Knowledge_CaptureFromFinding_CreatesDraftV1 proves Phase 5
// Slice 1's ingestion use case against the real database: capturing a real
// published Finding produces a DRAFT KnowledgeItem version 1 whose content
// and digest match the source verbatim.
func TestIntegration_Knowledge_CaptureFromFinding_CreatesDraftV1(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	claim := "Provider X's model is 30% cheaper at comparable quality, unique per run: " + uuid.New().String()
	findingID := publishFindingWithClaim(t, app, ctx, principalID, claim)

	res := app.CaptureKnowledgeCandidate(ctx, CaptureKnowledgeCandidateRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: knowledgedomain.SourceResearchFinding, SourceID: findingID,
	})
	if res.Outcome != Accepted || res.ResourceID == nil {
		t.Fatalf("CaptureKnowledgeCandidate outcome = %s (reasons: %v)", res.Outcome, res.Reasons)
	}

	item, err := app.GetKnowledgeItem(ctx, *res.ResourceID)
	if err != nil {
		t.Fatalf("GetKnowledgeItem: %v", err)
	}
	if item.Version != 1 {
		t.Fatalf("Version = %d, want 1", item.Version)
	}
	if item.Status != knowledgedomain.StatusDraft {
		t.Fatalf("Status = %s, want DRAFT", item.Status)
	}
	if item.Claim != claim {
		t.Fatalf("Claim mismatch: %q", item.Claim)
	}
	if item.DuplicateOfItemID != nil {
		t.Fatalf("DuplicateOfItemID = %v, want nil for a first-of-its-kind claim", *item.DuplicateOfItemID)
	}
}

// TestIntegration_Knowledge_RecaptureUnchangedContentRejected proves a
// second capture from the same, unedited source is rejected rather than
// creating a pointless identical version.
func TestIntegration_Knowledge_RecaptureUnchangedContentRejected(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	findingID := publishFindingWithClaim(t, app, ctx, principalID, "Recapture test claim, unique per test run: "+uuid.New().String())

	first := app.CaptureKnowledgeCandidate(ctx, CaptureKnowledgeCandidateRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: knowledgedomain.SourceResearchFinding, SourceID: findingID,
	})
	if first.Outcome != Accepted {
		t.Fatalf("first capture outcome = %s (reasons: %v)", first.Outcome, first.Reasons)
	}

	second := app.CaptureKnowledgeCandidate(ctx, CaptureKnowledgeCandidateRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: knowledgedomain.SourceResearchFinding, SourceID: findingID,
	})
	if second.Outcome != Rejected {
		t.Fatalf("second capture outcome = %s, want REJECTED", second.Outcome)
	}
	if len(second.Reasons) != 1 || second.Reasons[0] != "content_unchanged_since_last_capture" {
		t.Fatalf("second capture reasons = %v, want [content_unchanged_since_last_capture]", second.Reasons)
	}
}

// TestIntegration_Knowledge_DuplicateAcrossSourcesFlagged proves the
// exact-content-digest duplicate signal: two distinct Findings with
// identical claim text produce two distinct KnowledgeItems, with the
// second's DuplicateOfItemID pointing at the first — a review signal only,
// never an automatic merge.
func TestIntegration_Knowledge_DuplicateAcrossSourcesFlagged(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	sharedClaim := "Duplicate-detection test claim, unique per run: " + uuid.New().String()
	finding1 := publishFindingWithClaim(t, app, ctx, principalID, sharedClaim)
	finding2 := publishFindingWithClaim(t, app, ctx, principalID, sharedClaim)

	res1 := app.CaptureKnowledgeCandidate(ctx, CaptureKnowledgeCandidateRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: knowledgedomain.SourceResearchFinding, SourceID: finding1,
	})
	if res1.Outcome != Accepted || res1.ResourceID == nil {
		t.Fatalf("capture 1 outcome = %s (reasons: %v)", res1.Outcome, res1.Reasons)
	}

	res2 := app.CaptureKnowledgeCandidate(ctx, CaptureKnowledgeCandidateRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: knowledgedomain.SourceResearchFinding, SourceID: finding2,
	})
	if res2.Outcome != Accepted || res2.ResourceID == nil {
		t.Fatalf("capture 2 outcome = %s (reasons: %v)", res2.Outcome, res2.Reasons)
	}

	item2, err := app.GetKnowledgeItem(ctx, *res2.ResourceID)
	if err != nil {
		t.Fatalf("GetKnowledgeItem: %v", err)
	}
	if item2.DuplicateOfItemID == nil || *item2.DuplicateOfItemID != *res1.ResourceID {
		t.Fatalf("DuplicateOfItemID = %v, want %v", item2.DuplicateOfItemID, *res1.ResourceID)
	}
}

// TestIntegration_Knowledge_VersionIncrementsOnChangedContent proves
// re-capture with genuinely different content increments Version rather
// than creating a new KnowledgeItemID. A real Finding's claim can never
// actually change once published, so this seeds a fabricated prior version
// directly through the repository first — the same
// bypass-the-external-boundary-to-prove-the-mechanism pattern already used
// by submitFakeResult and fabricated AuthenticatedEvidence elsewhere in
// this suite.
func TestIntegration_Knowledge_VersionIncrementsOnChangedContent(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	findingID := publishFindingWithClaim(t, app, ctx, principalID, "Version-increment test claim, unique per run: "+uuid.New().String())

	fabricatedItemID := uuid.New()
	if err := app.Knowledge.CaptureItem(ctx, &knowledgedomain.KnowledgeItem{
		KnowledgeItemID:       fabricatedItemID,
		OrganizationID:        app.Fixtures.Organization().OrganizationID,
		Version:               1,
		Claim:                 "fabricated prior-version content",
		ContentDigest:         "fabricated-digest-does-not-match-the-real-finding",
		Classification:        knowledgedomain.ClassificationInternal,
		SourceType:            knowledgedomain.SourceResearchFinding,
		SourceID:              findingID,
		ProducedByPrincipalID: principalID,
		ProducedByMethod:      knowledgedomain.MethodSourceVerbatim,
		Status:                knowledgedomain.StatusDraft,
		CreatedAt:              time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed fabricated v1: %v", err)
	}

	res := app.CaptureKnowledgeCandidate(ctx, CaptureKnowledgeCandidateRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: knowledgedomain.SourceResearchFinding, SourceID: findingID,
	})
	if res.Outcome != Accepted || res.ResourceID == nil {
		t.Fatalf("capture outcome = %s (reasons: %v)", res.Outcome, res.Reasons)
	}
	if *res.ResourceID != fabricatedItemID {
		t.Fatalf("ResourceID = %v, want the same KnowledgeItemID as the seeded row %v", *res.ResourceID, fabricatedItemID)
	}

	item, err := app.GetKnowledgeItem(ctx, fabricatedItemID)
	if err != nil {
		t.Fatalf("GetKnowledgeItem: %v", err)
	}
	if item.Version != 2 {
		t.Fatalf("Version = %d, want 2", item.Version)
	}
}
