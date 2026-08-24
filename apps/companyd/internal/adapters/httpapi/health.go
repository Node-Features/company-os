// Package httpapi is the thin HTTP adapter companyd exposes today. It
// performs no governed decisions — see docs/architecture/application.md.
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// DBPinger is satisfied by the persistence adapter. Defined here, on the
// consumer side, so this package never imports pgx directly.
type DBPinger interface {
	Ping(ctx context.Context) error
}

// Diagnostics is satisfied by internal/adapters/notify/realtime.Sweeper.
// Read-only, side-effect-free values (docs/architecture/daemon.md:
// "Health probes are side-effect free") — never authoritative, per
// docs/architecture/observability.md's non-authority invariant; a nil
// Diagnostics simply omits these fields from the response.
type Diagnostics interface {
	Diagnostics() (backlog int, lastReconcileAt time.Time)
}

type HealthResponse struct {
	Status          string `json:"status"`
	DB              string `json:"db"`
	OutboxBacklog   *int   `json:"outboxBacklog,omitempty"`
	LastReconcileAt string `json:"lastReconcileAt,omitempty"`
}

// HealthHandler reports process liveness, database reachability, and
// (when diag is non-nil) outbox-sweeper diagnostics. Pass a nil DBPinger
// when no DATABASE_URL is configured — the response then reports db:
// "not_configured" instead of attempting a connection. Pass a nil
// Diagnostics when the sweeper isn't running (same degraded-mode shape as
// a nil DBPinger) — the diagnostic fields are simply omitted.
func HealthHandler(db DBPinger, diag Diagnostics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := HealthResponse{Status: "ok"}

		switch {
		case db == nil:
			resp.DB = "not_configured"
		default:
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()
			if err := db.Ping(ctx); err != nil {
				resp.DB = "unreachable"
				resp.Status = "degraded"
			} else {
				resp.DB = "ok"
			}
		}

		if diag != nil {
			backlog, lastReconcileAt := diag.Diagnostics()
			resp.OutboxBacklog = &backlog
			if !lastReconcileAt.IsZero() {
				resp.LastReconcileAt = lastReconcileAt.UTC().Format(time.RFC3339)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if resp.Status != "ok" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(resp)
	}
}
