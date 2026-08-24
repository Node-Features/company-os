package application

import (
	"context"
	"errors"
	"testing"
	"time"

	knowledgedomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"
	researchdomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/research"
	"github.com/google/uuid"
)

// publishFindingWithClaim drives a real Signal -> ResearchQuestion ->
// Finding chain (same shape as research_integration_test.go) and returns
// the published Finding's ID plus the KnowledgeItemID that PublishFinding's
// own automatic capture (ROADMAP.md Phase 5 Slice 4) produced for it, so
// Knowledge's tests have both a real source and its already-captured DRAFT
// candidate to work with.
func publishFindingWithClaim(t *testing.T, app *Application, ctx context.Context, principalID uuid.UUID, claim string) (findingID, knowledgeItemID uuid.UUID) {
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
	if finding.KnowledgeItemID == nil {
		t.Fatalf("PublishFinding.KnowledgeItemID = nil (reasons: %v), want the automatically captured item's ID", finding.Reasons)
	}
	return *finding.ResourceID, *finding.KnowledgeItemID
}

// TestIntegration_Knowledge_PublishFinding_AutoCapturesDraftV1 proves
// ROADMAP.md Phase 5 Slice 4's end-to-end wiring against the real database:
// publishing a Finding, with no separate manual capture call, produces a
// DRAFT KnowledgeItem version 1 whose content and digest match the source
// verbatim. Before Slice 4 this required an explicit
// CaptureKnowledgeCandidate call (Slice 1) — PublishFinding now performs it
// automatically as a derived side effect.
func TestIntegration_Knowledge_PublishFinding_AutoCapturesDraftV1(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	claim := "Provider X's model is 30% cheaper at comparable quality, unique per run: " + uuid.New().String()
	_, knowledgeItemID := publishFindingWithClaim(t, app, ctx, principalID, claim)

	item, err := app.GetKnowledgeItem(ctx, knowledgeItemID)
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
// manual re-capture from the same, unedited source is rejected rather than
// creating a pointless identical version. Since PublishFinding (Slice 4)
// already auto-captures v1, the manual CaptureKnowledgeCandidate call this
// test drives is itself the "recapture" under test — there's no separate
// first manual call to make anymore.
func TestIntegration_Knowledge_RecaptureUnchangedContentRejected(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	findingID, _ := publishFindingWithClaim(t, app, ctx, principalID, "Recapture test claim, unique per test run: "+uuid.New().String())

	recapture := app.CaptureKnowledgeCandidate(ctx, CaptureKnowledgeCandidateRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: knowledgedomain.SourceResearchFinding, SourceID: findingID,
	})
	if recapture.Outcome != Rejected {
		t.Fatalf("recapture outcome = %s, want REJECTED", recapture.Outcome)
	}
	if len(recapture.Reasons) != 1 || recapture.Reasons[0] != "content_unchanged_since_last_capture" {
		t.Fatalf("recapture reasons = %v, want [content_unchanged_since_last_capture]", recapture.Reasons)
	}
}

// TestIntegration_Knowledge_DuplicateAcrossSourcesFlagged proves the
// exact-content-digest duplicate signal: two distinct Findings with
// identical claim text produce two distinct, automatically captured
// KnowledgeItems, with the second's DuplicateOfItemID pointing at the
// first — a review signal only, never an automatic merge.
func TestIntegration_Knowledge_DuplicateAcrossSourcesFlagged(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	sharedClaim := "Duplicate-detection test claim, unique per run: " + uuid.New().String()
	_, knowledgeItemID1 := publishFindingWithClaim(t, app, ctx, principalID, sharedClaim)
	_, knowledgeItemID2 := publishFindingWithClaim(t, app, ctx, principalID, sharedClaim)

	item2, err := app.GetKnowledgeItem(ctx, knowledgeItemID2)
	if err != nil {
		t.Fatalf("GetKnowledgeItem: %v", err)
	}
	if item2.DuplicateOfItemID == nil || *item2.DuplicateOfItemID != knowledgeItemID1 {
		t.Fatalf("DuplicateOfItemID = %v, want %v", item2.DuplicateOfItemID, knowledgeItemID1)
	}
}

