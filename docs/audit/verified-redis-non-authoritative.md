# Verified: Redis Never Authoritative (Full-Repo Re-Audit)

Status: VERIFIED (2026-08-24)

Closes out a direct request to audit the repository against the invariant: Postgres is
authoritative organizational/workflow state; Redis, if used, must be strictly disposable
cache/coordination and must never become authoritative. This is the evidence record for that
audit. It supersedes nothing in [`findings.md`](findings.md) §1.0's Redis row or §5's
disposable-state inventory — it re-confirms both with broader scope and fresh commands, run in a
separate session from the 2026-08-24 audit that produced `findings.md`.

## Scope

`findings.md`'s original Redis check (§1.0, "Method" §) was `grep -rn redis -i` over
`apps/companyd` only, cited as the evidence for "Zero lines of code anywhere... Plan-only." This
pass widens that to the full repository and to every category named in the audit request:
Redis-backed workflow state, execution state, locks, queues, session state, caches,
recovery/startup assumptions, and code paths that could fail to recover from an empty Redis.

## Evidence

Commands run against the working tree at `HEAD` (`bcaaa45`), independent of and in addition to
`findings.md`'s own citation:

| Check | Command / target | Result |
|---|---|---|
| Go source | `grep -rn -i "redis\|ioredis" apps/companyd` (all `.go` files) | No matches |
| Go dependencies | `apps/companyd/go.mod`, `apps/companyd/go.sum` | No `redis` entry of any kind |
| TS/TSX source | `grep -rn -i "redis\|ioredis\|upstash"` over `apps/web` (excluding `node_modules`) | No matches |
| Node dependencies | `package.json` (root and `apps/web`) | No `redis`/`ioredis`/`@upstash/redis`/`bullmq` entry |
| Env config | `apps/companyd/.env`, `apps/companyd/.env.example`, `apps/web/.env.example`, `apps/web/.env.local` | No `REDIS_*` variable |
| CI | `.github/workflows/*` | No Redis service container, no `REDIS_URL` | 
| Deployment config | repo-wide search for `docker-compose*`/service manifests | None exist in this repo at all (Vercel + Supabase-managed Postgres per `ADR-0004`, no container orchestration file) |
| Supabase config | `supabase/config.toml` | No Redis reference |
| Session state | [`apps/web/lib/session.ts`](../../apps/web/lib/session.ts) | Supabase Auth session (`supabase.auth.getSession()`), not Redis-backed |
| In-process pub/sub (possible Redis substitute) | [`internal/concurrency/eventbus.go`](../../apps/companyd/internal/concurrency/eventbus.go) | Pure in-memory `chan`-based bus, explicitly documented best-effort/non-authoritative — already inventoried as disposable in `findings.md` §5, not Redis |
| Locks (possible Redis substitute) | [`internal/concurrency/safestate.go`](../../apps/companyd/internal/concurrency/safestate.go) | In-process `sync.Mutex`-guarded state, not distributed, not Redis |
| Queues (possible Redis substitute) | [`internal/runtime/scheduler.go`](../../apps/companyd/internal/runtime/scheduler.go), `execution_intents` table | Postgres `FOR UPDATE SKIP LOCKED` polling per `ADR-0004`/`ADR-0009`'s own description of the existing pattern, not Redis |

Conclusion: **zero Redis code, config, or dependency exists anywhere in this repository.**

## Classification (per the requested A–D schema)

Not applicable — there is nothing to classify. No Redis usage exists in any category (workflow
state, execution state, locks, queues, session state, caches, recovery assumptions).

The one Redis *reference* in the repository is a decision, not code:
[`ADR-0009`](../adr/ADR-0009-caching-and-agent-messaging-infrastructure.md) (APPROVED) names one
planned path — `ROADMAP.md` Phase 12 Slice 1, intelligence-provider cooldown state — not yet
implemented. As specified (TTL-based, fail-open: a cache miss or Redis outage means "assume no
active cooldown," degrading to today's actual in-memory-only behavior rather than blocking
dispatch), that design already targets classification **A — safe disposable cache** once built:
Postgres/in-process state remains authoritative, and Redis is consulted only as an optimization on
top of a code path that already works without it.

## Recovery scenarios (per the requested list)

All six trivially hold today, because no code path reads or depends on Redis in any way:

1. Redis restart — N/A, no dependency
2. Redis data loss — N/A, no dependency
3. Redis eviction — N/A, no dependency
4. Network partition to Redis — N/A, no dependency
5. Application restart — N/A, no dependency
6. Multiple runtime instances — N/A, no dependency (and orthogonal: `companyd` is explicitly
   single-process per `ADR-0004`; see `docs/architecture/node.md`'s still-open multi-process
   question, unrelated to Redis)

**DELETE ALL REDIS DATA acceptance criterion: currently vacuously satisfied.** There is no Redis
data for CompanyOS to lose, and all organizational/workflow state already lives exclusively in
Postgres (`findings.md` §5's authoritative-tables list). Nothing to refactor.

## Tests

None added. There is no Redis-touching code path to write a "wipe Redis, verify recovery" test
against. When `ROADMAP.md` Phase 12 Slice 1 is implemented, that slice must include a test proving
the provider-cooldown cache's fail-open behavior on an empty/unreachable Redis — tracked as a
requirement for that slice, not a gap in this one, since the code doesn't exist to test yet.

## Dependencies

- [`findings.md`](findings.md) §1.0 (Redis row), §5 (disposable-state inventory) — the original,
  narrower-scope finding this re-confirms
- [`ADR-0009: Caching and Agent-Messaging Infrastructure`](../adr/ADR-0009-caching-and-agent-messaging-infrastructure.md) — the only Redis decision in the repo, and the fail-open design this doc validates against the requested invariant
- `ROADMAP.md` Phase 12 Slice 1 — where the first real Redis code will land; re-run this check's
  command table against that slice's diff before merging it
