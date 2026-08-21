package capability

import (
	"time"

	"github.com/google/uuid"
)

// RetryPolicy is owned per-capability, not invented ad hoc by Runtime, per
// docs/domain/execution.md. See first-slice plan decision #4 for the
// concrete constants used by the bounded-text-generation fixture.
type RetryPolicy struct {
	MaxAttempts int
	BackoffBase time.Duration
	BackoffMax  time.Duration
	FullJitter  bool
}

// CapabilityDefinition is the provider-independent outcome and dispatch
// contract. See docs/domain/capability.md.
type CapabilityDefinition struct {
	CapabilityDefinitionID uuid.UUID
	Version                int
	Name                   string
	InputSchema            map[string]string
	OutputSchema           map[string]string
	Timeout                time.Duration
	Retryable              bool
	RiskLevel              string
	// GovernanceAction is the Action this capability's dispatch is
	// evaluated against immediately before an external effect
	// (docs/architecture/application.md#dispatch-time-governance).
	GovernanceAction string
	Retry            RetryPolicy
	Active           bool
}

// CapabilityRequest is one bounded request to invoke a CapabilityDefinition.
type CapabilityRequest struct {
	CapabilityRequestID         uuid.UUID
	CapabilityDefinitionID      uuid.UUID
	CapabilityDefinitionVersion int
	Inputs                      map[string]any
}