// TestIntegration_Knowledge_VersionIncrementsOnChangedContent proves
// re-capture with genuinely different content increments Version rather
// than creating a new KnowledgeItemID. A real Finding's claim can never
// actually change once published, so this seeds a fabricated *next* version
// directly through the repository first — the same
// bypass-the-external-boundary-to-prove-the-mechanism pattern already used
// by submitFakeResult and fabricated AuthenticatedEvidence elsewhere in
// this suite. Since Slice 4's auto-capture already claims v1 for real (under
// the source's real KnowledgeItemID), the fabricated row here is v2 under
// that same ID — fabricating a competing v1 under a fresh ID would produce
// two different KnowledgeItemIDs both claiming "latest v1" for one source,
// a state the real system never produces.
func TestIntegration_Knowledge_VersionIncrementsOnChangedContent(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	principalID := uuid.New()

	findingID, knowledgeItemID := publishFindingWithClaim(t, app, ctx, principalID, "Version-increment test claim, unique per run: "+uuid.New().String())

	if err := app.Knowledge.CaptureItem(ctx, &knowledgedomain.KnowledgeItem{
		KnowledgeItemID:       knowledgeItemID,
		OrganizationID:        app.Fixtures.Organization().OrganizationID,
		Version:               2,
		Claim:                 "fabricated stale-version content",
		ContentDigest:         "fabricated-digest-does-not-match-the-real-finding",
		Classification:        knowledgedomain.ClassificationInternal,
		SourceType:            knowledgedomain.SourceResearchFinding,
		SourceID:              findingID,
		ProducedByPrincipalID: principalID,
		ProducedByMethod:      knowledgedomain.MethodSourceVerbatim,
		Status:                knowledgedomain.StatusDraft,
		CreatedAt:             time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed fabricated v2: %v", err)
	}

	res := app.CaptureKnowledgeCandidate(ctx, CaptureKnowledgeCandidateRequest{
		RequestID: uuid.New(), PrincipalID: principalID,
		SourceType: knowledgedomain.SourceResearchFinding, SourceID: findingID,
	})
	if res.Outcome != Accepted || res.ResourceID == nil {
		t.Fatalf("capture outcome = %s (reasons: %v)", res.Outcome, res.Reasons)
	}
	if *res.ResourceID != knowledgeItemID {
		t.Fatalf("ResourceID = %v, want the same KnowledgeItemID as the auto-captured row %v", *res.ResourceID, knowledgeItemID)
	}

	item, err := app.GetKnowledgeItem(ctx, knowledgeItemID)
	if err != nil {
		t.Fatalf("GetKnowledgeItem: %v", err)
	}
	if item.Version != 3 {
		t.Fatalf("Version = %d, want 3", item.Version)
	}
}

// captureKnowledgeItem publishes a real Finding with producerPrincipalID as
// the driving Principal and returns the resulting DRAFT KnowledgeItem that
// PublishFinding's own automatic capture (Slice 4) already produced for
// it — the fixture Slice 2/3 tests build on. No separate manual
// CaptureKnowledgeCandidate call is needed since Slice 4 wired that in.
func captureKnowledgeItem(t *testing.T, app *Application, ctx context.Context, producerPrincipalID uuid.UUID, claim string) knowledgedomain.KnowledgeItem {
	t.Helper()
	_, knowledgeItemID := publishFindingWithClaim(t, app, ctx, producerPrincipalID, claim)
	item, err := app.GetKnowledgeItem(ctx, knowledgeItemID)
	if err != nil {
		t.Fatalf("GetKnowledgeItem: %v", err)
	}
	return item
}

// TestIntegration_Knowledge_RequestApproval_ThenApprove proves the full
// knowledge.approve REQUIRE_APPROVAL round trip: request -> APPROVAL_REQUIRED
// (item still DRAFT) -> human approve -> item APPROVED with every review
// field populated.
func TestIntegration_Knowledge_RequestApproval_ThenApprove(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	producerID := uuid.New()
	requesterID := uuid.New()

	item := captureKnowledgeItem(t, app, ctx, producerID, "Approval-flow test claim, unique per run: "+uuid.New().String())

	reqRes := app.RequestKnowledgeApproval(ctx, RequestKnowledgeApprovalRequest{
		RequestID: uuid.New(), PrincipalID: requesterID,
		KnowledgeItemID: item.KnowledgeItemID, Version: item.Version, ContentDigest: item.ContentDigest,
	})
	if reqRes.Outcome != ApprovalRequired || reqRes.ApprovalID == nil {
		t.Fatalf("RequestKnowledgeApproval outcome = %s (reasons: %v)", reqRes.Outcome, reqRes.Reasons)
	}

	stillDraft, err := app.GetKnowledgeItem(ctx, item.KnowledgeItemID)
	if err != nil {
		t.Fatalf("GetKnowledgeItem: %v", err)
	}
	if stillDraft.Status != knowledgedomain.StatusDraft {
		t.Fatalf("Status = %s, want DRAFT while pending", stillDraft.Status)
	}

	decideRes := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *reqRes.ApprovalID, Approve: true})
	if decideRes.Outcome != Accepted || decideRes.ResourceID == nil {
		t.Fatalf("ResolveApproval(approve) outcome = %s (reasons: %v)", decideRes.Outcome, decideRes.Reasons)
	}

	approved, err := app.GetKnowledgeItem(ctx, item.KnowledgeItemID)
	if err != nil {
		t.Fatalf("GetKnowledgeItem: %v", err)
	}
	if approved.Status != knowledgedomain.StatusApproved {
		t.Fatalf("Status = %s, want APPROVED", approved.Status)
	}
	if approved.ReviewerPrincipalID == nil || approved.GovernanceDecisionID == nil || approved.ApprovalID == nil || approved.ReviewedAt == nil {
		t.Fatalf("review fields not populated: %+v", approved)
	}
}

