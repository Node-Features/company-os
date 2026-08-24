package application

import (
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/google/uuid"
)

// CreateWorkflowRequest is the ApplicationRequest for CREATE_WORKFLOW. The
// Objective/Definition/Capability references are implied by fixtures.Registry
// this slice (decision #5/#6) rather than carried on the request.
// RequestingPrincipalID is the caller-resolved, context-authenticated
// Principal (docs/audit/gap-approval-principal-attribution.md) — server-set
// by the HTTP handler from PrincipalFromContext, never client-asserted.
type CreateWorkflowRequest struct {
	RequestID             uuid.UUID
	IdempotencyKey        string
	RequestingPrincipalID uuid.UUID
}

// StartWorkflowRequest is the ApplicationRequest for START_WORKFLOW.
// RequestingPrincipalID: see CreateWorkflowRequest's doc comment.
type StartWorkflowRequest struct {
	RequestID             uuid.UUID
	IdempotencyKey        string
	WorkflowID            uuid.UUID
	ExpectedVersion       int64
	RequestingPrincipalID uuid.UUID
}

// SubmitResultRequest is the ApplicationRequest Runtime uses to submit
// execution evidence and a proposed Result (docs/architecture/application.md#runtime-result-use-case).
type SubmitResultRequest struct {
	RequestID       uuid.UUID
	IdempotencyKey  string
	ResultID        uuid.UUID
	ExpectedVersion int64
}

// CancelWorkflowRequest is the ApplicationRequest for CANCEL_WORKFLOW.
// RequestingPrincipalID: see CreateWorkflowRequest's doc comment.
type CancelWorkflowRequest struct {
	RequestID             uuid.UUID
	IdempotencyKey        string
	WorkflowID            uuid.UUID
	ExpectedVersion       int64
	RequestingPrincipalID uuid.UUID
}

// ResolveApprovalRequest is the ApplicationRequest for a human decision on
// a PENDING Approval (docs/domain/approval.md's lifecycle step 3).
// DecidingPrincipal is the caller-resolved, context-authenticated Principal
// (docs/audit/gap-approval-principal-attribution.md) — server-set by the
// HTTP handler from PrincipalFromContext, never client-asserted.
// ResolveApproval structurally rejects a non-HUMAN or self-approving
// decider regardless of what's passed here (ports.ErrNonHumanDecider /
// ErrSelfApproval), so an incorrectly-wired caller fails closed, not open.
type ResolveApprovalRequest struct {
	ApprovalID        uuid.UUID
	Approve           bool
	Reason            *string
	DecidingPrincipal principal.Principal
}
