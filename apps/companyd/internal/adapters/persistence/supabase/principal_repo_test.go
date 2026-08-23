package supabase

import (
	"context"
	"testing"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/google/uuid"
)

// testEvidence builds a fabricated principal.AuthenticatedEvidence for a
// fresh (issuer, subject) pair — bypassing the real Supabase Auth boundary
// (no live signed token needed) to prove FindOrCreateHumanBinding's
// mechanism against the real database, the same pattern
// internal/application/integration_test.go's submitFakeResult uses for
// Results.
func testEvidence() principal.AuthenticatedEvidence {
	email := "test-" + uuid.New().String() + "@example.test"
	return principal.AuthenticatedEvidence{
		SchemaVersion: 1,
		PrincipalType: principal.KindHuman,
		Issuer:        "https://test-issuer.example/auth/v1",
		Subject:       uuid.New().String(),
		Email:         &email,
		Method:        "supabase_jwt",
	}
}

// TestPrincipalRepository_FindOrCreateHumanBinding_IdempotentOnReplay
// proves ROADMAP.md Phase 3 Slice 6's onboarding flow against the real
// database: replaying the same evidence (as a second sign-in would) must
// resolve to the same durable Principal, not create a second one — and
// must leave exactly one principals row and one binding row behind.
func TestPrincipalRepository_FindOrCreateHumanBinding_IdempotentOnReplay(t *testing.T) {
	pool := requirePool(t)
	repo := NewPrincipalRepository(pool)
	ctx := context.Background()
	evidence := testEvidence()

	first, err := repo.FindOrCreateHumanBinding(ctx, fixtures.OrganizationID, evidence)
	if err != nil {
		t.Fatalf("first FindOrCreateHumanBinding: %v", err)
	}
	if first.Kind != principal.KindHuman {
		t.Fatalf("Kind = %s, want HUMAN", first.Kind)
	}
	if first.DisplayName != *evidence.Email {
		t.Fatalf("DisplayName = %q, want %q", first.DisplayName, *evidence.Email)
	}

	second, err := repo.FindOrCreateHumanBinding(ctx, fixtures.OrganizationID, evidence)
	if err != nil {
		t.Fatalf("second FindOrCreateHumanBinding: %v", err)
	}
	if second.PrincipalID != first.PrincipalID {
		t.Fatalf("replayed evidence resolved to a different PrincipalID: first=%s second=%s", first.PrincipalID, second.PrincipalID)
	}

	var principalCount int
	if err := pool.pool.QueryRow(ctx, `SELECT count(*) FROM principals WHERE external_issuer=$1 AND external_subject=$2`,
		evidence.Issuer, evidence.Subject).Scan(&principalCount); err != nil {
		t.Fatalf("count principals: %v", err)
	}
	if principalCount != 1 {
		t.Fatalf("principals rows for this (issuer, subject) = %d, want exactly 1", principalCount)
	}

	var bindingCount int
	if err := pool.pool.QueryRow(ctx, `SELECT count(*) FROM principal_organization_bindings WHERE principal_id=$1 AND organization_id=$2`,
		first.PrincipalID, fixtures.OrganizationID).Scan(&bindingCount); err != nil {
		t.Fatalf("count bindings: %v", err)
	}
	if bindingCount != 1 {
		t.Fatalf("binding rows for this Principal = %d, want exactly 1", bindingCount)
	}
}

// TestPrincipalRepository_FindOrCreateHumanBinding_DistinctSubjectsGetDistinctPrincipals
// proves onboarding doesn't collapse two different real people into one
// Principal.
func TestPrincipalRepository_FindOrCreateHumanBinding_DistinctSubjectsGetDistinctPrincipals(t *testing.T) {
	pool := requirePool(t)
	repo := NewPrincipalRepository(pool)
	ctx := context.Background()

	a, err := repo.FindOrCreateHumanBinding(ctx, fixtures.OrganizationID, testEvidence())
	if err != nil {
		t.Fatalf("FindOrCreateHumanBinding (a): %v", err)
	}
	b, err := repo.FindOrCreateHumanBinding(ctx, fixtures.OrganizationID, testEvidence())
	if err != nil {
		t.Fatalf("FindOrCreateHumanBinding (b): %v", err)
	}
	if a.PrincipalID == b.PrincipalID {
		t.Fatal("two distinct (issuer, subject) evidences resolved to the same PrincipalID")
	}
}
