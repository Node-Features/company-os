package principal

import (
	"time"

	"github.com/google/uuid"
)

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
// AuthenticatedPrincipalEvidence, still used by governance.Request's
// EvidencePresent-style checks that don't yet consume a real verified
// evidence value. AuthenticatedEvidence (below) is the real thing, for
// callers that have actually verified a token.
type Evidence struct {
	PrincipalID uuid.UUID
	Verified    bool
	Source      string
}

// AuthenticatedEvidence is the normalized result of successfully verifying
// external authentication evidence for a HumanPrincipal — this slice's
// first real implementation of docs/architecture/identity.md's
// AuthenticatedPrincipalEvidence (ROADMAP.md Phase 3 Slice 3), for the
// Human flow only (Agent/Service/Provider flows are deferred, per the
// roadmap's own scope note).
//
// Deliberately narrower than identity.md's full envelope: everything
// below is a real, cryptographically verified field. Out of scope this
// slice, and why: session identity (no AuthenticationSession store yet);
// ClaimAttestation records (no separate attestation persistence yet);
// assurance classification (no assurance policy defined yet);
// nonce/replay-protection (no session/nonce infra yet); and durable
// PrincipalID/PrincipalOrganizationBinding (no persisted Principal or
// binding store exists at all — docs/domain/principal.md's
// PrincipalOrganizationBinding has no table yet; ROADMAP.md Phase 3 Slice
// 6 is the dedicated later slice for real Organization/Principal
// persistence). This type's job ends at producing trustworthy, normalized
// claims — resolving Subject to a durable Principal is whichever later
// slice has somewhere real to resolve it to.
type AuthenticatedEvidence struct {
	SchemaVersion int
	PrincipalType Kind // always KindHuman this slice

	Issuer   string  // verified JWT "iss"
	Subject  string  // verified JWT "sub" — the issuer's durable user ID; an opaque issuer-subject reference per identity.md, not (yet) a CompanyOS PrincipalID
	Email    *string // verified JWT "email", when present
	Audience string  // verified JWT "aud"
	Method   string  // e.g. "supabase_jwt"

	IssuedAt   time.Time
	ExpiresAt  time.Time
	VerifiedAt time.Time // when this adapter verified it, not when the issuer issued it

	VerifierAdapter string // e.g. "supabaseauth"
}
