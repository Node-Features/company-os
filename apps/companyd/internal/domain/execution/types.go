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

// IntentStatus is execution_intents.status (docs/domain/workflow.md's
// ExecutionIntent, execution.md's claim lifecycle around it).
type IntentStatus string

const (
	IntentPending IntentStatus = "PENDING"
	IntentClaimed IntentStatus = "CLAIMED"
	IntentClosed  IntentStatus = "CLOSED"
)

// ExecutionUnit groups one ExecutionIntent with every ExecutionAttempt made
// against it, oldest first, for a Workflow. This is a read-model grouping
// for status/UI projections — it is not itself a persisted or authoritative
// record; the intent and attempt rows it groups are. Deliberately not
// called "Node": that word is reserved for a CompanyOS runtime/compute
// node — an addressable participant with its own resources and
// capabilities that a scheduler places work onto (see
// docs/architecture/node.md) — which is a different concept from this
// per-Workflow dispatch bookkeeping and does not exist in this codebase yet.
type ExecutionUnit struct {
	IntentID     uuid.UUID
	IntentStatus IntentStatus
	DueAt        time.Time
	Attempts     []ExecutionAttempt
}
