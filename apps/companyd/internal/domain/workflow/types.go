package workflow

import (
	"time"

	"github.com/google/uuid"
)

// State is one of the five first-slice authoritative Workflow states.
// See docs/domain/workflow.md#first-slice-lifecycle.
type State string

const (
	StatePlanned   State = "PLANNED"
	StateReady     State = "READY"
	StateCompleted State = "COMPLETED"
	StateFailed    State = "FAILED"
	StateCancelled State = "CANCELLED"
)

// IsTerminal reports whether no further command produces a transition from
// this state in the first slice.
func (s State) IsTerminal() bool {
	switch s {
	case StateCompleted, StateFailed, StateCancelled:
		return true
	default:
		return false
	}
}

// WorkflowDefinition specifies legal stages, transitions, and required
// capability for one class of Workflow. Commands are stored as strings
// (matching command.CommandType's underlying type) so this package has no
// dependency on internal/domain/command. See docs/domain/workflow.md.
type WorkflowDefinition struct {
	DefinitionID              uuid.UUID
	Version                   int
	Name                      string
	States                    []State
	Commands                  []string
	RequiredCapabilityID      uuid.UUID
	RequiredCapabilityVersion int
	TerminalStates            []State
	Active                    bool
}

// Workflow is one versioned, organization-scoped instance of a
// WorkflowDefinition. It is the sole authoritative aggregate this slice.
// See docs/domain/workflow.md.
type Workflow struct {
	OrganizationID        uuid.UUID
	WorkflowID            uuid.UUID
	Version               int64
	DefinitionID          uuid.UUID
	DefinitionVersion     int
	ObjectiveID           uuid.UUID
	State                 State
	InitiatingPrincipalID uuid.UUID
	CorrelationID         uuid.UUID
	CausationID           *uuid.UUID
	Inputs                map[string]any
	WaitReason            *string
	TerminalReason        *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ExecutionIntent is the immutable, persisted request for Runtime to
// perform one bounded capability step. Intent, not proof of dispatch or
// completion. See docs/domain/workflow.md and docs/domain/execution.md.
type ExecutionIntent struct {
	IntentID                    uuid.UUID
	OrganizationID              uuid.UUID
	WorkflowID                  uuid.UUID
	WorkflowVersion             int64
	CapabilityDefinitionID      uuid.UUID
	CapabilityDefinitionVersion int
	GovernanceDecisionID        uuid.UUID
	IdempotencyKey              string
	Constraints                 map[string]any
	DueAt                       time.Time
	Inputs                      map[string]any
}
