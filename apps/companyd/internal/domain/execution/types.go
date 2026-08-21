package execution

import (
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/google/uuid"
)

// AttemptStatus is the ExecutionAttempt lifecycle status. See
// docs/domain/execution.md.
type AttemptStatus string

const (
	StatusClaimed         AttemptStatus = "CLAIMED"
	StatusDispatched      AttemptStatus = "DISPATCHED"
	StatusWaiting         AttemptStatus = "WAITING"
	StatusSucceeded       AttemptStatus = "SUCCEEDED"
	StatusFailedRetryable AttemptStatus = "FAILED_RETRYABLE"
	StatusFailedTerminal  AttemptStatus = "FAILED_TERMINAL"
	StatusCancelled       AttemptStatus = "CANCELLED"
	StatusLeaseExpired    AttemptStatus = "LEASE_EXPIRED"
)

// legalTransitions is the exact table in docs/domain/execution.md:
// CLAIMED -> DISPATCHED | LEASE_EXPIRED
// DISPATCHED -> WAITING | SUCCEEDED | FAILED_RETRYABLE | FAILED_TERMINAL | CANCELLED | LEASE_EXPIRED
// WAITING -> DISPATCHED | CANCELLED | LEASE_EXPIRED
var legalTransitions = map[AttemptStatus][]AttemptStatus{
	StatusClaimed: {StatusDispatched, StatusLeaseExpired},
	StatusDispatched: {
		StatusWaiting, StatusSucceeded, StatusFailedRetryable,
		StatusFailedTerminal, StatusCancelled, StatusLeaseExpired,
	},
	StatusWaiting: {StatusDispatched, StatusCancelled, StatusLeaseExpired},
}

// CanTransitionTo reports whether next is a legal transition from s.
func (s AttemptStatus) CanTransitionTo(next AttemptStatus) bool {
	for _, n := range legalTransitions[s] {
		if n == next {
			return true
		}
	}
	return false
}

// IsTerminal reports whether no further transition is legal from s.
func (s AttemptStatus) IsTerminal() bool {
	return len(legalTransitions[s]) == 0
}

// ExecutionAttempt is one bounded attempt to fulfill an ExecutionIntent.
// Lease and Checkpoint (docs/domain/execution.md) are folded into columns
// here rather than separate types — this slice's only leasable subject is
// the attempt itself, with no partial progress to checkpoint (first-slice
// plan decision #9).
type ExecutionAttempt struct {
	AttemptID           uuid.UUID
	OrganizationID       uuid.UUID
	IntentID             uuid.UUID
	WorkflowID           uuid.UUID
	WorkflowVersion      int64
	LogicalOperationID   string
	AttemptNumber        int
	CapabilityRequestID  uuid.UUID
	Status               AttemptStatus
	LeaseOwner           *string
	LeaseFencingToken    *int64
	LeaseExpiresAt       *time.Time
	ProviderRunID        *string
	Checkpoint           map[string]any
	CreatedAt            time.Time
	LastHeartbeatAt      *time.Time
	NextAttemptDueAt     *time.Time
	TerminalAt           *time.Time
	ResultID             *uuid.UUID
}

// ClaimedExecution pairs a newly claimed ExecutionAttempt with the
// ExecutionIntent it was claimed against, since Runtime needs both to
// dispatch (capability, inputs) and to report back (workflow/version).
type ClaimedExecution struct {
	Attempt ExecutionAttempt
	Intent  workflow.ExecutionIntent
}
