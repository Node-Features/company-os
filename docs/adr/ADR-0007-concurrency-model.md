# ADR-0007: Concurrency Model

Status: PROPOSED

## Context

The framing behind this ADR — independent work runs concurrently; anything touching shared state, authority checks, or financial/business invariants is serialized — is already what this codebase does. No document had stated it as a single rule with a worked example, and no compiled code demonstrated the in-process synchronization case specifically. This ADR fills that gap; it does not change any existing behavior.

One correction belongs here: the request that produced this ADR asked for a `SafeState` type under `kernel/state/`. That location is wrong on two independent grounds, not just a path typo:

1. **Kernel holds no state between calls.** [`kernel.md`](../architecture/kernel.md): "Kernel state contains organizational facts, not worker leases, queue offsets, sockets, or process health," and every real `internal/kernel/workflow` function takes an explicit state snapshot as a parameter and returns a new one — there is no field anywhere in Kernel a mutex could guard, because Kernel never holds a value across two calls.
2. **Financial state is Finance's, not Kernel's.** `docs/departments/finance.md` (`APPROVED` this session) owns Budget/ResourceConstraint/ResourceUsage; kernel.md's five owned aggregates (Organization, Objective, Department, Workflow, Capability) do not include it. There is also no Go implementation of it yet at all — Phase 4 hasn't started — so a `SafeState` claiming to be financial state would be inventing fields this ADR has no authority to invent, the same issue `ADR-0005` flagged for Department.

The demonstration this ADR provides is instead an explicit, labeled illustration of a pattern that already exists for real: [`docs/architecture/persistence.md`](../architecture/persistence.md) requires "versioned writes with optimistic concurrency... conflicting writes fail rather than merge implicitly" for all authoritative state, and `ports.AuthoritativeStateRepository.CommitTransition` already implements exactly that against Postgres for Workflow (verified directly against `workflow_repo.go`: `UPDATE workflows SET ... WHERE ... version=$8`, checking `RowsAffected() != 1` to return `ErrConflict`). The illustration mirrors that same shape in memory, because a database round-trip isn't a self-contained goroutine demonstration.

## Decision

### Concurrency table

| Operation | Concurrent or Serialized | Mechanism | Reason |
|---|---|---|---|
| Two agents reading market data | **Concurrent** | No synchronization — reads of already-persisted, immutable-once-written records never contend | Nothing mutates while being read. *Illustrative: no `MarketData` or `Agent` Go type exists yet — `internal/domain/agent` doesn't exist despite `docs/domain/agent.md` being `APPROVED`.* |
| Two agents writing to shared financial state | **Serialized** | DB transaction + optimistic concurrency (version compare-and-write) | `persistence.md`: "conflicting writes fail rather than merge implicitly"; `finance.md`: budget/resource records cannot be double-counted or silently overwritten. *Illustrative: no Finance Go type exists yet (Phase 4 not started) — demonstrated below in `internal/concurrency/safestate.go`, the same pattern `CommitTransition` uses for Workflow.* |
| Agent publishing an event | **Serialized at commit, concurrent at delivery** | DB transaction (event row written inside the same transaction as the causing state change) + outbox for later async delivery | `events.md`: "Publication begins only from a persisted Event" — never a live, ordering-guaranteed publish call. *Illustrative: no Agent Go type exists; nothing calls `ports.Publisher.Publish` today — verified last turn, `event_outbox` rows are written but nothing reads or marks them published.* |
| Scheduler dispatching tasks | **Concurrent** | goroutine per claimed intent (`go func(c) {...}` in `Runtime.Sweep`) + `FOR UPDATE SKIP LOCKED` at claim time | Real, shipped: `internal/runtime/runtime.go`'s `Sweep` launches one goroutine per `ClaimedExecution`; `ports.ExecutionRepository.ClaimDueIntents`'s doc comment states the `FOR UPDATE SKIP LOCKED` claim, so two dispatch passes (or a retry) can never claim the same intent twice. |
| Workflow engine advancing state | **Serialized** | DB transaction + optimistic concurrency (`expectedVersion` compare-and-write, `ErrConflict` on mismatch) | Real, shipped: `ports.AuthoritativeStateRepository.CommitTransition`, verified against `workflow_repo.go`. `kernel.md`: "Unknown, stale, or conflicting state versions cause rejection rather than best-effort mutation." |
| Authority check before a privileged action | **Concurrent to evaluate, but not itself the safety mechanism** | Re-evaluated at dispatch time (`governance.md` step 8); actual serialization is the same DB compare-and-write as the row above | Real, shipped, and already documented on `governance.Authority.Authorize` (added last turn): "does NOT guarantee ordering or mutual exclusion between concurrent calls... the resulting write conflict is caught downstream by Persistence's compare-and-write." Two concurrent `Authorize` calls for the same resource can both legitimately return `ALLOW` — only the version check that follows actually prevents a lost update. |

Three of six required rows (market data, financial state, agent-publishing) are marked illustrative because their domain types don't exist in Go yet — the table states this plainly rather than presenting invented types as real.

### Why a mutex, not a channel, and why version-checked instead of a bare lock

