package supabase

import (
	"context"
	"errors"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/organization"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// OrganizationRepository implements ports.OrganizationRepository.
type OrganizationRepository struct{ p *Pool }

func NewOrganizationRepository(p *Pool) *OrganizationRepository { return &OrganizationRepository{p: p} }

func (r *OrganizationRepository) GetOrganization(ctx context.Context, organizationID uuid.UUID) (organization.Organization, error) {
	var org organization.Organization
	err := r.p.pool.QueryRow(ctx, `
		SELECT organization_id, name, status FROM organizations WHERE organization_id=$1`,
		organizationID,
	).Scan(&org.OrganizationID, &org.Name, &org.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return organization.Organization{}, ports.ErrNotFound
	}
	if err != nil {
		return organization.Organization{}, err
	}
	return org, nil
}