// TestIntegration_Knowledge_RequestApproval_ThenReject proves a human
// reject transitions the item to REJECTED (docs/architecture/knowledge.md's
// "separate rejected-review transition"), not left DRAFT, and that a
// second resolution of the same Approval is CONFLICT.
func TestIntegration_Knowledge_RequestApproval_ThenReject(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	producerID := uuid.New()
	requesterID := uuid.New()

	item := captureKnowledgeItem(t, app, ctx, producerID, "Rejection-flow test claim, unique per run: "+uuid.New().String())

	reqRes := app.RequestKnowledgeApproval(ctx, RequestKnowledgeApprovalRequest{
		RequestID: uuid.New(), PrincipalID: requesterID,
		KnowledgeItemID: item.KnowledgeItemID, Version: item.Version, ContentDigest: item.ContentDigest,
	})
	if reqRes.Outcome != ApprovalRequired || reqRes.ApprovalID == nil {
		t.Fatalf("RequestKnowledgeApproval outcome = %s (reasons: %v)", reqRes.Outcome, reqRes.Reasons)
	}

	decideRes := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *reqRes.ApprovalID, Approve: false})
	if decideRes.Outcome != Rejected {
		t.Fatalf("ResolveApproval(reject) outcome = %s, want REJECTED", decideRes.Outcome)
	}

	rejected, err := app.GetKnowledgeItem(ctx, item.KnowledgeItemID)
	if err != nil {
		t.Fatalf("GetKnowledgeItem: %v", err)
	}
	if rejected.Status != knowledgedomain.StatusRejected {
		t.Fatalf("Status = %s, want REJECTED", rejected.Status)
	}

	again := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *reqRes.ApprovalID, Approve: true})
	if again.Outcome != Conflict {
		t.Fatalf("second resolution outcome = %s, want CONFLICT", again.Outcome)
	}
}

// TestIntegration_Knowledge_RequestApproval_SelfReviewDenied proves
// separation-of-duties as a real Governance DENY (governance.Request.
// ExcludedPrincipalID): the review requester cannot be the candidate's own
// producer, and a DENY leaves the candidate unchanged
// (docs/architecture/knowledge.md).
func TestIntegration_Knowledge_RequestApproval_SelfReviewDenied(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	producerID := uuid.New()

	item := captureKnowledgeItem(t, app, ctx, producerID, "Self-review test claim, unique per run: "+uuid.New().String())

	reqRes := app.RequestKnowledgeApproval(ctx, RequestKnowledgeApprovalRequest{
		RequestID: uuid.New(), PrincipalID: producerID, // same Principal as the producer
		KnowledgeItemID: item.KnowledgeItemID, Version: item.Version, ContentDigest: item.ContentDigest,
	})
	if reqRes.Outcome != Denied {
		t.Fatalf("RequestKnowledgeApproval(self-review) outcome = %s, want DENIED", reqRes.Outcome)
	}

	unchanged, err := app.GetKnowledgeItem(ctx, item.KnowledgeItemID)
	if err != nil {
		t.Fatalf("GetKnowledgeItem: %v", err)
	}
	if unchanged.Status != knowledgedomain.StatusDraft {
		t.Fatalf("Status = %s, want DRAFT (unchanged)", unchanged.Status)
	}
}

