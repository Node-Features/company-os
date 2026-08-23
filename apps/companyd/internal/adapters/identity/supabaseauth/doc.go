// Package supabaseauth verifies Supabase Auth JWTs and normalizes them
// into principal.AuthenticatedEvidence. It is one ports.Authenticator
// adapter, implementing the Human flow only (ROADMAP.md Phase 3 Slice 3) —
// Supabase's token format (claim names, signing scheme) never leaks past
// this package boundary. Wiring Verifier into any live HTTP request path
// is Phase 3 Slice 4's job, not this package's.
// See docs/architecture/identity.md and ADR-0004 item 6.
package supabaseauth
