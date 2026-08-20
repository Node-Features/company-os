// Package httpapi is the thin HTTP adapter companyd exposes today: a single
// health endpoint. It performs no governed decisions — see
// docs/architecture/application.md.
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

type HealthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

// HealthHandler reports process liveness plus database reachability. Pass a
// nil DBPinger when no DATABASE_URL is configured — the response then reports
// db: "not_configured" instead of attempting a connection.
func HealthHandler(db DBPinger) http.HandlerFunc {
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

		w.Header().Set("Content-Type", "application/json")
		if resp.Status != "ok" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(resp)
	}
}
