// Package research holds the Research department's domain types
// (docs/departments/research.md, docs/workflows/research-loop.md). Minimal
// field sets only — narrower than research.md's full aspirational
// contract, per research-loop.md's own stated scope boundary.
package research

import (
	"time"

	"github.com/google/uuid"
)

// Signal is a provenance-bearing indication that something may have
// changed. This slice handles exactly one SourceType.
type Signal struct {
	SignalID               uuid.UUID
	OrganizationID         uuid.UUID
	SourceType             string
	Description            string
	SubmittedByPrincipalID uuid.UUID
	SubmittedAt            time.Time
}

// SourceTypeProviderModelChange is the one Signal type this slice handles —
// research-loop.md's chosen narrow signal type.
const SourceTypeProviderModelChange = "PROVIDER_MODEL_CHANGE"

// QuestionStatus is ResearchQuestion's lifecycle.
type QuestionStatus string

const (
	QuestionOpen   QuestionStatus = "OPEN"
	QuestionClosed QuestionStatus = "CLOSED"
)

// ResearchQuestion is an inquiry opened from a Signal. Minimal fields only —
// research.md's full envelope (hypotheses, quality/freshness criteria,
// source constraints, time/cost limits, conflicts of interest, security
// classification) is out of scope this slice.
type ResearchQuestion struct {
	QuestionID     uuid.UUID
	OrganizationID uuid.UUID
	SignalID       uuid.UUID
	Text           string
	Status         QuestionStatus
	CreatedAt      time.Time
}

// Evidence is an attributable observation recorded against a
// ResearchQuestion.
type Evidence struct {
	EvidenceID     uuid.UUID
	OrganizationID uuid.UUID
	QuestionID     uuid.UUID
	Source         string
	Content        string
	RetrievedAt    time.Time
	CreatedAt      time.Time
}

// FindingStatus is Finding's lifecycle.
type FindingStatus string

const (
	FindingPublished  FindingStatus = "PUBLISHED"
	FindingSuperseded FindingStatus = "SUPERSEDED"
)

// Finding is a versioned claim derived from cited Evidence. It cannot exceed
// what its evidence supports (research.md invariant) — enforced by
// requiring at least one EvidenceID at creation
// (internal/departments/research.ValidatePublishFinding).
type Finding struct {
	FindingID      uuid.UUID
	OrganizationID uuid.UUID
	QuestionID     uuid.UUID
	Claim          string
	EvidenceIDs    []uuid.UUID
	Status         FindingStatus
	CreatedAt      time.Time
}

// RecommendationStatus is Recommendation's lifecycle.
type RecommendationStatus string

const (
	RecommendationProposed RecommendationStatus = "PROPOSED"
	RecommendationExpired  RecommendationStatus = "EXPIRED"
)

// Recommendation is a proposed course of action linked to a Finding. It is
// neither authorization nor an Objective (research.md invariant) — turning
// one into an Objective proposal is ROADMAP.md Phase 4 Slice 4's job, not
// built here.
type Recommendation struct {
	RecommendationID uuid.UUID
	OrganizationID   uuid.UUID
	FindingID        uuid.UUID
	ProposedAction   string
	Status           RecommendationStatus
	CreatedAt        time.Time
}
