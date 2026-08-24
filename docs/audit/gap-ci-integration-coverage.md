# Gap: CI Never Runs Database-Backed Integration Tests

Status: APPROVED (2026-08-24) — problem statement and remediation plan approved. Implementation still requires the project owner to explicitly select and authorize this slice before any code changes, per this repository's doc-gate convention.

Severity: P1 — threatens runtime reliability (ambient, compounding risk on every future change). See [`findings.md`](findings.md) §1 ("CI"), §3.

## Problem

[`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) runs `go vet ./...` and `go test ./...` for `companyd` with **no `DATABASE_URL` or `SUPABASE_URL` set**, and `npm install && npm run build` for `web` (no `npm run lint`, no test step despite a `lint` script existing). Every test gated by `requireRealApp(t)` — some 40+ call sites across `integration_test.go`, `finance_integration_test.go`, `knowledge_integration_test.go`, `me_integration_test.go`, `objective_integration_test.go`, `research_integration_test.go`, plus the persistence-layer and JWT-live tests — self-skips via `t.Skip("DATABASE_URL not set...")`.

What actually runs automatically on every push/PR: `go vet`, the narrow set of fake-backed unit tests (`pipeline_test.go`), Kernel/Governance pure-function unit tests, and a Next.js type-check/build. **All of the correctness verification this project's own implementation notes cite as proof of correctness — state-transition correctness, concurrency behavior, idempotency, transactional atomicity, the entire Phase 3-5 governed-approval pipeline — has only ever been run manually, locally, by whoever was writing the code that session.** A regression in any of it would not be caught by CI today.

This is not a new discovery in isolation — [`testing/strategy.md:36-42`](../testing/strategy.md) already names this exact gap and scopes its fix to `ROADMAP.md` Phase 8 Slice 1 (ephemeral Postgres/Supabase service in CI). This gap doc exists to flag it as higher-priority than its current unstarted Phase-8 position implies: every P0/P1 fix in this audit directory ships without automated regression protection until this lands, including the fixes for the gaps this same audit is proposing.

## Invariant

Restores: a green CI run as meaningful evidence of correctness, not just compilation and a narrow unit-test slice.

## Proposed approach (plan-level only)

This gap doc does not redefine the target shape — [`testing/strategy.md`](../testing/strategy.md)'s Phase 8 Slice 1/2/3 plan already fixes it: an ephemeral Postgres/Supabase service in CI, a contract-test job, then `web` e2e. The recommendation here is sequencing, not design: pull Phase 8 Slice 1 forward, ahead of or alongside the other P0/P1 gaps in this directory, rather than leaving it in its current unstarted position behind Phases 6/7/9.

## Files likely to change

- [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) — add a database service, set `DATABASE_URL`/`SUPABASE_URL` (or a local-JWKS test double for the latter — `verifier_live_test.go` already self-skips without a real `SUPABASE_URL`, and CI likely needs the same httptest-JWKS pattern `verifier_test.go` already uses rather than a live Supabase project).
- [`apps/companyd`](../../apps/companyd) migrations — `cmd/migrate`'s non-idempotent replay-every-file behavior (see [`backlog-p2-p4.md`](backlog-p2-p4.md#migration-hygiene)) needs to work against a fresh, empty CI database each run — this is actually the one environment that behavior already fits (fresh DB every time), so no `cmd/migrate` change should be strictly required, but confirm before relying on it.
- `apps/web/package.json` / `.github/workflows/ci.yml` — add the missing `npm run lint` CI step (small, unrelated fix noticed in passing).

## Tests required

**Before:** none needed — the "before" state is simply the current CI log, which already shows every `requireRealApp` test skipped.

**After:**
- CI run showing the full `requireRealApp` suite executing and passing against the CI-provisioned database, not skipping.
- CI run showing `npm run lint` executing.

## Dependencies

- [`testing/strategy.md`](../testing/strategy.md) — canonical target shape, already APPROVED; this gap doc adds urgency, not new design.
- [`testing/contract-tests.md`](../testing/contract-tests.md), [`testing/failure-injection.md`](../testing/failure-injection.md)
- [`ROADMAP.md`](../../ROADMAP.md) Phase 8.
- Every other gap doc in this directory depends on this one for a durable regression guarantee once fixed.
