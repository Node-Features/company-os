package research

import (
	"testing"

	domain "github.com/Node-Features/company-os/apps/companyd/internal/domain/research"
	"github.com/google/uuid"
)

func contains(reasons []string, want string) bool {
	for _, r := range reasons {
		if r == want {
			return true
		}
	}
	return false
}

func TestValidateSubmitSignal(t *testing.T) {
	if reasons := ValidateSubmitSignal(domain.SourceTypeProviderModelChange, "GPT-6 released"); len(reasons) != 0 {
		t.Fatalf("valid signal rejected: %v", reasons)
	}
	if reasons := ValidateSubmitSignal("UNKNOWN_TYPE", "GPT-6 released"); !contains(reasons, ReasonUnknownSourceType) {
		t.Fatalf("unknown source type not rejected: %v", reasons)
	}
	if reasons := ValidateSubmitSignal(domain.SourceTypeProviderModelChange, "  "); !contains(reasons, ReasonMissingDescription) {
		t.Fatalf("blank description not rejected: %v", reasons)
	}
}

func TestValidateOpenQuestion(t *testing.T) {
	sig := domain.Signal{SignalID: uuid.New()}
	if reasons := ValidateOpenQuestion(sig, "Is this model cheaper?"); len(reasons) != 0 {
		t.Fatalf("valid question rejected: %v", reasons)
	}
	if reasons := ValidateOpenQuestion(sig, ""); !contains(reasons, ReasonMissingText) {
		t.Fatalf("blank text not rejected: %v", reasons)
	}
}

func TestValidateRecordEvidence(t *testing.T) {
	open := domain.ResearchQuestion{Status: domain.QuestionOpen}
	closed := domain.ResearchQuestion{Status: domain.QuestionClosed}

	if reasons := ValidateRecordEvidence(open, "https://example.com", "pricing page"); len(reasons) != 0 {
		t.Fatalf("valid evidence rejected: %v", reasons)
	}
	if reasons := ValidateRecordEvidence(closed, "https://example.com", "pricing page"); !contains(reasons, ReasonQuestionNotOpen) {
		t.Fatalf("evidence against a closed question not rejected: %v", reasons)
	}
	if reasons := ValidateRecordEvidence(open, "", ""); !contains(reasons, ReasonMissingSource) || !contains(reasons, ReasonMissingContent) {
		t.Fatalf("blank source/content not rejected: %v", reasons)
	}
}

func TestValidatePublishFinding(t *testing.T) {
	open := domain.ResearchQuestion{Status: domain.QuestionOpen}
	closed := domain.ResearchQuestion{Status: domain.QuestionClosed}
	evidenceIDs := []uuid.UUID{uuid.New()}

	if reasons := ValidatePublishFinding(open, "It's 30% cheaper", evidenceIDs); len(reasons) != 0 {
		t.Fatalf("valid finding rejected: %v", reasons)
	}
	if reasons := ValidatePublishFinding(open, "It's 30% cheaper", nil); !contains(reasons, ReasonNoEvidenceCited) {
		t.Fatalf("finding with no cited evidence not rejected: %v", reasons)
	}
	if reasons := ValidatePublishFinding(closed, "It's 30% cheaper", evidenceIDs); !contains(reasons, ReasonQuestionNotOpen) {
		t.Fatalf("finding against a closed question not rejected: %v", reasons)
	}
	if reasons := ValidatePublishFinding(open, "  ", evidenceIDs); !contains(reasons, ReasonMissingClaim) {
		t.Fatalf("blank claim not rejected: %v", reasons)
	}
}

func TestValidateIssueRecommendation(t *testing.T) {
	published := domain.Finding{Status: domain.FindingPublished}
	superseded := domain.Finding{Status: domain.FindingSuperseded}

	if reasons := ValidateIssueRecommendation(published, "Switch to the cheaper model"); len(reasons) != 0 {
		t.Fatalf("valid recommendation rejected: %v", reasons)
	}
	if reasons := ValidateIssueRecommendation(superseded, "Switch to the cheaper model"); !contains(reasons, ReasonFindingSuperseded) {
		t.Fatalf("recommendation from a superseded finding not rejected: %v", reasons)
	}
	if reasons := ValidateIssueRecommendation(published, ""); !contains(reasons, ReasonMissingAction) {
		t.Fatalf("blank action not rejected: %v", reasons)
	}
}
