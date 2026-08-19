# CompanyOS Daemon

Status: DRAFT

## Responsibility

The Daemon is the long-lived operational host that keeps CompanyOS available. It starts and supervises Runtime components, wakes due work, receives runtime-facing events, exposes health, and coordinates graceful shutdown. It is a process-lifecycle boundary, not an organizational decision-maker.

The Daemon owns:

- 24x7 process lifetime and bounded startup and shutdown sequences;
- construction and startup of configured Runtime workers, schedulers, listeners, and adapters;
- waking due timers and persisted work without owning timer meaning;
- receiving and forwarding Runtime events and authenticated provider callbacks;
- liveness, readiness, dependency health, worker health, and degraded-mode reporting;
- supervision, restart/backoff, crash-loop detection, draining, and lease release;
- signal handling and operational pause, resume, and emergency-stop mechanisms;
- process-scoped configuration loading and observability wiring.

## Non-responsibilities

The Daemon does not own:

- organization, objective, department, workflow, capability, policy, or approval semantics;
- legal state transitions or interpretation of provider results;
- authoritative state, execution history, checkpoints, queues, or timers;
- retry policy for workflow operations, capability dispatch, or external side effects;
- business scheduling rules such as when an objective should recur;
- model routing, agent reasoning, workspace execution, or provider credentials beyond secure injection;
- operating-system service managers, container orchestrators, or deployment infrastructure.

An OS service manager, container platform, or cloud scheduler may supervise the Daemon externally. That does not move CompanyOS semantics into infrastructure.

## Lifecycle

1. Load and validate process configuration; fail closed on invalid or missing security-critical configuration.
2. Initialize observability and connect to required persistence and messaging dependencies.
3. Run migrations only through an explicit deployment operation, not implicitly during ordinary startup.
4. Construct Runtime adapters, schedulers, workers, and callback/event listeners.
5. Recover expired leases and expose readiness only when mandatory components can safely accept work.
6. Wake due persisted work and continuously supervise components.
7. On shutdown, stop accepting work, drain bounded in-flight operations, checkpoint or abandon leases safely, then close dependencies.

Restart restores availability; it does not reconstruct truth from memory. Runtime recovery uses persisted state as defined in [Runtime](runtime.md).

## Supervision model

- A supervised component reports `starting`, `ready`, `degraded`, `stopping`, or `failed` plus a reason and last transition time.
- Unexpected exits restart with bounded exponential backoff and jitter.
- Repeated failure opens a circuit and marks the Daemon degraded or unready; it does not loop indefinitely at full speed.
- Required-component failure affects readiness. Optional-component failure is isolated and reported.
- Health probes are side-effect free and do not execute workflows.
- A watchdog may detect a stuck component, but termination and recovery must preserve leases and idempotency.

## Invariants

- Daemon memory is never authoritative organizational or execution state.
- Starting the Daemon multiple times cannot duplicate legal state transitions or side effects.
- Only persisted, due, and eligible work is woken.
- Waking work is not equivalent to authorizing or completing it.
- Component restart preserves Runtime execution identity and attempt rules.
- Readiness is false until mandatory dependencies and recovery checks succeed.
- Shutdown stops intake before draining and has an explicit maximum duration.
- Process supervision retry is distinct from Runtime operation retry.
- Provider callbacks are authenticated, recorded, deduplicated, and handed to Runtime; the Daemon does not interpret their organizational meaning.
- Emergency pause prevents new dispatch while preserving durable state and auditability.
- No component obtains broader credentials merely because it shares the Daemon process.
- Deployment topology can change without changing Kernel semantics.

## OSS evidence and disposition

The detailed six-project evidence matrix is maintained in [Runtime: OSS evidence](runtime.md#oss-evidence). For the Daemon boundary specifically:

- Temporal workers demonstrate disposable polling processes recovering from durable service history; CompanyOS borrows disposable workers but not provider-owned organizational truth.
- Inngest and Trigger.dev demonstrate separate queues, executors, heartbeats, concurrency control, and recovery; CompanyOS keeps those mechanics behind Runtime.
- JARVIS demonstrates an always-on daemon, startup of background services, health/status commands, and emergency controls; CompanyOS rejects combining that host with domain authority.
- LangGraph.js offers checkpoint/resume evidence but not a complete 24x7 supervisor, so it does not define the Daemon boundary.

## Open questions

- OPEN QUESTION: Is the first Daemon a single process with in-process components or a coordinator for separate worker processes?
- OPEN QUESTION: Which components are mandatory for readiness in the first vertical slice?
- OPEN QUESTION: Which deployment mechanism provides external restart and singleton/leader guarantees?
- OPEN QUESTION: How are scheduler leadership and duplicate wake-ups handled when multiple Daemons run?
- OPEN QUESTION: What drain deadline and lease-expiry behavior apply during shutdown?
- OPEN QUESTION: Which operational pause scopes are required: organization, department, workflow, capability, provider, or global?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [System context](system-context.md)
- [Runtime](runtime.md)
- Future security and deployment specifications
