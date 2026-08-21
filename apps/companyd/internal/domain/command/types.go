package command

import (
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/event"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/google/uuid"
)

// CommandType is one of the first-slice command vocabulary entries. See
// docs/domain/command.md. CANCEL_WORKFLOW is named but not wired to a
// legal Kernel transition path in this slice (out of scope).
type CommandType string

const (
	CreateWorkflow      CommandType = "CREATE_WORKFLOW"
	StartWorkflow       CommandType = "START_WORKFLOW"
	AcceptWorkflowResult CommandType = "ACCEPT_WORKFLOW_RESULT"
	RejectWorkflowResult CommandType = "REJECT_WORKFLOW_RESULT"
	CancelWorkflow       CommandType = "CANCEL_WORKFLOW"
)

// ActionFor is the exact Action mapping table in docs/domain/command.md.
var ActionFor = map[CommandType]string{
	CreateWorkflow:       "workflow.create",
	StartWorkflow:        "workflow.start",
	AcceptWorkflowResult: "workflow.result.accept",
	RejectWorkflowResult: "workflow.result.reject",
	CancelWorkflow:       "workflow.cancel",
}

// WorkflowCommandEnvelope is the transport-independent request to change
// Workflow state. See docs/domain/command.md.
type WorkflowCommandEnvelope struct {
	SchemaVersion         int
	CommandID             uuid.UUID
	RequestID             uuid.UUID
	IdempotencyKey        string
	CommandType           CommandType
	OrganizationID        uuid.UUID
	WorkflowID            uuid.UUID
	// ExpectedVersion is nil only for CREATE_WORKFLOW, where the command
	// instead asserts the aggregate is explicitly absent.
	ExpectedVersion       *int64
	ObjectiveID           uuid.UUID
	DefinitionID          uuid.UUID
	DefinitionVersion     int
	RequestingPrincipalID uuid.UUID
	Inputs                map[string]any
	// ResultID is set only for ACCEPT_WORKFLOW_RESULT/REJECT_WORKFLOW_RESULT.
	ResultID              *uuid.UUID
	DeclaredTime          time.Time
	CorrelationID         uuid.UUID
	CausationID           *uuid.UUID
}

// GovernedCommandProposal is the exact, immutable proposal Governance
// evaluates — it contains no next state, no events, no approval outcome,
// and no ExecutionIntent. See docs/domain/command.md.
type GovernedCommandProposal struct {
	ProposalID            uuid.UUID
	SchemaVersion         int
	ProposalDigest        string
	CommandID             uuid.UUID
	CommandDigest         string
	Action                string
	ResourceType          string
	ResourceID            string
	OrganizationID        uuid.UUID
	ExpectedVersion       *int64
	Arguments             map[string]any
	TrustedContextDigest  string
	EffectClassification  string
	ExpiresAt             time.Time
}

// KernelDecisionEnvelope is the Kernel's final decision for one proposal:
// either stable rejection reasons, or the accepted next state, events, and
// optional ExecutionIntent. See docs/domain/command.md.
type KernelDecisionEnvelope struct {
	DecisionID           uuid.UUID
	SchemaVersion        int
	ProposalID            uuid.UUID
	ProposalDigest        string
	OrganizationID        uuid.UUID
	AggregateID           uuid.UUID
	PriorVersion          *int64
	Accepted              bool
	RejectionReasons      []string
	NextState             workflow.State
	NextVersion           int64
	Events                []event.DomainEvent
	Intent                *workflow.ExecutionIntent
	GovernanceDecisionID  uuid.UUID
	ApprovalID            *uuid.UUID
	DecidedAt             time.Time
	CorrelationID         uuid.UUID
	CausationID           *uuid.UUID
}

// PendingCommand tracks a command awaiting REQUIRE_APPROVAL resolution.
// Unexercised by this slice's always-ALLOW policy, implemented per
// application.md's "never bypassed." See docs/domain/command.md.
type PendingCommand struct {
	PendingCommandID        uuid.UUID
	OrganizationID           uuid.UUID
	Status                   string
	CommandType              CommandType
	ProposalDigest           string
	GovernedPayload          []byte
	ExpectedWorkflowID       *uuid.UUID
	ExpectedWorkflowVersion  *int64
	GovernanceDecisionID     uuid.UUID
	ApprovalID               *uuid.UUID
	RequestID                uuid.UUID
	IdempotencyKey           string
	CorrelationID            uuid.UUID
	CreatedAt                time.Time
	ExpiresAt                *time.Time
	ClosedAt                 *time.Time
	ClosureReason            *string
}

// RejectionReasons is the fixed, stable rejection-reason vocabulary for
// this slice (first-slice plan decision #11).
const (
	ReasonOrganizationInactive  = "ORGANIZATION_INACTIVE"
	ReasonObjectiveNotApproved  = "OBJECTIVE_NOT_APPROVED"
	ReasonDefinitionInactive    = "DEFINITION_INACTIVE"
	ReasonCapabilityInactive    = "CAPABILITY_INACTIVE"
	ReasonWorkflowAlreadyExists = "WORKFLOW_ALREADY_EXISTS"
	ReasonWorkflowNotFound      = "WORKFLOW_NOT_FOUND"
	ReasonVersionMismatch       = "VERSION_MISMATCH"
	ReasonIllegalState          = "ILLEGAL_STATE"
	ReasonResultOutcomeMismatch = "RESULT_OUTCOME_MISMATCH"
	ReasonResultAlreadyDecided  = "RESULT_ALREADY_DECIDED"
	ReasonResultBindingMismatch = "RESULT_BINDING_MISMATCH"
	ReasonGovernanceDenied      = "GOVERNANCE_DENIED"
	ReasonGovernanceStale       = "GOVERNANCE_STALE"
)
