package objective

import "github.com/google/uuid"

// Status is the Objective lifecycle status.
type Status string

const (
	StatusProposed Status = "PROPOSED"
	StatusApproved Status = "APPROVED"
	StatusRetired  Status = "RETIRED"
)

// Objective is the organizational goal a Workflow serves. See
// docs/domain/objective.md.
type Objective struct {
	ObjectiveID      uuid.UUID
	OrganizationID   uuid.UUID
	Title            string
	Status           Status
	OwnerPrincipalID uuid.UUID
}
