// Package observability implements docs/architecture/observability.md:
// correlation identity, structured logging, and operational metrics for
// CompanyOS's execution lifecycle (Command -> Governance decision -> Domain
// transition -> Execution intent -> Lease/Claim -> Execution attempt ->
// Provider execution -> Result). Diagnostic output only — never
// authoritative domain state, per that document's Non-authority invariant.
package observability

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// ExecutionContext is the correlation identity for one execution's
// progress through the lifecycle. Every field already exists as a domain
// field elsewhere (command.WorkflowCommandEnvelope.CorrelationID,
// execution.ExecutionAttempt.AttemptID, policy.GovernanceDecision.Outcome,
// ...) — this type carries them through context.Context and into
// structured logs/metrics; it does not mint a second identity scheme.
//
// Every field is optional and populated progressively as a request moves
// through HTTP, Application, Governance, and Runtime. A field left at its
// zero value is simply omitted from Attrs(), never synthesized.
type ExecutionContext struct {
	CorrelationID        uuid.UUID
	CommandID            uuid.UUID
	OrganizationID       uuid.UUID
	WorkflowID           uuid.UUID
	ExecutionIntentID    uuid.UUID
	ExecutionAttemptID   uuid.UUID
	PrincipalID          uuid.UUID
	GovernanceDecisionID uuid.UUID
	GovernanceOutcome    string
	Provider             string
	RetryCount           *int
	LeaseState           string
	FailureReason        string
	LifecycleState       string
	StartedAt            *time.Time
	CompletedAt          *time.Time
}

type contextKey struct{}

// WithExecutionContext attaches ec to ctx, merged over whatever
// ExecutionContext is already attached (see With) — so each layer (HTTP,
// Application, Governance, Runtime) can progressively enrich the same
// identity as new IDs become known, rather than replacing it.
func WithExecutionContext(ctx context.Context, ec ExecutionContext) context.Context {
	merged := FromContext(ctx).With(ec)
	return context.WithValue(ctx, contextKey{}, merged)
}

// FromContext returns the ExecutionContext attached to ctx, or the zero
// value if none has been attached yet.
func FromContext(ctx context.Context) ExecutionContext {
	ec, _ := ctx.Value(contextKey{}).(ExecutionContext)
	return ec
}

// With returns a copy of ec with every non-zero field of other applied
// over it, so an earlier layer's fields survive a later layer enriching
// only the fields it knows about.
func (ec ExecutionContext) With(other ExecutionContext) ExecutionContext {
	merged := ec
	if other.CorrelationID != uuid.Nil {
		merged.CorrelationID = other.CorrelationID
	}
	if other.CommandID != uuid.Nil {
		merged.CommandID = other.CommandID
	}
	if other.OrganizationID != uuid.Nil {
		merged.OrganizationID = other.OrganizationID
	}
	if other.WorkflowID != uuid.Nil {
		merged.WorkflowID = other.WorkflowID
	}
	if other.ExecutionIntentID != uuid.Nil {
		merged.ExecutionIntentID = other.ExecutionIntentID
	}
	if other.ExecutionAttemptID != uuid.Nil {
		merged.ExecutionAttemptID = other.ExecutionAttemptID
	}
	if other.PrincipalID != uuid.Nil {
		merged.PrincipalID = other.PrincipalID
	}
	if other.GovernanceDecisionID != uuid.Nil {
		merged.GovernanceDecisionID = other.GovernanceDecisionID
	}
	if other.GovernanceOutcome != "" {
		merged.GovernanceOutcome = other.GovernanceOutcome
	}
	if other.Provider != "" {
		merged.Provider = other.Provider
	}
	if other.RetryCount != nil {
		merged.RetryCount = other.RetryCount
	}
	if other.LeaseState != "" {
		merged.LeaseState = other.LeaseState
	}
	if other.FailureReason != "" {
		merged.FailureReason = other.FailureReason
	}
	if other.LifecycleState != "" {
		merged.LifecycleState = other.LifecycleState
	}
	if other.StartedAt != nil {
		merged.StartedAt = other.StartedAt
	}
	if other.CompletedAt != nil {
		merged.CompletedAt = other.CompletedAt
	}
	return merged
}

// Attrs renders ec as structured slog attributes, omitting any field
// still at its zero value rather than logging a meaningless empty ID —
// never a string-interpolated message, per observability.md's structured-
// logging decision.
func (ec ExecutionContext) Attrs() []slog.Attr {
	var attrs []slog.Attr
	addString := func(key, value string) {
		if value != "" {
			attrs = append(attrs, slog.String(key, value))
		}
	}
	addID := func(key string, id uuid.UUID) {
		if id != uuid.Nil {
			attrs = append(attrs, slog.String(key, id.String()))
		}
	}
	addID("correlation_id", ec.CorrelationID)
	addID("command_id", ec.CommandID)
	addID("organization_id", ec.OrganizationID)
	addID("workflow_id", ec.WorkflowID)
	addID("execution_intent_id", ec.ExecutionIntentID)
	addID("execution_attempt_id", ec.ExecutionAttemptID)
	addID("principal_id", ec.PrincipalID)
	addID("governance_decision_id", ec.GovernanceDecisionID)
	addString("governance_outcome", ec.GovernanceOutcome)
	addString("provider", ec.Provider)
	if ec.RetryCount != nil {
		attrs = append(attrs, slog.Int("retry_count", *ec.RetryCount))
	}
	addString("lease_state", ec.LeaseState)
	addString("failure_reason", ec.FailureReason)
	addString("lifecycle_state", ec.LifecycleState)
	if ec.StartedAt != nil {
		attrs = append(attrs, slog.Time("started_at", *ec.StartedAt))
	}
	if ec.CompletedAt != nil {
		attrs = append(attrs, slog.Time("completed_at", *ec.CompletedAt))
	}
	return attrs
}
