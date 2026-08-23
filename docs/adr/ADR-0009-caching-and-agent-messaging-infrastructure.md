# ADR-0009: Caching and Agent-Messaging Infrastructure

Status: APPROVED (2026-08-23, project owner `Node-Features`)

## Context

`companyd` today does everything synchronously against Postgres, per `ADR-0004`: every governed
action reads and writes Postgres directly, `Runtime`'s dispatch loop polls Postgres, and the
realtime layer (`ROADMAP.md` Phase 1 Slice 8) polls `event_outbox` into Supabase Realtime
Broadcast. No cache layer exists anywhere in this codebase today.

Two related needs surfaced directly from planning `docs/architecture/ui-ux.md` (Phase 10) and the
unbuilt Agent domain (`docs/domain/agent.md`, `APPROVED`, `ROADMAP.md` Phase 11):

1. Read paths that get hot once a real product UI exists — Governance policy lookups (especially
   if policy becomes persisted data instead of hardcoded Go, an open item this ADR does not
   resolve), `KnowledgeItem` retrieval, and intelligence-provider health/cooldown state — hit
   Postgres on every request with zero caching.
2. Once `Agent` records exist and are expected to communicate with each other to get work done,
   they need an asynchronous coordination mechanism that doesn't require every interaction to be a
   synchronous Postgres write-then-poll cycle.

The project owner asked specifically for Redis and QStash.

## Decision

Split into two independent adoptions — they solve different problems and have different
readiness here.

### Redis — cache layer, adopted

Redis is adopted strictly as a **disposable, rebuildable cache**, never an authoritative store.
This is not a new invariant this ADR introduces; `persistence.md`'s own invariant list states it
directly: "Conversation, cache, search index, vector index, provider state, checkpoint, and
message history are never authoritative business state" — Redis is the first concrete cache
*technology* adopted under a rule that already existed and already named "cache" and "provider
state" specifically. `knowledge.md`'s "disposable projections" language for Knowledge specifically
is the same rule applied to one record class, not a second, separate rule.

**First concrete cache path, decided: intelligence-provider fallback/cooldown state.**
`fallback.Provider` already tracks this, but only in one process's memory — it doesn't survive a
restart and can't be shared across more than one `companyd` process, which is an operational
correctness gap today, not merely a latency optimization the way the other candidates
(Governance policy lookups once persisted; `GetKnowledgeItem`/`QueryKnowledge` reads) are. Chosen
as the first path specifically because it's the simplest to reason about: **TTL-based
invalidation** (the cooldown already has a natural expiry — `providerCooldown = 60 * time.Second`
in `cmd/companyd/main.go` today — Redis just needs to hold that same TTL), and a **fail-open
cold-cache fallback**: a Redis miss or outage means "assume no active cooldown, the provider is
eligible to try" — degrading to the exact behavior this codebase has today (in-memory-only, so
effectively no cross-process cooldown sharing), never blocking dispatch. The Governance-policy and
`KnowledgeItem` candidates remain real future work but are deliberately not decided here — each has
a harder invalidation story (write-through on every policy edit; consistency with the
approve/reject transition) that deserves its own review, not inherited from this ADR by default.

Every cached read must define its invalidation trigger and a cold-cache fallback to Postgres
before it ships — a Redis outage must degrade latency, never correctness or availability.

### QStash — `web`-side only, decided

Kept separate from Redis deliberately, because its fit is not as clean and this ADR should say so
rather than rubber-stamp the original request. QStash's value proposition — durable HTTP callbacks
without a long-running worker — is strongest for serverless/edge deployments. `companyd` is
explicitly not that: `ADR-0004` chose one long-running Go process with its own
Postgres-outbox-polling worker (`Daemon`-supervised `Runtime`, and separately the realtime
`Sweeper`), which already solves "reliable async dispatch" once, inside that process. A second
async mechanism for the same class of problem inside `companyd` would be net-new complexity for no
clear win.

**Decided (project owner, 2026-08-23): QStash is scoped to `web` only.** `web` already runs on
Vercel (serverless, `ADR-0004`) — QStash's actual fit, not a workaround. It is adopted for
scheduled/webhook-driven work on that side (candidate first use: Phase 10's UI surface, as a
concrete need arises), tied to a real slice rather than adopted speculatively. `companyd` keeps its
existing Postgres-outbox pattern unchanged — this decision does not introduce a second async
mechanism there.

