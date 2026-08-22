package event

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the canonical event record. See docs/domain/event.md.
type DomainEvent struct {
	EventID        uuid.UUID
	OrganizationID uuid.UUID
	EventType      string
	SchemaVersion  int
	SubjectType    string
	SubjectID      uuid.UUID
	SubjectVersion int64
	OccurredAt     time.Time
	CorrelationID  uuid.UUID
	CausationID    *uuid.UUID
	WorkflowID     *uuid.UUID
	ObjectiveID    *uuid.UUID
	PrincipalID    *uuid.UUID
	Payload        map[string]any
}

// First-slice WorkflowEvent types (workflow.md's produced Event types).
const (
	TypeWorkflowCreated   = "WORKFLOW_CREATED"
	TypeWorkflowStarted   = "WORKFLOW_STARTED"
	TypeWorkflowCompleted = "WORKFLOW_COMPLETED"
	TypeWorkflowFailed    = "WORKFLOW_FAILED"
	TypeWorkflowCancelled = "WORKFLOW_CANCELLED"
)
