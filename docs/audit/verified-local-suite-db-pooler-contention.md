# Verified: Full-Suite Local Hang Is Remote-Pooler Contention, Not a CI Risk

Status: VERIFIED (2026-08-25) — root-caused by direct investigation; no code change made, and none is warranted. Diagnostic record for the hardening milestone's Increment 1 (bounded time-box, per [`.claude` plan / this session](../../.companyos/agent-memory/current-state.md)).

## What was observed

A `go test ./... -count=1` run in this local dev environment hit the default 10-minute per-package timeout inside `internal/application`, stuck in `TestIntegration_ProposeObjective_RejectLeavesNoObjective` → `RunResourceEvaluation` → `GetPerformanceProfile` → `pgx.Conn.QueryRow`. The same test passes cleanly in isolation (`go test ./internal/application/ -run TestIntegration_ProposeObjective_RejectLeavesNoObjective`) in ~23s — a number that itself matches [`fixed-runtime-resilience.md`](fixed-runtime-resilience.md)'s independent isolated rerun of the identical test almost exactly (23.24s vs. 23.46s), reinforcing this is the same test, the same class of dependency, not a new code path.

## Root cause

Two facts combine:

1. **No `pgxpool.MaxConns` is configured anywhere** in this codebase. `internal/adapters/persistence/supabase/pool.go`'s `Connect` (the one function every production and test call site uses — `cmd/companyd/main.go`, `requireRealApp`, `runtime_test.go`, `sweeper_test.go`, plus a standalone raw `pgxpool.New` in `idempotency_race_integration_test.go`) calls `pgxpool.New(ctx, connString)` with no `Config.MaxConns` override, so pgx's own default applies per pool (`max(4, runtime.NumCPU())`).
2. **This local dev environment's `DATABASE_URL` points at Supabase's remote, capacity-limited session-mode pooler** (`aws-1-eu-west-1.pooler.supabase.com:5432`, confirmed via `.env`) — deliberately session-mode, not transaction-mode, since `pool.go`'s own doc comment explains `LISTEN`/`NOTIFY` (ADR-0004's notification-recovery path) requires it. A session-mode pooler enforces a hard, shared cap on total concurrent connections across every simultaneous client of that Supabase project.

`go test ./...` launches roughly 15-20 packages' test binaries as separate OS processes, largely concurrently (Go's default cross-package test parallelism). Each package that touches the real DB opens its own uncapped pool. Aggregate connection demand across all of them, at the same moment, against one remote, capacity-limited pooler, can exceed what the pooler grants — a query then blocks waiting for a connection slot rather than failing fast, for as long as it takes another process's pool to release one. This explains a genuine multi-minute stall on a trivial `SELECT`, which a transient network blip (this repo's previously-used catch-all explanation, e.g. `fixed-test-suite-nondeterminism.md`'s "sporadic wsarecv/connection-timeout errors... not reproduced... consistent with genuine network blips, not a logic bug") does not fully account for — those prior incidents typically fail fast (`wsasend`/`wsarecv` errors within seconds), not hang silently for 10+ minutes. This may in fact be the more precise mechanism behind some of that repo history's previously-unexplained "blips," not a new phenomenon.

`internal/application`'s own tests are not the bug: `requireRealApp` correctly closes its pool via `t.Cleanup(pool.Close)` after every test, so there's no leak *within* one package's sequential run — the contention is *across* concurrently-running packages, an artifact of how `go test ./...` schedules work, not of any one package's connection hygiene.

## Why this is not a CI risk

`ci.yml`'s `go-integration` job (`ci.yml:145-163`) — the job that actually runs `go test ./... -count=1 -v -timeout=15m` as a single whole-module command, the same shape that hung locally — connects to a **local, ephemeral `postgres:17` service container on the same GitHub Actions runner** (`localhost:5432`, `POSTGRES_DB: companyos_ci`), not the remote Supabase pooler. A loopback connection to a freshly-started container has: no pooler connection cap (plain Postgres `max_connections` defaults to 100, comfortably above realistic aggregate demand from this repo's package count at pgx's small per-pool default), and near-zero network latency (governance-decision round trips that took 2-6s each against the remote pooler in this local run would be sub-millisecond against a loopback container). CI's exposure to this specific failure mode is categorically different from, and much lower than, this local dev environment's.

## Why no code change was made

A real fix exists in principle (cap `MaxConns` explicitly, e.g. via `pgxpool.ParseConfig`+`NewWithConfig` in `pool.go`), but `Connect` is shared, unmodified, by the production daemon (`cmd/companyd/main.go`) and every test call site. Capping it low enough to matter for local full-suite runs against the remote pooler would either under-provision the production daemon's real concurrent-request connection needs, or require threading a second connection-pool-sizing concept through `Connect`'s callers — touching `main.go`, three sites in `runtime_test.go`, `integration_test.go`, `sweeper_test.go`, `idempotency_race_integration_test.go`, and `adapters/persistence/supabase/integration_test.go` — for a condition that (a) doesn't reproduce in CI and (b) doesn't threaten authoritative state (a slow/stuck test is not a corrupted Workflow). That blast radius is disproportionate to a local-dev-only test-runner characteristic; making it would be exactly the kind of unrelated refactor this milestone's own instructions warn against. **Advisory instead**: running `go test ./...` locally against the real dev DB can stall; run affected packages individually, or point `DATABASE_URL` at a local Postgres instance for full-suite local runs, to avoid remote-pooler contention.

## Dependencies

- [`findings.md`](findings.md), [`fixed-test-suite-nondeterminism.md`](fixed-test-suite-nondeterminism.md) — the prior "network blip" framing this refines with a concrete mechanism
- [`fixed-ci-integration-coverage.md`](fixed-ci-integration-coverage.md) — CI's actual job/service-container shape, confirmed not equivalently exposed
- [`docs/testing/concurrency-guarantees.md`](../testing/concurrency-guarantees.md)