Agent-to-Agent messaging (the original motivating need, ROADMAP.md Phase 11 Slice 4) is explicitly
**not** decided by this ADR to run on QStash. That remains deferred until `node.md`'s multi-process
question resolves — building cross-process Agent messaging on a queuing technology before the
architecture has decided whether CompanyOS is single- or multi-process would still be premature
commitment, regardless of QStash's now-decided `web` role. Phase 11 Slice 4 may land on a narrower
single-process interim mechanism (e.g. the existing Postgres-outbox pattern, reused) if the Node
question is still open when that slice starts.

## Consequences

### Positive

- Read latency drops on hot paths without changing Postgres's role as the sole source of truth.
- Provider cooldown state becomes shareable across processes instead of silently resetting on
  every restart or redeploy.
- A messaging primitive is identified for future Agent-to-Agent work without over-committing to it
  before the Node question is resolved.

### Costs and risks

- A new infrastructure dependency and failure mode: Redis unavailability must degrade gracefully,
  not error — this has to be designed into the first cache, not retrofitted later.
- QStash stays intentionally out of `companyd` — using it there would duplicate the Postgres-outbox
  pattern that already works, so this decision leaves that boundary explicit rather than implicit.
- Cache invalidation correctness is a recurring, easy-to-get-wrong source of bugs; each cached read
  needs its own reviewed invalidation story, not a blanket "cache everything."

## Alternatives rejected by this proposal

- An in-Postgres cache (a materialized view or `UNLOGGED` table) for hot reads — rejected as the
  first move because it still costs a Postgres round trip; Redis is chosen specifically to avoid
  that, not because in-Postgres caching techniques are wrong in general.
- Building Agent-to-Agent messaging directly on QStash now — rejected until `node.md` resolves
  single- vs. multi-process; committing to a cross-process messaging technology before deciding
  cross-process execution exists is architecturally premature.
- A Postgres-backed job/message table for Agent messaging, mirroring the existing outbox pattern —
  **not rejected, genuinely still open**; this ADR does not conclude QStash is definitively better,
  only that the choice is made once the Node question is resolved, not before.

## Acceptance criteria

All met:

1. ✅ Concrete cache-invalidation design for the first adopted cache path (provider cooldown state:
   TTL-based, fail-open cold-cache fallback — see "Redis" above).
2. ✅ Explicit decision: QStash starts on `web` only, not `companyd` (project owner, 2026-08-23).
3. ✅ Confirmed against the real text of `persistence.md`'s invariant list (quoted directly above,
   not paraphrased) and `knowledge.md`'s disposable-projection language — no contradiction.
4. ✅ Project owner sign-off, 2026-08-23 (this review).

## Open questions

Remaining, genuinely unresolved — not blocking acceptance, carried forward:

- OPEN QUESTION: Self-hosted Redis vs. a managed provider (e.g. Upstash Redis, which pairs
  naturally with Upstash QStash and fits the already-Vercel-hosted `web` deployment) — not decided
  here.
- OPEN QUESTION: Who authorizes an Agent-to-Agent message — does it go through Governance like
  every other governed action, or is it non-authoritative "Agent memory" per `agent.md`'s own
  invariant ("Agent messages and memory are non-authoritative unless accepted by the owning domain
  operation")? This ADR decides `web`-only QStash scope; it does not decide Agent-messaging
  authorization or transport.
- OPEN QUESTION: When the Governance-policy-lookup and `KnowledgeItem`-read cache paths are
  eventually built, each needs its own invalidation design reviewed — not inherited from the
  provider-cooldown path's TTL approach by default, since their correctness requirements differ.

## Dependencies

- [ADR-0004: First-Slice Technology Stack](ADR-0004-first-slice-technology-stack.md)
- [Persistence](../architecture/persistence.md) — "cache... provider state... never authoritative business state," the invariant Redis is adopted under
- [Knowledge](../architecture/knowledge.md) — disposable-projection precedent, same rule applied to one record class
- [Node](../architecture/node.md) — the multi-process question Agent-to-Agent messaging (not `web`'s QStash use, which this ADR decides directly) still depends on
- [Agent](../domain/agent.md) — the eventual consumer of Agent-to-Agent messaging
- [Intelligence](../architecture/intelligence.md) — `fallback.Provider`'s existing in-memory
  cooldown state, the decided first cache path
