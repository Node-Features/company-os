package httpapi

import (
	"context"
	"net/http"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/Node-Features/company-os/apps/companyd/internal/identity"
	"github.com/Node-Features/company-os/apps/companyd/internal/observability"
)

// principalContextKey is unexported so only this package can populate the
// request context — callers read it via PrincipalFromContext.
type principalContextKey struct{}

// PrincipalFromContext returns the durable Principal ResolvePrincipal
// resolved for this request, if any. Not consumed by Application this
// slice (ROADMAP.md Phase 3 Slice 6's scope note) — available here for
// logging/future slices, the same deferred-consumption shape as Slice 4's
// EvidenceFromContext.
func PrincipalFromContext(ctx context.Context) (principal.Principal, bool) {
	p, ok := ctx.Value(principalContextKey{}).(principal.Principal)
	return p, ok
}

// ResolvePrincipal wraps next so it only runs once the AuthenticatedEvidence
// RequireHumanAuth already verified has been resolved to a durable
// Principal via resolver — this is the onboarding flow (ROADMAP.md Phase 3
// Slice 6) that creates a HumanPrincipal and its PrincipalOrganizationBinding
// on first sign-in. Must run after RequireHumanAuth, which is what
// populates the evidence this reads.
func ResolvePrincipal(resolver *identity.Resolver, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		evidence, ok := EvidenceFromContext(r.Context())
		if !ok {
			// Programmer error, not a caller fault: ResolvePrincipal is only
			// ever wired after RequireHumanAuth.
			observability.Logger(r.Context()).Error("httpapi: ResolvePrincipal called without prior RequireHumanAuth evidence")
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		resolved, err := resolver.ResolveHuman(r.Context(), evidence)
		if err != nil {
			observability.Logger(r.Context()).Error("httpapi: resolve principal failed", "error", err.Error())
			writeError(w, http.StatusServiceUnavailable, "principal resolution unavailable")
			return
		}

		ctx := context.WithValue(r.Context(), principalContextKey{}, resolved)
		ctx = observability.WithExecutionContext(ctx, observability.ExecutionContext{
			OrganizationID: resolved.OrganizationID,
			PrincipalID:    resolved.PrincipalID,
		})
		next(w, r.WithContext(ctx))
	}
}
