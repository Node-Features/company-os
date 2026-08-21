package principal

import "github.com/google/uuid"

// Kind distinguishes the Principal categories named in
// docs/domain/principal.md. Only ServicePrincipal is populated this slice
// (see decision #5 in the first-slice plan); the others are named so
// Kernel/Governance code can already switch on Kind without a later
// breaking change.
type Kind string

const (
	KindHuman    Kind = "HUMAN"
	KindAgent    Kind = "AGENT"
	KindService  Kind = "SERVICE"
	KindProvider Kind = "PROVIDER"
)

// Principal is a durable actor identity. See docs/domain/principal.md.
type Principal struct {
	PrincipalID    uuid.UUID
	OrganizationID uuid.UUID
	Kind           Kind
	DisplayName    string
}

// Evidence stands in for docs/domain/identity.md's
// AuthenticatedPrincipalEvidence. Real Supabase-Auth JWT verification is
// deferred to Phase 2; this slice only checks a stub evidence marker is
// present (decision #5).
type Evidence struct {
	PrincipalID uuid.UUID
	Verified    bool
	Source      string
}
