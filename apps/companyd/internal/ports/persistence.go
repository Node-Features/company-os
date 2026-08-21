// Package ports declares the interfaces Application, Kernel, and Runtime
// consume, and that internal/adapters implements against concrete
// infrastructure. See docs/architecture/persistence.md#required-persistence-ports.
package ports

import (
	"context"
	"errors"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/command"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/execution"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/google/uuid"
)

// ErrNotFound is returned by LoadWorkflow when no Workflow exists for the
// given identity — CREATE_WORKFLOW's legality requires verifying absence,
// not inferring it from a nil result.
var ErrNotFound = errors.New("ports: not found")

// ErrConflict is returned when a compare-and-write's expected version (or,
// for CreateWorkflow, expected absence) does not match current state.
var ErrConflict = errors.New("ports: version conflict")

// ResultDecision tells CommitTransition to mark a Result's accepted/decided_at
// columns within the same transaction as the Workflow state write, so a
// Result's decision is never observably out of sync with the transition it
// caused.
type ResultDecision struct {
	ResultID uuid.UUID
	Accepted bool
}

// AuthoritativeStateRepository is the first-slice Workflow aggregate port.
// One Workflow is addressed by (OrganizationID, WorkflowID) with an opaque
// monotonic version; every write is compare-and-write, never a merge. See
// docs/architecture/persistence.md.
type AuthoritativeStateRepository interface {
	LoadWorkflow(ctx context.Context, orgID, workflowID uuid.UUID) (*workflow.Workflow, error)

	// CreateWorkflow atomically writes the new Workflow (version 1), its
	// DomainEvents, and their outbox rows in one transaction. Returns
	// ErrConflict if a Workflow already exists for this identity.
	CreateWorkflow(ctx context.Context, w *workflow.Workflow, events []event.DomainEvent, decisionID uuid.UUID) error

	// CommitTransition atomically writes the next Workflow version, its
	// DomainEvents/outbox rows, the GovernanceDecision reference, an
	// optional new ExecutionIntent, and optional PendingCommand closure /
	// Approval consumption. Returns ErrConflict if expectedVersion does
	// not match the current stored version.
	CommitTransition(
		ctx context.Context,
		w *workflow.Workflow,
		expectedVersion int64,
		events []event.DomainEvent,
		decisionID uuid.UUID,
		intent *workflow.ExecutionIntent,
		closePendingCommand *uuid.UUID,
		consumeApproval *uuid.UUID,
		decideResult *ResultDecision,
	) error

	// SaveGovernanceDecision persists a GovernanceDecision independent of
	// a Workflow transition (used by the dispatch-time re-check, which
	// records a decision without necessarily committing Workflow state).
	SaveGovernanceDecision(ctx context.Context, decisionID, orgID, requestID, correlationID, principalID uuid.UUID,
		action, resourceType, resourceID, proposalDigest, trustedContextDigest, policyVersion, autonomyLevel, outcome string,
		matchedRuleID, reason *string) error

	// IdempotencyLookup and IdempotencyStore implement the replay guard
	// application.md requires: retrying the same logical request returns
	// the original outcome rather than creating a second transition.
	IdempotencyLookup(ctx context.Context, orgID uuid.UUID, key string) (found bool, outcome string, err error)
	IdempotencyStore(ctx context.Context, orgID, requestID uuid.UUID, key, outcome string) error
}

// ExecutionRepository is the Runtime execution-mechanics port: claiming
// due ExecutionIntents under a lease, recording attempt status, and
// persisting Results. See docs/domain/execution.md.
type ExecutionRepository interface {
	// ClaimDueIntents claims up to limit due, PENDING intents under
	// FOR UPDATE SKIP LOCKED, creates one CLAIMED ExecutionAttempt per
	// claim, and returns both.
	ClaimDueIntents(ctx context.Context, orgID uuid.UUID, limit int, leaseDuration time.Duration, workerID string) ([]execution.ClaimedExecution, error)

	RecordDispatched(ctx context.Context, attemptID uuid.UUID, fencingToken int64, providerRunID string) error

	// RecordTerminal transitions an attempt to a terminal or
	// FAILED_RETRYABLE status.
	RecordTerminal(ctx context.Context, attemptID uuid.UUID, fencingToken int64, status execution.AttemptStatus, resultID *uuid.UUID) error

	// ScheduleRetry resets a FAILED_RETRYABLE attempt's intent back to
	// PENDING at dueAt, so the next Sweep re-claims it as a new attempt
	// against the same intent (logical operation), per execution.md's
	// "reuses logical-operation identity, new ExecutionAttempt identity
	// each time."
	ScheduleRetry(ctx context.Context, orgID, intentID uuid.UUID, dueAt time.Time) error

	SaveResult(ctx context.Context, r *result.Result) error
	GetResult(ctx context.Context, orgID, resultID uuid.UUID) (*result.Result, error)

	// GetLatestResult serves the status read endpoint's latestResult field.
	// Returns ErrNotFound if the Workflow has no Result yet.
	GetLatestResult(ctx context.Context, orgID, workflowID uuid.UUID) (*result.Result, error)
}

// PendingCommandRepository persists a REQUIRE_APPROVAL command and its
// Approval — implemented even though this slice's always-ALLOW policy
// never reaches it, per application.md's "never bypassed."
type PendingCommandRepository interface {
	CreatePendingApproval(ctx context.Context, pc *command.PendingCommand, decisionID uuid.UUID, appr *approval.Approval) error
}
