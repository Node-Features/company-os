package organization

import "github.com/google/uuid"

// Status is the Organization lifecycle status.
type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

// Organization is the isolation boundary every Workflow, Objective, and
// Principal is scoped to. See docs/domain/organization.md.
type Organization struct {
	OrganizationID uuid.UUID
	Name           string
	Status         Status
}
