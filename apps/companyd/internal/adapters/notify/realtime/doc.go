// Package realtime publishes post-commit notifications over Supabase
// Realtime (Postgres LISTEN/NOTIFY) as the best-effort direct path.
// The durable fallback (Runtime's polling sweep) lives in internal/runtime.
// See ADR-0004 item 4.
package realtime
