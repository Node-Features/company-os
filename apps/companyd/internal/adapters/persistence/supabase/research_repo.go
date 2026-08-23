package supabase

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/research"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ResearchRepository implements ports.ResearchRepository.
// See docs/workflows/research-loop.md.
type ResearchRepository struct{ p *Pool }

func NewResearchRepository(p *Pool) *ResearchRepository { return &ResearchRepository{p: p} }

func (r *ResearchRepository) CreateSignal(ctx context.Context, s *research.Signal) error {
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO signals (signal_id, organization_id, source_type, description, submitted_by_principal_id, submitted_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		s.SignalID, s.OrganizationID, s.SourceType, s.Description, s.SubmittedByPrincipalID, s.SubmittedAt)
	return err
}

func (r *ResearchRepository) GetSignal(ctx context.Context, organizationID, signalID uuid.UUID) (research.Signal, error) {
	var s research.Signal
	err := r.p.pool.QueryRow(ctx, `
		SELECT signal_id, organization_id, source_type, description, submitted_by_principal_id, submitted_at
		FROM signals WHERE organization_id=$1 AND signal_id=$2`,
		organizationID, signalID,
	).Scan(&s.SignalID, &s.OrganizationID, &s.SourceType, &s.Description, &s.SubmittedByPrincipalID, &s.SubmittedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return research.Signal{}, ports.ErrNotFound
	}
	return s, err
}

func (r *ResearchRepository) CreateResearchQuestion(ctx context.Context, q *research.ResearchQuestion) error {
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO research_questions (question_id, organization_id, signal_id, text, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		q.QuestionID, q.OrganizationID, q.SignalID, q.Text, q.Status, q.CreatedAt)
	return err
}

func (r *ResearchRepository) GetResearchQuestion(ctx context.Context, organizationID, questionID uuid.UUID) (research.ResearchQuestion, error) {
	var q research.ResearchQuestion
	var status string
	err := r.p.pool.QueryRow(ctx, `
		SELECT question_id, organization_id, signal_id, text, status, created_at
		FROM research_questions WHERE organization_id=$1 AND question_id=$2`,
		organizationID, questionID,
	).Scan(&q.QuestionID, &q.OrganizationID, &q.SignalID, &q.Text, &status, &q.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return research.ResearchQuestion{}, ports.ErrNotFound
	}
	if err != nil {
		return research.ResearchQuestion{}, err
	}
	q.Status = research.QuestionStatus(status)
	return q, nil
}

func (r *ResearchRepository) RecordEvidence(ctx context.Context, e *research.Evidence) error {
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO evidence (evidence_id, organization_id, question_id, source, content, retrieved_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.EvidenceID, e.OrganizationID, e.QuestionID, e.Source, e.Content, e.RetrievedAt, e.CreatedAt)
	return err
}

func (r *ResearchRepository) ListEvidence(ctx context.Context, organizationID, questionID uuid.UUID) ([]research.Evidence, error) {
	rows, err := r.p.pool.Query(ctx, `
		SELECT evidence_id, organization_id, question_id, source, content, retrieved_at, created_at
		FROM evidence WHERE organization_id=$1 AND question_id=$2 ORDER BY created_at`,
		organizationID, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []research.Evidence
	for rows.Next() {
		var e research.Evidence
		if err := rows.Scan(&e.EvidenceID, &e.OrganizationID, &e.QuestionID, &e.Source, &e.Content, &e.RetrievedAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

func (r *ResearchRepository) PublishFinding(ctx context.Context, f *research.Finding) error {
	evidenceIDs, err := json.Marshal(f.EvidenceIDs)
	if err != nil {
		return err
	}
	_, err = r.p.pool.Exec(ctx, `
		INSERT INTO findings (finding_id, organization_id, question_id, claim, evidence_ids, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		f.FindingID, f.OrganizationID, f.QuestionID, f.Claim, evidenceIDs, f.Status, f.CreatedAt)
	return err
}

func (r *ResearchRepository) GetFinding(ctx context.Context, organizationID, findingID uuid.UUID) (research.Finding, error) {
	var f research.Finding
	var status string
	var evidenceIDs []byte
	err := r.p.pool.QueryRow(ctx, `
		SELECT finding_id, organization_id, question_id, claim, evidence_ids, status, created_at
		FROM findings WHERE organization_id=$1 AND finding_id=$2`,
		organizationID, findingID,
	).Scan(&f.FindingID, &f.OrganizationID, &f.QuestionID, &f.Claim, &evidenceIDs, &status, &f.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return research.Finding{}, ports.ErrNotFound
	}
	if err != nil {
		return research.Finding{}, err
	}
	f.Status = research.FindingStatus(status)
	if err := json.Unmarshal(evidenceIDs, &f.EvidenceIDs); err != nil {
		return research.Finding{}, err
	}
	return f, nil
}

func (r *ResearchRepository) ListFindingsByQuestion(ctx context.Context, organizationID, questionID uuid.UUID) ([]research.Finding, error) {
	rows, err := r.p.pool.Query(ctx, `
		SELECT finding_id, organization_id, question_id, claim, evidence_ids, status, created_at
		FROM findings WHERE organization_id=$1 AND question_id=$2 ORDER BY created_at`,
		organizationID, questionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []research.Finding
	for rows.Next() {
		var f research.Finding
		var status string
		var evidenceIDs []byte
		if err := rows.Scan(&f.FindingID, &f.OrganizationID, &f.QuestionID, &f.Claim, &evidenceIDs, &status, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.Status = research.FindingStatus(status)
		if err := json.Unmarshal(evidenceIDs, &f.EvidenceIDs); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

func (r *ResearchRepository) IssueRecommendation(ctx context.Context, rec *research.Recommendation) error {
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO recommendations (recommendation_id, organization_id, finding_id, proposed_action, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		rec.RecommendationID, rec.OrganizationID, rec.FindingID, rec.ProposedAction, rec.Status, rec.CreatedAt)
	return err
}

func (r *ResearchRepository) GetRecommendation(ctx context.Context, organizationID, recommendationID uuid.UUID) (research.Recommendation, error) {
	var rec research.Recommendation
	var status string
	err := r.p.pool.QueryRow(ctx, `
		SELECT recommendation_id, organization_id, finding_id, proposed_action, status, created_at
		FROM recommendations WHERE organization_id=$1 AND recommendation_id=$2`,
		organizationID, recommendationID,
	).Scan(&rec.RecommendationID, &rec.OrganizationID, &rec.FindingID, &rec.ProposedAction, &status, &rec.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return research.Recommendation{}, ports.ErrNotFound
	}
	if err != nil {
		return research.Recommendation{}, err
	}
	rec.Status = research.RecommendationStatus(status)
	return rec, nil
}

func (r *ResearchRepository) ListRecommendationsByFinding(ctx context.Context, organizationID, findingID uuid.UUID) ([]research.Recommendation, error) {
	rows, err := r.p.pool.Query(ctx, `
		SELECT recommendation_id, organization_id, finding_id, proposed_action, status, created_at
		FROM recommendations WHERE organization_id=$1 AND finding_id=$2 ORDER BY created_at`,
		organizationID, findingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []research.Recommendation
	for rows.Next() {
		var rec research.Recommendation
		var status string
		if err := rows.Scan(&rec.RecommendationID, &rec.OrganizationID, &rec.FindingID, &rec.ProposedAction, &status, &rec.CreatedAt); err != nil {
			return nil, err
		}
		rec.Status = research.RecommendationStatus(status)
		list = append(list, rec)
	}
	return list, rows.Err()
}

var _ ports.ResearchRepository = (*ResearchRepository)(nil)
