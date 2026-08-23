package knowledge

import (
	"time"

	"github.com/google/uuid"
)

// Status is KnowledgeItem's lifecycle. See docs/domain/knowledge.md.
// Phase 5 Slice 1 (ingestion/versioning) only ever writes StatusDraft;
// Phase 5 Slice 2 (knowledge.approve) additionally writes StatusApproved
// and StatusRejected. StatusInReview/StatusSuperseded/StatusExpired remain
// unexercised — future work.
type Status string

const (
	StatusDraft      Status = "DRAFT"
	StatusInReview   Status = "IN_REVIEW"
	StatusApproved   Status = "APPROVED"
	StatusRejected   Status = "REJECTED"
	StatusSuperseded Status = "SUPERSEDED"
	StatusExpired    Status = "EXPIRED"
)

// SourceType is the kind of thing a KnowledgeItem was captured from. Phase 5
// Slice 1 supports exactly one.
type SourceType string

const (
	SourceResearchFinding SourceType = "RESEARCH_FINDING"
)

// ProducedByMethod records how a KnowledgeItem's content was produced. Phase
// 5 Slice 1 only ever verbatim-copies an existing source's content — no
// model-assisted extraction or synthesis exists yet.
type ProducedByMethod string

const (
	MethodSourceVerbatim ProducedByMethod = "SOURCE_VERBATIM"
)

// Classification is access/retention classification (knowledge.md step 2:
// "classify access, retention, and tenant scope"). Phase 5 Slice 1 only ever
// writes one value — the taxonomy itself is knowledge.md's own open
// question ("What taxonomy and scope hierarchy are required for the first
// vertical slice?"), not invented here.
type Classification string

const (
	ClassificationInternal Classification = "INTERNAL"
)

// KnowledgeItem is one immutable version of a versioned, organization-scoped
// claim (docs/domain/knowledge.md). KnowledgeItemID is stable across
// versions; each new normalization of the same source is a new row with an
// incremented Version, never an overwrite.
//
// Deferred, per docs/domain/knowledge.md's own minimum contract: purpose,
// audience, validity conditions, review deadline, retention, and
// contradiction/supersession/derivation links.
type KnowledgeItem struct {
	KnowledgeItemID       uuid.UUID
	OrganizationID        uuid.UUID
	Version               int
	Claim                 string
	ContentDigest         string
	Classification        Classification
	SourceType            SourceType
	SourceID              uuid.UUID
	ProducedByPrincipalID uuid.UUID
	ProducedByMethod      ProducedByMethod
	Status                Status
	// DuplicateOfItemID is a review signal only (knowledge.md: "detect
	// potential duplicates... as review signals, not automatic merges") —
	// it never causes a merge or blocks capture.
	DuplicateOfItemID *uuid.UUID
	CreatedAt         time.Time
	// ReviewerPrincipalID, GovernanceDecisionID, ApprovalID, and ReviewedAt
	// are all nil until knowledge.approve (Phase 5 Slice 2) resolves —
	// docs/domain/knowledge.md: "Governance decision, authenticated
	// reviewer Principal, Authority/policy versions, and review evidence
	// when approved." ReviewerPrincipalID is the actual decider (always
	// fixtures.Registry.ApproverPrincipal() today), distinct from whichever
	// Principal requested the review.
	ReviewerPrincipalID *uuid.UUID
	GovernanceDecisionID *uuid.UUID
	ApprovalID           *uuid.UUID
	ReviewedAt           *time.Time
}
