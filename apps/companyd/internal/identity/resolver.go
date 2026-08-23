package identity

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// Resolver owns durable Principal resolution (docs/architecture/identity.md)
// against the one real Organization (ROADMAP.md Phase 3 Slice 6 — a
// single-organization deployment).
type Resolver struct {
	Principals     ports.PrincipalRepository
	OrganizationID uuid.UUID
}

func NewResolver(principals ports.PrincipalRepository, organizationID uuid.UUID) *Resolver {
	return &Resolver{Principals: principals, OrganizationID: organizationID}
}

// ResolveHuman resolves verified Human authentication evidence to a durable
// Principal, creating it (and its PrincipalOrganizationBinding) on first
// sign-in. Not consumed by Application this slice — see ROADMAP.md Phase 3
// Slice 6's scope note.
func (res *Resolver) ResolveHuman(ctx context.Context, evidence principal.AuthenticatedEvidence) (principal.Principal, error) {
	return res.Principals.FindOrCreateHumanBinding(ctx, res.OrganizationID, evidence)
}
