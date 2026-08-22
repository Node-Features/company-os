# CompanyOS Failure-Injection Testing

Status: APPROVED

## Purpose

This document specifies crash, retry, replay, and recovery testing — grounding validation of [`workspaces.md`](../architecture/workspaces.md)'s recovery/reconciliation guarantees (lease fencing, checkpoint resume, corruption retry) and [`runtime.md`](../architecture/runtime.md)'s equivalent guarantees before Phase 6 builds them, and before Phase 9 relies on them in production.

## Principle: test the failure semantics already specified, don't invent new ones

`runtime.md` and `daemon.md` already specify detailed failure semantics (retry classification, lease/fencing, at-least-once delivery assumptions, crash-safe recovery from persisted state). This document's job is to specify *how those specified behaviors get tested*, not to add new failure-handling rules. A failure-injection test that reveals behavior contradicting `runtime.md` or `daemon.md` is a bug in the implementation or in those documents — not license to invent a new rule inside a test.

## Failure classes to inject

| Failure class | What it simulates | Must-hold invariant (from `runtime.md`/`daemon.md`) |
|---|---|---|
| Worker crash mid-dispatch | Runtime process dies after claiming an intent, before recording a result | A persisted intent is never lost; the lease expires and a later Sweep reclaims it as a new attempt against the same logical operation |
| Duplicate delivery | The same result/event is submitted more than once | `SubmitResult`'s idempotency-key replay guard returns the original outcome, not a second transition |
| Lease expiry under a slow attempt | An attempt's lease expires while it is still legitimately running | `LEASE_EXPIRED` is recorded; a duplicate in-flight attempt does not corrupt state if it later reports back (fencing token rejects the stale write) |
| Concurrent claim race | Two Runtime instances (or two goroutines) attempt to claim the same due intent simultaneously | Exactly one claim succeeds (`FOR UPDATE SKIP LOCKED` semantics); the other observes no claim, not a corrupted or double-executed attempt |
| Outbox publish failure | `RealtimePublisher`/`Sweeper` fails to publish a batch | Rows remain unpublished (`MarkPublishFailed`), retried on the next Sweep; no event is silently dropped |
| Daemon shutdown mid-drain | Shutdown signal arrives while attempts are in flight | Bounded drain completes in-flight work before dependencies close, per `daemon.md`'s Lifecycle step 7; abandoned leases are safe to reclaim after restart |
| Retry exhaustion | An attempt fails past its capability's `MaxAttempts` | Terminal failure (`FAILED_TERMINAL`) is recorded and `REJECT_WORKFLOW_RESULT` submitted — no silent infinite retry |
| Stale-version write | A commit is attempted against an outdated `expectedVersion` | `ErrConflict` surfaces as the `CONFLICT` outcome; no state is corrupted or partially applied |

## Test construction pattern

Each failure-class test:

1. Drives the system to the specific pre-failure state (e.g., an attempt `CLAIMED` but not yet `DISPATCHED`).
2. Injects the failure directly against persistence or the in-process component — not by relying on real wall-clock timing where a controllable substitute exists (e.g., force an expired lease by writing a past `lease_expires_at` rather than sleeping past the real lease duration).
3. Drives the recovery path (a Sweep, a retry, a restart) and asserts the must-hold invariant from the table above, against real persistence where the invariant is a persistence-level guarantee (compare-and-write, fencing), or against a fake where the invariant is a pure ordering/concurrency property per [`strategy.md`](strategy.md)'s fake-sanction decision.

## Relationship to integration and contract tests

Failure-injection tests are integration-shaped (they exercise real persistence and real concurrency where the invariant depends on it) but distinguished from `strategy.md`'s general integration tests by intent: an integration test proves a use case's happy-path and direct-error-path correctness; a failure-injection test proves the system recovers correctly from a fault that was not directly requested by any caller. A conformance suite ([`contract-tests.md`](contract-tests.md)) proves a port's general contract; a failure-injection test proves a specific fault scenario across the whole claim-dispatch-report loop, potentially spanning multiple ports.

## Priority order

1. Worker crash mid-dispatch and concurrent claim race — these are the two failure modes most directly exercised by Runtime's core claim loop and most likely to hide a real bug today.
2. Duplicate delivery and stale-version write — both already have a partial existing home in `internal/application`'s test suite (idempotency replay, conflict-on-stale-version); this document's job is to make sure they are explicitly labeled as failure-injection coverage, not incidentally covered.
3. Outbox publish failure — directly testable against `Sweeper`/`OutboxRepository` with a `Publisher` stub that fails on demand.
4. Daemon shutdown mid-drain and lease expiry under a slow attempt — require more test scaffolding (a way to hold an attempt "in flight" deliberately) and are lower priority until Phase 6 introduces longer-running workspace sessions where this matters more.

## Invariants

- A failure-injection test asserts a specific invariant already stated in `runtime.md` or `daemon.md`; it does not encode a new failure-handling rule that doesn't already exist in those documents.
- Where a failure can be triggered deterministically (writing a past-due lease, a stale version, a duplicate idempotency key), the test does so rather than relying on real time or real crashes.
- A failure-injection test that fails means either the implementation violates an already-specified invariant, or the specification itself needs revision — never that the test should be loosened to match observed behavior.

## Open questions

- OPEN QUESTION: is there a need for genuine process-level crash testing (killing the actual `companyd` process mid-operation) beyond simulating the same effect at the persistence/component level, and if so, where does that scaffolding live?
- OPEN QUESTION: how are these tests run in CI given they may require real timing/concurrency control not easily reproducible in a shared CI database — does this require the same ephemeral database service `strategy.md`'s Phase 8 Slice 1 introduces, or a dedicated environment?
- OPEN QUESTION: what failure classes does Phase 6's Workspace/CodingAgentRuntime introduce that aren't covered by this document's Runtime-centric table (e.g., workspace provider unavailability, partial checkpoint corruption)?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [`strategy.md`](strategy.md)
- [`contract-tests.md`](contract-tests.md)
- [Runtime](../architecture/runtime.md)
- [Daemon](../architecture/daemon.md)
- [Execution domain](../domain/execution.md)
- [Workspaces](../architecture/workspaces.md)
