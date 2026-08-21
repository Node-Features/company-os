package approval

import (
	"time"

	"github.com/google/uuid"
)

// Status is the Approval lifecycle. See docs/domain/approval.md. Unexercised
// by this slice's always-ALLOW policy (application.md's REQUIRE_APPROVAL
// path is implemented but not reachable from the first-slice policy set).
type Status string

const (
	StatusPending   Status = "PENDING"
	StatusApproved  Status = "APPROVED"
	StatusConsumed  Status = "CONSUMED"
	StatusRejected  Status = "REJECTED"
	StatusExpired   Status = "EXPIRED"
	StatusCancelled Status = "CANCELLED"
	StatusRevoked   Status = "REVOKED"
)

type Approval struct {
	ApprovalID           uuid.UUID
	OrganizationID       uuid.UUID
	PendingCommandID     uuid.UUID
	RequestingPrincipalID uuid.UUID
	Action               string
	ResourceType         string
	ResourceID           string
	ProposalDigest       string
	Status               Status
	DecidedByPrincipalID *uuid.UUID
	DecidedAt            *time.Time
	Reason               *string
	CreatedAt            time.Time
}
