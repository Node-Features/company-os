package result

import (
	"time"

	"github.com/google/uuid"
)

// Outcome is the reported capability outcome. See docs/domain/result.md.
type Outcome string

const (
	OutcomeSucceeded     Outcome = "SUCCEEDED"
	OutcomeFailed        Outcome = "FAILED"
	OutcomeCancelled     Outcome = "CANCELLED"
	OutcomeTimedOut      Outcome = "TIMED_OUT"
	OutcomePartial       Outcome = "PARTIAL"
	OutcomeIndeterminate Outcome = "INDETERMINATE"
)

// AcceptsResult reports whether this outcome drives ACCEPT_WORKFLOW_RESULT.
func (o Outcome) AcceptsResult() bool { return o == OutcomeSucceeded }

// RejectsResult reports whether this outcome drives REJECT_WORKFLOW_RESULT.
// PARTIAL is treated as FAILED in this slice per docs/domain/workflow.md.
func (o Outcome) RejectsResult() bool {
	switch o {
	case OutcomeFailed, OutcomeTimedOut, OutcomePartial:
		return true
	default:
		return false
	}
}

// Result is the reported outcome of one ExecutionIntent's execution
// attempt. See docs/domain/result.md.
type Result struct {
	ResultID             uuid.UUID
	OrganizationID       uuid.UUID
	ResultType           string
	WorkflowID           uuid.UUID
	ObjectiveID          uuid.UUID
	IntentID             uuid.UUID
	AttemptID            uuid.UUID
	CapabilityRequestID  uuid.UUID
	IdempotencyKey       string
	ProducingPrincipalID uuid.UUID
	ProviderAdapter      string
	ModelID              string
	Outcome              Outcome
	Output               map[string]any
	ErrorClassification  *string
	Retryable            *bool
	StartedAt            time.Time
	ObservedAt           time.Time
	ReportedAt           time.Time
	Accepted             *bool
	DecidedAt            *time.Time
}
