# ADR-0006: Daemon Responsibilities and Boot Sequence

Status: PROPOSED

## Context

[`docs/architecture/daemon.md`](../architecture/daemon.md) (`APPROVED`) already fixes, in prose, what the Daemon owns and does not own, as part of the layer boundary [`ADR-0001`](ADR-0001-kernel-runtime-daemon.md) established. No document pins a concrete, ordered boot sequence or ties that sequence to the one real Go entrypoint — this ADR does that, the same way [`ADR-0005`](ADR-0005-kernel-interface-contracts.md) fixed Kernel's Go-level contract without re-deciding Kernel's ownership.

Two corrections belong in this Context because the request that produced this ADR got them wrong, and getting them right changes the boot sequence itself:

1. There is one daemon binary: `companyd` (module `github.com/Node-Features/company-os/apps/companyd`, entrypoint `apps/companyd/cmd/companyd/main.go`), fixed by [`ADR-0004`](ADR-0004-first-slice-technology-stack.md)'s "`companyd` as one Go process" topology. Not a second `companyosd` binary — that would contradict the approved topology, not implement it.
2. Kernel has no boot-time stage. [`kernel.md`](../architecture/kernel.md) is explicit that Kernel is "usable without a running scheduler, worker, daemon, model, or external workflow engine" and lists "process startup, shutdown, restart" under its own non-responsibilities. Kernel is stateless — `internal/kernel/workflow`'s functions are invoked synchronously, inline, from inside Application at request time. There is nothing to construct or start at boot.

## Decision

### Daemon responsibilities restated from daemon.md, not re-decided here

The Daemon owns exactly what [daemon.md](../architecture/daemon.md) already specifies:

| Daemon owns | Detail |
|---|---|
| Process lifecycle | 24x7 process lifetime; bounded startup and shutdown sequences; signal handling; supervision, restart/backoff, crash-loop detection; draining; lease release; operational pause/resume/emergency-stop |
| Config loading | Process-scoped configuration loading |
| Logging/observability bootstrap | Observability wiring |
| Runtime construction and startup | Configured Runtime workers, schedulers, listeners, and adapters |
| Health reporting | Liveness, readiness, dependency health, worker health, degraded-mode reporting |
| Waking due work | Without owning timer *meaning* |

The Daemon does **not** own, per daemon.md's non-responsibilities:

| Not Daemon's | Actually owned by |
|---|---|
| Organization, objective, department, workflow, capability, policy, approval semantics | Kernel (`kernel.md`) |
| Legal state transitions or interpretation of provider results | Kernel |
| Authoritative state, execution history, checkpoints, queues, timers | Runtime / Persistence |
| Retry policy for workflow operations, capability dispatch, external side effects | Runtime (`runtime.md`) |
| Business scheduling rules (e.g. when an objective should recur) | The owning department, via governed Application use cases |
| Model routing, agent reasoning, workspace execution | Intelligence routing / Agent domain |
| Provider credentials beyond secure injection | Deployment/secrets management |
| OS service managers, container orchestrators, deployment infrastructure | External supervision (systemd, per `ADR-0004`) — may supervise the Daemon, but that does not move CompanyOS semantics into infrastructure |

Both tables restate already-approved text; this ADR adds no new ownership claim.

### Boot sequence

Grounded in the real, already-shipped `apps/companyd/cmd/companyd/main.go`, corrected for the two points in Context:

