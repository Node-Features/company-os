package supabase

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/google/uuid"
)

// PrincipalRepository implements ports.PrincipalRepository.
type PrincipalRepository struct{ p *Pool }

func NewPrincipalRepository(p *Pool) *PrincipalRepository { return &PrincipalRepository{p: p} }

// FindOrCreateHumanBinding resolves evidence's (Issuer, Subject) to a
// durable HumanPrincipal bound to organizationID (ROADMAP.md Phase 3 Slice
// 6's onboarding flow). The upsert's ON CONFLICT ... DO UPDATE SET
// display_name = principals.display_name is a deliberate no-op — it exists
// only so RETURNING fires and hands back the existing row on a replay,
// without touching a display name the Principal might have acquired since
// (this slice never edits it after creation, but the mechanism shouldn't
// assume that stays true).
func (r *PrincipalRepository) FindOrCreateHumanBinding(ctx context.Context, organizationID uuid.UUID, evidence principal.AuthenticatedEvidence) (principal.Principal, error) {
	tx, err := r.p.pool.Begin(ctx)
	if err != nil {
		return principal.Principal{}, err
	}
	defer tx.Rollback(ctx)

	displayName := evidence.Subject
	if evidence.Email != nil && *evidence.Email != "" {
		displayName = *evidence.Email
	}

	var result principal.Principal
	var kind string
	err = tx.QueryRow(ctx, `
		INSERT INTO principals (principal_id, organization_id, kind, display_name, external_issuer, external_subject)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (external_issuer, external_subject)
		DO UPDATE SET display_name = principals.display_name
		RETURNING principal_id, organization_id, kind, display_name`,
		uuid.New(), organizationID, string(principal.KindHuman), displayName, evidence.Issuer, evidence.Subject,
	).Scan(&result.PrincipalID, &result.OrganizationID, &kind, &result.DisplayName)
	if err != nil {
		return principal.Principal{}, err
	}
	result.Kind = principal.Kind(kind)

	if _, err := tx.Exec(ctx, `
		INSERT INTO principal_organization_bindings (binding_id, principal_id, organization_id, status)
		VALUES ($1,$2,$3,'ACTIVE')
		ON CONFLICT (principal_id, organization_id) DO NOTHING`,
		uuid.New(), result.PrincipalID, organizationID); err != nil {
		return principal.Principal{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return principal.Principal{}, err
	}
	return result, nil
}