Both choices are justified in the code itself (`safestate.go`'s doc comments), not just here, so the two stay together as the file evolves. Summary: a mutex fits because the protected operation is a single fast synchronous compare-then-write with no I/O in the critical section — exactly `sync.Mutex`'s designed case, unlike `internal/daemon`, which uses a channel-owned-by-one-goroutine pattern because it interleaves several distinct kinds of work, not one. A bare `Adjust(delta)` guarded only by a mutex would be memory-safe but would let two callers each apply a delta computed from stale information — memory safety is not the same guarantee as `persistence.md`'s "conflicting writes fail rather than merge implicitly." Requiring the caller to state the version it read, and failing loudly on mismatch, is the part of the pattern that would still be correct if this were backed by a real database instead of an in-process mutex — the mutex is this illustration's implementation detail; the version-checked contract is what generalizes.

## Consequences

### Positive

- States, in one place, a rule that was previously only demonstrable by reading five separate files (`runtime.go`, `evaluate.go`, `workflow_repo.go`, `persistence.md`, `governance.md`).
- Gives a compiled, empirically-run proof (`safestate_test.go`) that the optimistic-concurrency pattern has no lost updates under real concurrent goroutines, not just an assertion that it should — 100 goroutines racing to adjust the same balance produced 4,855 observed version conflicts, all correctly retried, ending at the exact expected balance. `go test -race` could not be run in this environment (its CGo/gcc toolchain isn't cooperating with Go's race-detector build on this machine — an environment gap, not a code issue); `runtime.Gosched()` was added between each goroutine's read and write specifically to widen the race window enough that the conflict-and-retry path is actually exercised rather than accidentally avoided by scheduling luck.
- Corrects a specific, checkable misconception (Kernel-as-lock-holder) before it lands in code.

### Costs and risks

- `internal/concurrency` is a new package with no caller anywhere in `companyd` — it exists solely as a worked reference. If it's never linked from anywhere a future reader would actually look, it risks going stale unnoticed. (Mitigated partially by this ADR and by `current-state.md`, but neither is enforced by the compiler.)
- The three illustrative table rows cannot be verified against real code, only against approved-but-unimplemented docs — if `docs/domain/agent.md` or the eventual Finance Go types diverge from what's assumed here, this table would silently go stale too.

## Alternatives rejected by this proposal

- **A single global mutex over "all shared state":** rejected — would serialize genuinely independent work (two different organizations' workflows, two unrelated capability dispatches), directly contradicting `runtime.md`'s per-intent goroutine dispatch and turning a correctness mechanism into a throughput bottleneck. It also provides no protection the moment more than one `companyd` process runs — an open question `daemon.md` has not settled.
- **An in-memory mutex as the real production mechanism for authoritative state** (i.e., treating `SafeState` as more than an illustration): rejected — `persistence.md`: "Conversation, cache, search index, vector index, provider state, checkpoint, and message history are never authoritative business state," and `daemon.md`: "Daemon memory is never authoritative organizational or execution state." An in-process lock cannot protect against a second process, a second `companyd` instance, or a direct database write from anywhere else — only the database's own compare-and-write can.
- **Channel-based serialization for `SafeState`:** rejected for this specific case (see Decision) — reasonable for an owner goroutine interleaving heterogeneous work, not for one synchronous compare-and-write operation with no I/O.

## Acceptance criteria

- [x] Cross-checked against `runtime.md`, `kernel.md`, `governance.md`, `events.md`, and `persistence.md`; the `CommitTransition`/`workflow_repo.go` compare-and-write claim was verified directly against the implementation, not inferred from its interface comment.
- [x] `safestate.go` and `safestate_test.go` compile and pass (100-goroutine contention test: 4,855 version conflicts observed and correctly retried, zero lost updates). `-race` could not be run in this environment — its CGo toolchain isn't currently working here — so this is a strong functional proof, not a formally race-detector-verified one.
- [ ] the project owner reviews and explicitly changes `Status: PROPOSED` to `Status: APPROVED`.

## Open questions

- OPEN QUESTION: today, one goroutine race (two concurrent HTTP requests hitting the same Workflow) is already possible even with `companyd` as one process (`ADR-0004`) — is a database-level advisory lock or `SELECT ... FOR UPDATE` on the `workflows` row needed in addition to the application-level `expectedVersion` check, or does Postgres's default isolation level make the existing compare-and-write sufficient on its own? Not answered here — `workflow_repo.go` was read for this ADR, but Postgres isolation-level behavior under concurrent `UPDATE ... WHERE version=$N` was not independently verified.
- Carried forward from `daemon.md`, unresolved by this ADR: "Is the first Daemon a single process with in-process components or a coordinator for separate worker processes?" — this ADR's in-memory illustration would need to be reconsidered (or explicitly retired in favor of the database-only mechanism) if that answer becomes "coordinator for separate worker processes."

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [Persistence](../architecture/persistence.md)
- [Kernel](../architecture/kernel.md)
- [Runtime](../architecture/runtime.md)
- [Governance](../architecture/governance.md)
- [Events](../architecture/events.md)
- [Finance department](../departments/finance.md)
- [ADR-0001](ADR-0001-kernel-runtime-daemon.md)
- [ADR-0005](ADR-0005-kernel-interface-contracts.md)
