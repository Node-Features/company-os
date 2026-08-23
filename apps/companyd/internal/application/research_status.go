package application

import (
	"context"

	"github.com/google/uuid"
)

// ResearchQuestionStatusView is the read projection served by
// GET /v1/research/questions/{questionId} — mirrors WorkflowView's role for
// the Workflow status endpoint.
type ResearchQuestionStatusView struct {
	QuestionID string
	SignalID   string
	Text       string
	Status     string
	Evidence   []EvidenceView
	Findings   []FindingView
}

type EvidenceView struct {
	EvidenceID string
	Source     string
	Content    string
}

type FindingView struct {
	FindingID       string
	Claim           string
	Status          string
	EvidenceIDs     []string
	Recommendations []RecommendationView
}

type RecommendationView struct {
	RecommendationID string
	ProposedAction   string
	Status           string
}

// GetResearchQuestionStatus assembles the full read projection for one
// ResearchQuestion: its Evidence, and every Finding published against it
// with that Finding's own Recommendations.
func (a *Application) GetResearchQuestionStatus(ctx context.Context, questionID uuid.UUID) (ResearchQuestionStatusView, error) {
	orgID := a.Fixtures.Organization().OrganizationID

	q, err := a.Research.GetResearchQuestion(ctx, orgID, questionID)
	if err != nil {
		return ResearchQuestionStatusView{}, err
	}

	evidence, err := a.Research.ListEvidence(ctx, orgID, questionID)
	if err != nil {
		return ResearchQuestionStatusView{}, err
	}
	evidenceViews := make([]EvidenceView, len(evidence))
	for i, e := range evidence {
		evidenceViews[i] = EvidenceView{EvidenceID: e.EvidenceID.String(), Source: e.Source, Content: e.Content}
	}

	findings, err := a.Research.ListFindingsByQuestion(ctx, orgID, questionID)
	if err != nil {
		return ResearchQuestionStatusView{}, err
	}
	findingViews := make([]FindingView, len(findings))
	for i, f := range findings {
		recs, err := a.Research.ListRecommendationsByFinding(ctx, orgID, f.FindingID)
		if err != nil {
			return ResearchQuestionStatusView{}, err
		}
		recViews := make([]RecommendationView, len(recs))
		for j, r := range recs {
			recViews[j] = RecommendationView{RecommendationID: r.RecommendationID.String(), ProposedAction: r.ProposedAction, Status: string(r.Status)}
		}
		evidenceIDs := make([]string, len(f.EvidenceIDs))
		for j, id := range f.EvidenceIDs {
			evidenceIDs[j] = id.String()
		}
		findingViews[i] = FindingView{
			FindingID:       f.FindingID.String(),
			Claim:           f.Claim,
			Status:          string(f.Status),
			EvidenceIDs:     evidenceIDs,
			Recommendations: recViews,
		}
	}

	return ResearchQuestionStatusView{
		QuestionID: q.QuestionID.String(),
		SignalID:   q.SignalID.String(),
		Text:       q.Text,
		Status:     string(q.Status),
		Evidence:   evidenceViews,
		Findings:   findingViews,
	}, nil
}
