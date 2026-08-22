// Package realtime holds Sweeper, the poll loop that drives
// event_outbox rows to Supabase Realtime — the best-effort direct path
// web's browser client subscribes to (internal/ports/publisher.go). Sweeper
// reads internal/ports.OutboxRepository and calls internal/ports.Publisher;
// nothing did before Phase 1 Slice 8. The Publisher implementation itself
// (internal/adapters/persistence/supabase.RealtimePublisher) lives in the
// persistence package, not here — it publishes via Supabase's "Broadcast
// from Database" realtime.send() SQL function over the same DATABASE_URL
// pool every other repository uses, not an HTTP call, so it belongs with
// the other Postgres-backed adapters. It broadcasts (not postgres_changes:
// that would need an RLS SELECT policy this slice deliberately doesn't add,
// per ROADMAP.md Phase 9 Slice 4). See ADR-0004 item 4 and
// docs/architecture/events.md.
package realtime
