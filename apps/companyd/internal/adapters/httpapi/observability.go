package httpapi

import (
	"net/http"

	"github.com/Node-Features/company-os/apps/companyd/internal/observability"
	"github.com/google/uuid"
)

// WithObservability wraps next so every request carries a fresh
// correlation identity (docs/architecture/observability.md) from the
// moment it enters companyd's HTTP layer, before RequireHumanAuth or
// ResolvePrincipal run — so even an unauthenticated request's rejection is
// logged with a correlation ID. ResolvePrincipal enriches the same
// ExecutionContext with PrincipalID once the caller is resolved; this
// middleware never overwrites fields a later layer adds (see
// ExecutionContext.With).
func WithObservability(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := observability.WithExecutionContext(r.Context(), observability.ExecutionContext{
			CorrelationID: uuid.New(),
		})
		next(w, r.WithContext(ctx))
	}
}
