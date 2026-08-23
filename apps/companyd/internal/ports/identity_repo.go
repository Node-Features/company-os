package ports

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/organization"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/google/uuid"
)

// OrganizationRepository reads the one real, persisted Organization
// (ROADMAP.md Phase 3 Slice 6). GetOrganization returns ErrNotFound if no
// row exists — companyd fails closed rather than silently falling back to
// a hardcoded value.
type OrganizationRepository interface {
	GetOrganization(ctx context.Context, organizationID uuid.UUID) (organization.Organization, error)
}

// PrincipalRepository resolves authenticated evidence to a durable
// Principal (docs/architecture/identity.md: "Identity owns durable
// Principal resolution, organization bindings"). FindOrCreateHumanBinding
// is idempotent: replaying the same evidence's (Issuer, Subject) returns
// the same Principal, creating it (and its PrincipalOrganizationBinding)
// only on first sign-in.
type PrincipalRepository interface {
	FindOrCreateHumanBinding(ctx context.Context, organizationID uuid.UUID, evidence principal.AuthenticatedEvidence) (principal.Principal, error)
}
