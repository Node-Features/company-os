package application

import (
	"time"

	"github.com/google/uuid"
)

// Research ApplicationRequests (docs/workflows/research-loop.md). Unlike
// Workflow requests, each carries an explicit PrincipalID — the real
// authenticated, resolved Principal the HTTP handler read from context
// (httpapi.PrincipalFromContext), not a fixtures.Registry stand-in. No
// IdempotencyKey — this slice deliberately has no replay guard (see
// research-loop.md's scope boundary).

type SubmitSignalRequest struct {
	RequestID      uuid.UUID
	PrincipalID    uuid.UUID
	SourceType     string
	Description    string
}

type OpenResearchQuestionRequest struct {
	RequestID   uuid.UUID
	PrincipalID uuid.UUID
	SignalID    uuid.UUID
	Text        string
}

type RecordEvidenceRequest struct {
	RequestID   uuid.UUID
	PrincipalID uuid.UUID
	QuestionID  uuid.UUID
	Source      string
	Content     string
	RetrievedAt time.Time
}

type PublishFindingRequest struct {
	RequestID   uuid.UUID
	PrincipalID uuid.UUID
	QuestionID  uuid.UUID
	Claim       string
	EvidenceIDs []uuid.UUID
}

type IssueRecommendationRequest struct {
	RequestID      uuid.UUID
	PrincipalID    uuid.UUID
	FindingID      uuid.UUID
	ProposedAction string
}