// TestIntegration_Knowledge_RequestApproval_StaleVersionRejected proves a
// version that changed between request and resolution is rejected before
// Governance is even reached (docs/architecture/knowledge.md: "a stale item
// version... requires a new Governance evaluation"), not silently approved
// against stale content. Seeds the newer version directly through the
// repository, bypassing Application — the same pattern Slice 1's
// version-increment test uses, since nothing in Application itself can
// create a second version for an item mid-review (this item's source
// Finding can never change).
func TestIntegration_Knowledge_RequestApproval_StaleVersionRejected(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	producerID := uuid.New()
	requesterID := uuid.New()

	item := captureKnowledgeItem(t, app, ctx, producerID, "Stale-version test claim, unique per run: "+uuid.New().String())

	reqRes := app.RequestKnowledgeApproval(ctx, RequestKnowledgeApprovalRequest{
		RequestID: uuid.New(), PrincipalID: requesterID,
		KnowledgeItemID: item.KnowledgeItemID, Version: item.Version, ContentDigest: item.ContentDigest,
	})
	if reqRes.Outcome != ApprovalRequired || reqRes.ApprovalID == nil {
		t.Fatalf("RequestKnowledgeApproval outcome = %s (reasons: %v)", reqRes.Outcome, reqRes.Reasons)
	}

	if err := app.Knowledge.CaptureItem(ctx, &knowledgedomain.KnowledgeItem{
		KnowledgeItemID:       item.KnowledgeItemID,
		OrganizationID:        item.OrganizationID,
		Version:               item.Version + 1,
		Claim:                 "changed content after the review request was made",
		ContentDigest:         "a-different-digest-simulating-changed-content",
		Classification:        knowledgedomain.ClassificationInternal,
		SourceType:            item.SourceType,
		SourceID:              item.SourceID,
		ProducedByPrincipalID: producerID,
		ProducedByMethod:      knowledgedomain.MethodSourceVerbatim,
		Status:                knowledgedomain.StatusDraft,
		CreatedAt:             time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed newer version: %v", err)
	}

	decideRes := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *reqRes.ApprovalID, Approve: true})
	if decideRes.Outcome != Rejected {
		t.Fatalf("ResolveApproval(approve) outcome = %s, want REJECTED (stale)", decideRes.Outcome)
	}
	if len(decideRes.Reasons) != 1 || decideRes.Reasons[0] != "knowledge_item_stale_version_or_digest" {
		t.Fatalf("reasons = %v, want [knowledge_item_stale_version_or_digest]", decideRes.Reasons)
	}

	latest, err := app.GetKnowledgeItem(ctx, item.KnowledgeItemID)
	if err != nil {
		t.Fatalf("GetKnowledgeItem: %v", err)
	}
	if latest.Status != knowledgedomain.StatusDraft {
		t.Fatalf("latest version Status = %s, want DRAFT (never transitioned)", latest.Status)
	}
}

