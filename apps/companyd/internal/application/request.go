package application

import "github.com/google/uuid"

// CreateWorkflowRequest is the ApplicationRequest for CREATE_WORKFLOW. The
// requesting Principal and Objective/Definition/Capability references are
// implied by fixtures.Registry this slice (decision #5/#6) rather than
// carried on the request.
type CreateWorkflowRequest struct {
	RequestID      uuid.UUID
	IdempotencyKey string
}

// StartWorkflowRequest is the ApplicationRequest for START_WORKFLOW.
type StartWorkflowRequest struct {
	RequestID       uuid.UUID
	IdempotencyKey  string
	WorkflowID      uuid.UUID
	ExpectedVersion int64
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
type CancelWorkflowRequest struct {
	RequestID       uuid.UUID
	IdempotencyKey  string
	WorkflowID      uuid.UUID
	ExpectedVersion int64
}

// ResolveApprovalRequest is the ApplicationRequest for a human decision on
// a PENDING Approval (docs/domain/approval.md's lifecycle step 3). The
// deciding Principal is never carried on the request — like every other
// Principal reference this slice, it's server-resolved from
// fixtures.Registry.ApproverPrincipal(), never client-asserted.
type ResolveApprovalRequest struct {
	ApprovalID uuid.UUID
	Approve    bool
	Reason     *string
}
