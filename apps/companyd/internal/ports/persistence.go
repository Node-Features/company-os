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
	// not match the current stored version. closeOutstandingIntent, when
	// true, atomically marks any still-PENDING ExecutionIntent for this
	// Workflow CLOSED in the same transaction — CANCEL_WORKFLOW's durable
	// record that Runtime must not claim it, per
	// docs/domain/execution.md's invariants. It does not touch an intent
	// already CLAIMED; an in-flight attempt is not assumed to stop
	// immediately.
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
		closeOutstandingIntent bool,
	) error

	// SaveGovernanceDecision persists a GovernanceDecision independent of
	// a Workflow transition (used by the dispatch-time re-check, which
	// records a decision without necessarily committing Workflow state).
	SaveGovernanceDecision(ctx context.Context, decisionID, orgID, requestID, correlationID, principalID uuid.UUID,
		action, resourceType, resourceID, proposalDigest, trustedContextDigest, policyVersion, autonomyLevel, outcome string,
		matchedRuleID, reason *string) error

	// GovernanceDecisionExists reports whether a GovernanceDecision has
	// been persisted for the given organization/action/resource — used to
	// verify SaveGovernanceDecision's unconditional-persistence invariant
	// (governance.md: "Policy, authority, approval, and decision records
	// are persisted before dependent execution continues"). Looked up by
	// resource identity rather than decision ID: several callers (e.g.
	// CreateWorkflow) never expose the decision ID on their Result — it is
	// not meant to be public API — so a test verifying persistence must be
	// able to ask "was this action's decision recorded?" without it.
	GovernanceDecisionExists(ctx context.Context, orgID uuid.UUID, action, resourceType, resourceID string) (bool, error)

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

	// ListExecutionUnits serves the status read endpoint's units field:
	// every ExecutionIntent for this Workflow plus every ExecutionAttempt
	// made against each, oldest first. A read-only projection, not a claim.
	ListExecutionUnits(ctx context.Context, orgID, workflowID uuid.UUID) ([]execution.ExecutionUnit, error)
}

// PendingCommandRepository persists a REQUIRE_APPROVAL command and its
// Approval, and resolves a human decision against it (ROADMAP.md Phase 3
// Slice 2).
type PendingCommandRepository interface {
	CreatePendingApproval(ctx context.Context, pc *command.PendingCommand, decisionID uuid.UUID, appr *approval.Approval) error

	// ResolveApproval records a human decision — PENDING -> APPROVED or
	// PENDING -> REJECTED — and returns the updated Approval plus its
	// associated PendingCommand so the caller can proceed (resume on
	// approve, nothing further on reject — the PendingCommand is already
	// closed in the same transaction). The WHERE status='PENDING' guard is
	// the concurrency control: a second call against an already-decided
	// Approval affects zero rows and returns ErrConflict, preventing
	// double-resolution/double-resumption without a separate lock.
	ResolveApproval(ctx context.Context, approvalID, decidedByPrincipalID uuid.UUID, approve bool, reason *string) (*command.PendingCommand, *approval.Approval, error)
}
