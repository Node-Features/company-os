package objective

import (
	"time"

	"github.com/google/uuid"
)

// Status is the Objective lifecycle status. This slice (ROADMAP.md Phase 4
// Slice 4, the creation gate) only ever writes StatusProposed — the
// APPROVED/ACTIVE/BLOCKED/COMPLETED/CANCELLED/SUPERSEDED transitions
// docs/domain/objective.md's full lifecycle describes are future work, not
// built here.
type Status string

const (
	StatusProposed Status = "PROPOSED"
	StatusApproved Status = "APPROVED"
	StatusRetired  Status = "RETIRED"
)

// SourceType is which adaptive-loop contract a proposed Objective traces
// back to (docs/architecture/departments.md's "Objective creation gate":
// "A Finding, Recommendation, Evaluation, ResourceEvaluation... can only
// propose an Objective"). Metric/event/escalation/feedback-result sources
// are out of scope this slice — only the four contracts a distinct
// Application use case already exists for.
type SourceType string

const (
	SourceFinding            SourceType = "FINDING"
	SourceRecommendation     SourceType = "RECOMMENDATION"
	SourceEvaluation         SourceType = "EVALUATION"
	SourceResourceEvaluation SourceType = "RESOURCE_EVALUATION"
)

// Objective is the organizational goal a Workflow serves. See
// docs/domain/objective.md. Narrower than that document's full minimum
// contract — department/parent-objective references, desired outcomes,
// exclusions, priority, time bounds, success MetricDefinition, and
// resource-constraint/risk references are all deferred, same convention
// every prior Phase 4 slice used. SourceType/SourceID trace which
// adaptive-loop contract proposed this Objective;
// GovernanceDecisionID/ApprovalID trace the governed proposal that created
// it (ApprovalID is nil only in the structurally-unreachable direct-ALLOW
// path — this slice's policy always requires human approval first).
type Objective struct {
	ObjectiveID          uuid.UUID
	OrganizationID       uuid.UUID
	Title                string
	Intent               string
	Status               Status
	OwnerPrincipalID     uuid.UUID
	SourceType           SourceType
	SourceID             uuid.UUID
	GovernanceDecisionID uuid.UUID
	ApprovalID           *uuid.UUID
	CreatedAt            time.Time
}
