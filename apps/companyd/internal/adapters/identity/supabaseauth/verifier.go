package supabaseauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/golang-jwt/jwt/v5"
)

// authenticatedAudience is the "aud" claim Supabase Auth sets for a
// signed-in user's access token. Anonymous sign-ins use a different
// audience, but this project has anonymous auth disabled (confirmed
// against the live project), so this is the only audience Verify accepts.
const authenticatedAudience = "authenticated"

// verifierAdapterName is recorded on every AuthenticatedEvidence this
// package produces (principal.AuthenticatedEvidence.VerifierAdapter).
const verifierAdapterName = "supabaseauth"

// claims is Supabase Auth's JWT shape restricted to what this slice
// verifies and normalizes — Supabase's token format stays entirely
// inside this package, per ADR-0004 item 6.
type claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// Verifier implements ports.Authenticator for Supabase Auth's Human flow.
// See docs/architecture/identity.md and ADR-0004 item 6.
type Verifier struct {
	issuer string
	keys   keyfunc.Keyfunc
}

// New constructs a Verifier for the Supabase project at supabaseURL. It
// fetches that project's JWKS (published at
// {supabaseURL}/auth/v1/.well-known/jwks.json — public by design, no
// secret needed) and keeps it refreshed in the background for the
// lifetime of ctx, via keyfunc's own default refresh behavior.
func New(ctx context.Context, supabaseURL string) (*Verifier, error) {
	issuer := strings.TrimRight(supabaseURL, "/") + "/auth/v1"
	return newWithJWKSURL(ctx, issuer, issuer+"/.well-known/jwks.json")
}

// newWithJWKSURL is New's implementation with the JWKS URL separated out
// from the issuer it's normally derived from, so tests can point it at a
// local JWKS server without depending on a live Supabase project.
func newWithJWKSURL(ctx context.Context, issuer, jwksURL string) (*Verifier, error) {
	keys, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ports.ErrVerifierUnavailable, err)
	}
	return &Verifier{issuer: issuer, keys: keys}, nil
}

// VerifyHumanToken implements ports.Authenticator. It never returns a
// partial or ambiguous AuthenticatedEvidence — any failure is one of the
// ports sentinel errors, per identity.md's "unknown or indeterminate
// verification fails closed."
func (v *Verifier) VerifyHumanToken(ctx context.Context, rawToken string) (principal.AuthenticatedEvidence, error) {
	var c claims
	token, err := jwt.ParseWithClaims(rawToken, &c, v.keys.KeyfuncCtx(ctx),
		jwt.WithValidMethods([]string{"ES256"}), // pins the algorithm — defense against alg-confusion attacks
		jwt.WithIssuer(v.issuer),
		jwt.WithAudience(authenticatedAudience),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return principal.AuthenticatedEvidence{}, classifyError(err)
	}
	if !token.Valid {
		return principal.AuthenticatedEvidence{}, ports.ErrInvalidSignature
	}
	if c.Subject == "" {
		return principal.AuthenticatedEvidence{}, fmt.Errorf("%w: missing sub claim", ports.ErrMalformedToken)
	}

	var email *string
	if c.Email != "" {
		email = &c.Email
	}

	audience := ""
	if len(c.Audience) > 0 {
		audience = c.Audience[0]
	}

	return principal.AuthenticatedEvidence{
		SchemaVersion:   1,
		PrincipalType:   principal.KindHuman,
		Issuer:          c.Issuer,
		Subject:         c.Subject,
		Email:           email,
		Audience:        audience,
		Method:          "supabase_jwt",
		IssuedAt:        c.IssuedAt.Time,
		ExpiresAt:       c.ExpiresAt.Time,
		VerifiedAt:      time.Now().UTC(),
		VerifierAdapter: verifierAdapterName,
	}, nil
}

// classifyError maps golang-jwt's specific validation errors onto this
// package's normalized ports sentinels (identity.md's "normalized
// authentication failures"), falling back to ErrMalformedToken for
// anything this slice doesn't distinguish further.
func classifyError(err error) error {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return fmt.Errorf("%w: %s", ports.ErrExpiredToken, err)
	case errors.Is(err, jwt.ErrTokenInvalidIssuer):
		return fmt.Errorf("%w: %s", ports.ErrWrongIssuer, err)
	case errors.Is(err, jwt.ErrTokenInvalidAudience):
		return fmt.Errorf("%w: %s", ports.ErrWrongAudience, err)
	case errors.Is(err, jwt.ErrTokenSignatureInvalid):
		return fmt.Errorf("%w: %s", ports.ErrInvalidSignature, err)
	case errors.Is(err, keyfunc.ErrKeyfunc):
		return fmt.Errorf("%w: %s", ports.ErrVerifierUnavailable, err)
	default:
		return fmt.Errorf("%w: %s", ports.ErrMalformedToken, err)
	}
}

var _ ports.Authenticator = (*Verifier)(nil)
