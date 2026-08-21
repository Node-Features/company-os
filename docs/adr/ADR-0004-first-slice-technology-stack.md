# ADR-0004: First-Slice Technology Stack and Deployment Topology

Status: APPROVED

## Context

`ARCHITECTURE.md`, `kernel.md`, `runtime.md`, `daemon.md`, `persistence.md`, and `identity.md` deliberately select no concrete technology, so that organizational semantics remain provider-independent. But no Phase 1 vertical slice can be built without picking concrete technology behind those ports. `.companyos/agent-memory/current-state.md` tracks four unresolved implementation choices: the first Runtime implementation, the persistence adapter and its state model, the notification-recovery mechanism, and the first `CapabilityDefinition`. The project owner has specified a starting stack — Next.js for UI and APIs, Supabase for persistence, Go for core services (Kernel, Daemon, Runtime) — and confirmed Identity and Governance are placed alongside them.

## Proposed decision

1. **Deployment topology.** One long-lived Go service (`companyd`) hosts Kernel, Application orchestration, Governance, Identity, Runtime, and Daemon as one process. These remain logically distinct modules per [ADR-0001](ADR-0001-kernel-runtime-daemon.md) — co-locating them in one process is a deployment choice, not a boundary change. Next.js hosts the UI and the HTTP-facing Application adapter: its API routes translate transport requests into `ApplicationRequest`s and forward them to `companyd`; they make no governed decision themselves. Next.js may read Supabase directly for non-authoritative UI projections only, per `persistence.md`'s allowance for stale, non-authoritative read paths.

   **Local development:** `companyd` runs via `go run`, with [`air`](https://github.com/air-verse/air) for hot-reload against local env vars pointing at Supabase (either the Supabase CLI's local Docker stack or a shared hosted dev project). Next.js runs via `next dev` against a local `companyd` port.

   **Production:** `companyd` runs on a small DigitalOcean VPS, supervised by systemd. This directly satisfies `daemon.md`'s own expectation that *"an OS service manager, container platform, or cloud scheduler may supervise the Daemon externally"* — systemd's restart/backoff policy is that external supervision layer, distinct from Runtime's own operation-level retry per `execution.md`. Next.js deploys to Vercel; Supabase remains a managed hosted project. `companyd`'s exact restart/backoff configuration and firewall/network exposure to Supabase remain implementation detail, not an architectural decision.
2. **Persistence adapter.** Supabase-hosted Postgres implements the Persistence ports using state-plus-outbox, not event sourcing — this is largely already implied by the approved port shapes: `AuthoritativeStateRepository` as versioned rows with compare-and-write; `EventRepository`/`Outbox` as an immutable events table plus an outbox table for post-commit publication; `ExecutionRepository` for Runtime's attempts, leases, checkpoints, waits, and retries per [`execution.md`](../domain/execution.md).
3. **Runtime implementation.** Internal, written in Go inside `companyd`, implementing the `execution.md` contract directly against the Supabase-backed `ExecutionRepository` — not an adapter over Temporal, Inngest, Trigger.dev, or another external workflow engine.
4. **Notification recovery.** Post-commit notification publishes over Supabase Realtime (Postgres `LISTEN`/`NOTIFY`) as the best-effort direct path. Runtime additionally runs a periodic polling sweep against Persistence for due, unclaimed `ExecutionIntent`s as the durable fallback, per `application.md`'s explicit allowance for "durable intent discovery, an outbox publisher, or both." No separate message broker is introduced.
5. **First `CapabilityDefinition`.** A minimal `IntelligenceCapability` for short, bounded text generation, satisfied by exactly one registered `ModelProfile` and `ProviderAdapter`. This proves the `ExecutionIntent` → dispatch → Result → `ACCEPT_WORKFLOW_RESULT`/`REJECT_WORKFLOW_RESULT` loop end-to-end without requiring Workspace isolation, Git, or coding-agent complexity.
6. **Identity adapter.** Supabase Auth is registered as one `Authenticator` adapter for `HumanPrincipal` sessions. Identity's Principal and session contracts remain canonical; Supabase Auth's token/session format does not leak past the adapter boundary, per the invariant that authentication adapters "are replaceable and cannot define CompanyOS Principal or authorization semantics."

## Consequences

### Positive

- Every port selected here already exists as an approved contract; this ADR fixes adapters, not architecture — no `docs/architecture/*.md` or `docs/domain/*.md` document requires a boundary change.
- Co-locating Kernel/Application/Governance/Identity/Runtime/Daemon in one Go process minimizes network hops in the tightest part of the system (the proposal-validation → Governance → final-decision sequence), while goroutines give Runtime cheap concurrent execution without a separate worker fleet.
- Supabase supplies Postgres, Realtime, and Auth as one managed unit, so the persistence and notification-recovery decisions above require no additional infrastructure to stand up.
- The chosen first `CapabilityDefinition` is the smallest one that still proves real dispatch mechanics, keeping the vertical slice genuinely minimal.

### Costs and risks

- `companyd` as one process is a single deployment unit for six logically separate modules; a future need to scale or isolate one of them (for example, Runtime under heavy load) requires a deliberate split, not just a config change.
- Supabase Realtime's `LISTEN`/`NOTIFY` has known delivery limits under connection loss; the polling sweep exists specifically because this path is best-effort, not because it's sufficient alone.
- A single `ModelProfile`/`ProviderAdapter` pair means the first slice does not exercise Intelligence's ranking/fallback logic — that remains unproven until a second profile is added.
- Go for `companyd` and TypeScript for Next.js means two languages and two dependency ecosystems from day one; the boundary between them (the Application adapter contract) has to stay disciplined or it becomes an informal second orchestration layer.
- A single DigitalOcean VPS is a single point of failure with manual patching, scaling, and restart configuration, unlike a managed platform; systemd handles process-level restart, but host-level failure (VPS outage, disk, OS) has no built-in redundancy at this stage.

## Alternatives rejected by this proposal

- **Node/TypeScript for core services instead of Go:** rejected — the project owner's stated starting point, and consistent with Go's goroutine-based concurrency model fitting the Daemon's continuously-alive, department-loops-wake-on-work shape discussed earlier in this project's design conversations.
- **A separate message broker (e.g., a queue service) for notification:** rejected for the first slice — no demonstrated requirement yet; Supabase Realtime plus a polling fallback satisfies the existing invariant that notification is a recoverable hint, not a dependency.
- **Event-sourcing persistence:** rejected — the approved `EventRepository`/`Outbox` port already describes an outbox pattern, not replay-derived state; adopting event sourcing now would mean building capability the ports don't ask for.
- **Adapting an external workflow engine (Temporal, Inngest, Trigger.dev) for the first Runtime:** rejected per [ADR-0001](ADR-0001-kernel-runtime-daemon.md)'s minimal-infrastructure principle — none of them is justified before a first slice demonstrates a concrete durability requirement beyond what an internal implementation provides.
- **An Engineering/coding-agent capability as the first `CapabilityDefinition`:** rejected — it requires Workspace isolation, Git discipline, and coding-agent provider integration, all real but unnecessary weight for proving Workflow lifecycle mechanics.
- **A managed container/PaaS platform (Fly.io, Railway, Render) for `companyd`:** considered and not chosen — the project owner's stated preference is a self-managed DigitalOcean VPS with systemd supervision instead.

## Acceptance criteria

- [x] Every technology selected here implements an existing approved port or adapter boundary (`persistence.md`, `runtime.md`, `identity.md`, `execution.md`, `intelligence.md`) without requiring a change to any of them.
- [x] No `CRITICAL` or `MAJOR` conflict with `ADR-0001`, `ADR-0002`, `ADR-0003`, or any approved architecture/domain document (see [Audit finding severity](../../AGENTS.md#audit-finding-severity)).
- [x] the project owner reviews and explicitly changes `Status: PROPOSED` to `Status: APPROVED`.

The four open questions below are unresolved implementation detail within the decisions already fixed above (which LLM model, which transport, which RLS policies, which retry constants) — not a condition of accepting the deployment topology, persistence adapter, Runtime implementation, notification-recovery mechanism, first `CapabilityDefinition`, or Identity adapter decided by this ADR.

## Open questions

- OPEN QUESTION: Which concrete LLM provider and model back the first `ModelProfile`/`ProviderAdapter`?
- OPEN QUESTION: What transport does Next.js use to call `companyd` — REST, gRPC, or another RPC mechanism?
- OPEN QUESTION: What Supabase Row-Level Security policy design enforces organization isolation at the database layer, alongside Application-layer checks?
- OPEN QUESTION: What are the default attempt limits, backoff basis, and jitter for the first Retry policy (tracked jointly with `execution.md`'s open questions)?