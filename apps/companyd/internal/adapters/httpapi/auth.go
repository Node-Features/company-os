package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
)

// evidenceContextKey is unexported so only this package can populate the
// request context — callers read it via EvidenceFromContext.
type evidenceContextKey struct{}

// EvidenceFromContext returns the AuthenticatedEvidence RequireHumanAuth
// verified for this request, if any. Not consumed by Application this
// slice (ROADMAP.md Phase 3 Slice 4's scope note) — available here for
// logging/future slices.
func EvidenceFromContext(ctx context.Context) (principal.AuthenticatedEvidence, bool) {
	ev, ok := ctx.Value(evidenceContextKey{}).(principal.AuthenticatedEvidence)
	return ev, ok
}

// RequireHumanAuth wraps next so it only runs once rawToken from the
// request's "Authorization: Bearer <token>" header has been verified by
// authn. Every governed HTTP route uses this — companyd never trusts a
// client-asserted Principal (docs/architecture/identity.md,
// ROADMAP.md Phase 3 Slice 4). Verified evidence is attached to the
// request context for next via EvidenceFromContext; Application's choice
// of acting Principal is unchanged this slice (still fixtures.Registry).
func RequireHumanAuth(authn ports.Authenticator, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawToken, ok := bearerToken(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "missing or malformed Authorization header")
			return
		}

		evidence, err := authn.VerifyHumanToken(r.Context(), rawToken)
		if err != nil {
			writeError(w, statusForAuthError(err), "authentication failed")
			return
		}

		ctx := context.WithValue(r.Context(), evidenceContextKey{}, evidence)
		next(w, r.WithContext(ctx))
	}
}

func bearerToken(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", false
	}
	return token, true
}

// statusForAuthError maps ports' normalized authentication failure
// sentinels onto HTTP status codes. Every case but ErrVerifierUnavailable
// is the caller's fault (401); an unavailable verifier is companyd's own
// dependency failing (503), matching statusCodeFor's existing
// caller-fault-vs-server-fault split in workflows.go.
func statusForAuthError(err error) int {
	if errors.Is(err, ports.ErrVerifierUnavailable) {
		return http.StatusServiceUnavailable
	}
	return http.StatusUnauthorized
}
