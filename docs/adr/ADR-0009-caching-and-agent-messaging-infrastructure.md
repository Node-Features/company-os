# ADR-0009: Caching and Agent-Messaging Infrastructure

Status: PROPOSED

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
This is not a new invariant this ADR introduces; it is the same rule `persistence.md` and
`knowledge.md` already state for every projection in this system ("Search, embeddings, graphs,
summaries, and caches are disposable projections... authoritative status is loaded from its
repository" — `knowledge.md`). Redis is the first concrete cache *technology* adopted under a rule
that already existed.

First candidate hot paths (not committed to exhaustively — the first implementation slice picks
one, real design not invented here): Governance policy-rule lookups (once persisted — see the open
question below), intelligence-provider fallback/cooldown state (`fallback.Provider` already tracks
this, but only in one process's memory — it doesn't survive a restart and can't be shared across
more than one `companyd` process), and `GetKnowledgeItem`/`QueryKnowledge` reads.

Every cached read must define its invalidation trigger and a cold-cache fallback to Postgres
before it ships — a Redis outage must degrade latency, never correctness or availability. This is
an acceptance criterion (below), not left to each implementer to improvise per-cache.

### QStash — async agent messaging, proposed but not committed

Kept separate deliberately, because its fit is not as clean as Redis's and this ADR should say so
rather than rubber-stamp the request. QStash's value proposition — durable HTTP callbacks without
a long-running worker — is strongest for serverless/edge deployments. `companyd` is explicitly not
that: `ADR-0004` chose one long-running Go process with its own Postgres-outbox-polling worker
(`Daemon`-supervised `Runtime`, and separately the realtime `Sweeper`), which already solves
"reliable async dispatch" once, inside that process.

QStash's clearest fit today is on the `web` side (Vercel, already serverless) for
scheduled/webhook-driven work — not as a replacement for `companyd`'s existing dispatch loop. Its
stronger long-term case is Agent-to-Agent messaging once more than one `companyd`-equivalent
process exists (`docs/architecture/node.md`, currently `DRAFT`, with its own open question — "does
the first slice need multiple Nodes at all" — genuinely unresolved).

Decision: adopt QStash for `web`-side scheduled/webhook work if and when a Phase 10 slice actually
needs it. Defer any Agent-to-Agent messaging use of it until `node.md`'s multi-process question is
resolved — building Agent messaging on a cross-process queuing technology before the architecture
has decided whether CompanyOS is single- or multi-process is premature commitment.

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
- Using QStash inside `companyd` itself (not just `web`) would duplicate the Postgres-outbox
  pattern that already works there — a second async mechanism for the same class of problem is
  complexity without a clear win until multi-Node is real.
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

Not yet met — this ADR is `PROPOSED`. Acceptance requires:

1. A concrete cache-invalidation design for at least the first adopted cache path.
2. An explicit decision on whether QStash usage starts on `web`, `companyd`, both, or neither, tied
   to a real `ROADMAP.md` Phase 10/11 slice rather than adopted speculatively.
3. Confirmation this doesn't contradict `persistence.md`'s authoritative-store ownership or
   `knowledge.md`'s disposable-projection invariant.
4. Project owner sign-off per `docs/adr/README.md`'s acceptance process.

## Open questions

- OPEN QUESTION: What is the first concrete cache path, and its exact invalidation trigger (TTL,
  write-through, event-driven)?
- OPEN QUESTION: Does QStash get adopted at all before `node.md`'s multi-Node question is
  resolved, or does Agent messaging wait on that?
- OPEN QUESTION: Self-hosted Redis vs. a managed provider (e.g. Upstash Redis, which pairs
  naturally with Upstash QStash and fits the already-Vercel-hosted `web` deployment) — not decided
  here.
- OPEN QUESTION: Who authorizes an Agent-to-Agent message — does it go through Governance like
  every other governed action, or is it non-authoritative "Agent memory" per `agent.md`'s own
  invariant ("Agent messages and memory are non-authoritative unless accepted by the owning domain
  operation")? This ADR does not answer that; it only proposes the transport.

## Dependencies

- [ADR-0004: First-Slice Technology Stack](ADR-0004-first-slice-technology-stack.md)
- [Persistence](../architecture/persistence.md)
- [Knowledge](../architecture/knowledge.md) — disposable-projection precedent
- [Node](../architecture/node.md) — the multi-process question this ADR's QStash decision depends on
- [Agent](../domain/agent.md) — the eventual consumer of Agent-to-Agent messaging
- [Intelligence](../architecture/intelligence.md) — `fallback.Provider`'s existing in-memory
  cooldown state, the first candidate cache path