1. **OS starts the `companyd` process.** systemd in production (`ADR-0004`); `air` hot-reload in local development.
2. **Signal context established.** `SIGINT`/`SIGTERM` handled via `signal.NotifyContext` before any dependency is touched, so shutdown is cooperative from the first line of `main`.
3. **Config loading.** `.env` best-effort locally (`godotenv.Load`); production sets real environment variables (`DATABASE_URL`, `PORT`, provider API keys, Supabase credentials).
4. **Logging bootstrap.** Standard-library `log` only today — see Open Questions; daemon.md's "observability wiring" is broader than what exists.
5. **Adapter construction.** Persistence pool (Supabase/Postgres) connects first; intelligence provider adapters (Gemini, OpenAI, Anthropic behind one `fallback.ProviderAdapter`) construct only if that connection succeeded.
6. **Application construction.** Wires persistence repositories, fixtures, and the Runtime wake-up channel. This is what will invoke Kernel decision functions later, per governed request — this step does not "start" Kernel; there is nothing to start.
7. **Runtime construction — the runtime subsystems stage.** Dispatch loop, poll interval, lease duration, provider handle.
8. **Daemon construction and `Start()`.** Supervises Runtime's dispatch loop. This is process-lifecycle ownership in action, not a handoff to Kernel.
9. **Agent/workflow-layer entry points registered.** The `/v1/workflows...` HTTP routes, reachable only once steps 5–8 succeeded; `/health` is always registered, even in degraded mode (no `DATABASE_URL`).
10. **HTTP server starts listening.**
11. **Block until shutdown signal**, then: stop accepting HTTP, drain in-flight requests (bounded 5s timeout), shut down the Daemon (which stops Runtime) — dependency-reverse order, per daemon.md's supervision model.

No step is "start Kernel." The original request's proposed sequence — OS → daemon → kernel → runtime subsystems → agent/workflow layer — named one; this sequence corrects that.

### Go entrypoint

`apps/companyd/cmd/companyd/main.go` already implements this sequence and is already wiring-only — no business logic, matching what was asked for. Rather than create a second, parallel file that would either drift from it or duplicate it, it has been annotated in place with comments marking each stage above (see the file itself, and Alternatives below for why a separate skeleton was rejected).

## Consequences

### Positive

- Ties an approved architecture document to the one real entrypoint that implements it, closing a gap `ROADMAP.md` and `daemon.md` both left implicit.
- Corrects a boot-sequence misconception (Kernel-as-boot-stage) before it propagates into other docs, code comments, or a future contributor's mental model.

### Costs and risks

- Comments are not compiler-enforced. If `main.go`'s structure changes later, the stage comments can silently drift out of sync with reality — nothing currently tests that they match.

## Alternatives rejected by this proposal

- **A separate `companyosd` binary:** rejected — contradicts `ADR-0004`'s one-process topology, and would duplicate Application/Runtime/Daemon wiring for no isolation benefit that ADR identified.
- **A literal Kernel construction/start step in `main.go` (e.g. `kernel.New()`):** rejected — Kernel is stateless per `kernel.md`; nothing to construct or start. Modeling one in code would misrepresent the architecture, not just a diagram.
- **A brand-new illustrative skeleton file alongside the real `main.go`:** rejected — the real file is already skeleton-level (wiring only, no business logic). A second file would either drift from it over time or be redundant, with no clear answer to which one is "the" reference.

## Acceptance criteria

- [x] Cross-checked against `daemon.md`, `ADR-0001`, `ADR-0004`, and `kernel.md` for contradictions — none found. Single-session review, not a dedicated audit.
- [ ] the project owner reviews and explicitly changes `Status: PROPOSED` to `Status: APPROVED`.

## Open questions

- OPEN QUESTION: daemon.md claims "observability wiring" as owned, but `main.go` uses only stdlib `log` — no structured logging, metrics, or tracing exists today. Is this in scope before Phase 9's "production-ready" checklist (`ROADMAP.md`), or deferred further?
- Carried forward from daemon.md, unresolved by this ADR: "Is the first Daemon a single process with in-process components or a coordinator for separate worker processes?" and "Which deployment mechanism provides external restart and singleton/leader guarantees?"

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [Daemon](../architecture/daemon.md)
- [Kernel](../architecture/kernel.md)
- [ADR-0001](ADR-0001-kernel-runtime-daemon.md)
- [ADR-0004](ADR-0004-first-slice-technology-stack.md)
- [ADR-0005](ADR-0005-kernel-interface-contracts.md)