// TestIntegration_Knowledge_QueryDefaultReturnsOnlyApproved_LatestApprovedVersionPerItem
// proves the key retrieval-contract scenario: an item with an APPROVED
// version and a newer, not-yet-reviewed DRAFT version must still surface at
// its latest APPROVED version under the default (APPROVED-only) query — not
// be skipped, and not return the DRAFT one.
func TestIntegration_Knowledge_QueryDefaultReturnsOnlyApproved_LatestApprovedVersionPerItem(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	producerID := uuid.New()
	requesterID := uuid.New()

	item := captureKnowledgeItem(t, app, ctx, producerID, "Query-default test claim, unique per run: "+uuid.New().String())

	reqRes := app.RequestKnowledgeApproval(ctx, RequestKnowledgeApprovalRequest{
		RequestID: uuid.New(), PrincipalID: requesterID,
		KnowledgeItemID: item.KnowledgeItemID, Version: item.Version, ContentDigest: item.ContentDigest,
	})
	if reqRes.Outcome != ApprovalRequired || reqRes.ApprovalID == nil {
		t.Fatalf("RequestKnowledgeApproval outcome = %s (reasons: %v)", reqRes.Outcome, reqRes.Reasons)
	}
	decideRes := app.ResolveApproval(ctx, ResolveApprovalRequest{ApprovalID: *reqRes.ApprovalID, Approve: true})
	if decideRes.Outcome != Accepted {
		t.Fatalf("ResolveApproval(approve) outcome = %s (reasons: %v)", decideRes.Outcome, decideRes.Reasons)
	}

	// Simulate a newer, not-yet-reviewed version alongside the approved one
	// — bypasses Application, same pattern used throughout this file.
	if err := app.Knowledge.CaptureItem(ctx, &knowledgedomain.KnowledgeItem{
		KnowledgeItemID:       item.KnowledgeItemID,
		OrganizationID:        item.OrganizationID,
		Version:               item.Version + 1,
		Claim:                 "a newer, not-yet-reviewed revision",
		ContentDigest:         "a-different-digest-for-the-newer-revision",
		Classification:        knowledgedomain.ClassificationInternal,
		SourceType:            item.SourceType,
		SourceID:              item.SourceID,
		ProducedByPrincipalID: producerID,
		ProducedByMethod:      knowledgedomain.MethodSourceVerbatim,
		Status:                knowledgedomain.StatusDraft,
		CreatedAt:             time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed newer draft version: %v", err)
	}

	results, err := app.QueryKnowledge(ctx, QueryKnowledgeRequest{})
	if err != nil {
		t.Fatalf("QueryKnowledge: %v", err)
	}
	var found *knowledgedomain.KnowledgeItem
	for i := range results {
		if results[i].KnowledgeItemID == item.KnowledgeItemID {
			found = &results[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("item %v not found in default query results", item.KnowledgeItemID)
	}
	if found.Version != item.Version || found.Status != knowledgedomain.StatusApproved {
		t.Fatalf("found version/status = %d/%s, want %d/APPROVED (the approved version, not the newer draft)", found.Version, found.Status, item.Version)
	}
}

// TestIntegration_Knowledge_QueryDraftInclusiveWithoutPurposeRejected proves
// a draft-inclusive query without a purpose is rejected, not silently
// widened.
func TestIntegration_Knowledge_QueryDraftInclusiveWithoutPurposeRejected(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()

	_, err := app.QueryKnowledge(ctx, QueryKnowledgeRequest{Statuses: []knowledgedomain.Status{knowledgedomain.StatusDraft}})
	if !errors.Is(err, ErrKnowledgeQueryPurposeRequired) {
		t.Fatalf("err = %v, want ErrKnowledgeQueryPurposeRequired", err)
	}
}

// TestIntegration_Knowledge_QueryDraftInclusiveWithPurposeReturnsDraft
// proves a properly-labeled draft-inclusive query (a non-empty purpose)
// succeeds and returns the DRAFT item with its real Status visible — the
// "label."
func TestIntegration_Knowledge_QueryDraftInclusiveWithPurposeReturnsDraft(t *testing.T) {
	app := requireRealApp(t)
	ctx := context.Background()
	producerID := uuid.New()

	item := captureKnowledgeItem(t, app, ctx, producerID, "Query-draft-inclusive test claim, unique per run: "+uuid.New().String())

	purpose := "editorial review dashboard"
	results, err := app.QueryKnowledge(ctx, QueryKnowledgeRequest{
		Statuses: []knowledgedomain.Status{knowledgedomain.StatusDraft},
		Purpose:  &purpose,
	})
	if err != nil {
		t.Fatalf("QueryKnowledge: %v", err)
	}
	var found *knowledgedomain.KnowledgeItem
	for i := range results {
		if results[i].KnowledgeItemID == item.KnowledgeItemID {
			found = &results[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("item %v not found in draft-inclusive query results", item.KnowledgeItemID)
	}
	if found.Status != knowledgedomain.StatusDraft {
		t.Fatalf("Status = %s, want DRAFT", found.Status)
	}
}
