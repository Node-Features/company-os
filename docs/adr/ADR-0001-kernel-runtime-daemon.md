# ADR-0001: Kernel, Runtime, and Daemon Separation

Status: APPROVED

## Context

CompanyOS needs organizational semantics (what is legal), execution mechanics (how work happens), and continuous availability (keeping the system running) to evolve independently. Without an explicit boundary, execution or process-lifetime concerns tend to leak into domain meaning — a workflow engine's retry model becomes the definition of "retry," a scheduler's uptime becomes a stand-in for organizational authority. [`ARCHITECTURE.md`](../../ARCHITECTURE.md), [`kernel.md`](../architecture/kernel.md), [`runtime.md`](../architecture/runtime.md), [`daemon.md`](../architecture/daemon.md), and [`execution.md`](../domain/execution.md) already specify this boundary in detail as drafts; this record proposes accepting that boundary as CompanyOS's execution model.

## Proposed decision

Adopt the three-layer separation already specified across the referenced documents:

1. **Kernel** is the provider-independent domain authority. It decides legal organizational transitions from a command plus an explicitly loaded authoritative-state snapshot, and is usable without a running scheduler, worker, daemon, model, or external workflow engine. It never reads wall-clock time, network state, or environment variables implicitly, and never owns queues, workers, leases, or process lifetime.
2. **Runtime** executes only Kernel-authorized, persisted intent. It owns execution mechanics — attempts, leases, checkpoints, waits, retries, and resume, per the canonical [Execution domain](../domain/execution.md) — but cannot create, waive, or reinterpret Kernel legality, and cannot itself advance authoritative Workflow (or other aggregate) state.
3. **Daemon** is the continuous-availability and process-lifecycle boundary. It starts and supervises Runtime components, wakes due persisted work, and exposes health — but owns no organizational semantics, and Daemon memory is never authoritative state. Waking work is not equivalent to authorizing or completing it.

Coordination among the three happens only through the [Application layer](../architecture/application.md)'s two-stage Kernel validation, exact Governance evaluation, and atomic persistence sequence — never through a direct call from Daemon or Runtime into Kernel decision logic, and never through Runtime inferring legality from its own execution state.

## Consequences

### Positive

- Infrastructure choices (queue technology, scheduler, database, deployment topology, single-process vs. distributed Daemon) can change without redesigning organizational semantics.
- The Kernel is independently testable — legality can be verified without a running scheduler, database, or network.
- A workflow engine, queue, or process supervisor can be replaced by swapping a Runtime or Daemon adapter, without touching Kernel or domain contracts.
- Crash recovery is well-defined: Runtime execution state (leases, checkpoints) can be lost or reconstructed without corrupting organizational truth, because that truth lives only in Kernel-decided, atomically persisted state.

### Costs and risks

- Three explicit layers add indirection and coordination overhead compared to a monolithic engine that both decides and executes.
- The boundary requires ongoing discipline: it is easy for a Runtime adapter to "helpfully" encode a business rule (for example, treating a specific HTTP status as an organizational rejection) unless reviews actively watch for it.
- Atomic persistence of state, events, and execution intent across the Application/Kernel/Persistence boundary is a real implementation burden that a single-engine design would not face.

## Alternatives rejected by this proposal

- **A monolithic workflow engine owns both meaning and execution:** rejected because it makes engine internals (history format, retry semantics, checkpoint schema) the organizational source of truth. The OSS evidence in `runtime.md` and `persistence.md` explicitly borrows durable-execution mechanics from Temporal, LangGraph, Inngest, and Trigger.dev while rejecting their history/state as CompanyOS domain authority.
- **The Daemon directly executes business logic:** rejected because it conflates process lifetime with organizational decision-making. `daemon.md`'s OSS evidence explicitly rejects JARVIS's combined always-on-daemon-plus-domain-authority pattern on this basis.
- **Application dispatches directly to providers with no Runtime layer:** rejected because it loses durable execution identity, leases, retries, and resumability — every provider failure would then require re-deciding legality from scratch instead of resuming a bounded execution attempt.
- **Kernel reads live process/infrastructure state to decide legality:** rejected because it would make organizational decisions non-deterministic and untestable in isolation, contradicting the invariant that Kernel decisions are deterministic for the same command, state, policy inputs, and declared time.

## Acceptance criteria

- [x] `kernel.md`, `runtime.md`, `daemon.md`, `execution.md`, and `application.md`'s orchestration sequence are internally consistent — verified by the fresh read-only architecture audit completed 2026-08-20, which found no `CRITICAL` or `MAJOR` contradiction among them (see [Audit finding severity](../../AGENTS.md#audit-finding-severity)).
- [x] `kernel.md`, `runtime.md`, `daemon.md`, `execution.md`, and `application.md` are themselves `APPROVED` architecture and domain documents (2026-08-20), not merely mutually consistent drafts.
- [x] the project owner reviews and explicitly changes `Status: PROPOSED` to `Status: APPROVED`.

The first concrete Runtime implementation/adapter and Daemon deployment topology remain open implementation choices (tracked in `.companyos/agent-memory/current-state.md`) and are not required before accepting this boundary — this ADR fixes the shape of the separation, not its first concrete implementation.

## Open questions

- OPEN QUESTION: Is the first Daemon a single process with in-process components or a coordinator for separate worker processes?
- OPEN QUESTION: Which components are mandatory for readiness in the first vertical slice?
- OPEN QUESTION: How are scheduler leadership and duplicate wake-ups handled when multiple Daemons run?
- OPEN QUESTION: Will the first Runtime be an internal implementation or an adapter over a durable engine?
