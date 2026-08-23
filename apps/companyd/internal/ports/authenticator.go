// Authenticator and AttestationVerifier ports consumed by Identity.
// See docs/architecture/identity.md.
package ports

import (
	"context"
	"errors"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
)

// Authenticator verifies external authentication evidence and normalizes
// it into principal.AuthenticatedEvidence — one adapter per Principal
// type/method (docs/architecture/identity.md). This slice defines only
// the Human flow (ROADMAP.md Phase 3 Slice 3's scope note); Agent/
// Service/Provider authenticators are future work, added as their own
// methods here when they exist rather than a single overloaded one.
//
// A non-nil error is always one of the sentinels below (or wraps one) —
// never a partial/ambiguous AuthenticatedEvidence. identity.md:
// "Unknown or indeterminate verification fails closed."
type Authenticator interface {
	VerifyHumanToken(ctx context.Context, rawToken string) (principal.AuthenticatedEvidence, error)
}

// Normalized authentication failure classes (identity.md's "normalized
// authentication failures" list) — a scoped subset covering what a JWT
// verifier can actually distinguish; the full ~15-category list needs
// session/revocation infrastructure this slice doesn't build.
var (
	ErrMalformedToken    = errors.New("ports: malformed authentication token")
	ErrInvalidSignature  = errors.New("ports: invalid token signature")
	ErrExpiredToken      = errors.New("ports: expired token")
	ErrWrongIssuer       = errors.New("ports: untrusted token issuer")
	ErrWrongAudience     = errors.New("ports: wrong token audience")
	ErrVerifierUnavailable = errors.New("ports: authenticator verifier unavailable")
)
