// Package supabaseauth verifies Supabase Auth JWTs and normalizes them
// into AuthenticatedPrincipalEvidence. It is one Authenticator adapter;
// Supabase's token format never leaks past this package boundary.
// See docs/architecture/identity.md and ADR-0004 item 6.
package supabaseauth
