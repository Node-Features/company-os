# CompanyOS Testing Strategy

Status: APPROVED

## Purpose

This document specifies overall testing responsibilities and levels, and settles `ROADMAP.md` Phase 2 Slice 8's open question: whether `internal/application`'s in-memory fake repositories (`fakeRepo`, `fakeExec`, `fakePending`) should be retired in favor of integration-only tests, or kept as a documented, sanctioned pattern other modules reuse.

## Test levels

| Level | What it proves | Where it runs | Speed/isolation |
|---|---|---|---|
| Unit | Pure logic: Kernel decision functions, Governance policy evaluation, pipeline orchestration against in-memory doubles | `go test` per package, no external dependency | Fastest; fully deterministic |
| Integration | The same use case's outcome against real persistence: real Postgres/Supabase, real transactions, real compare-and-write conflicts | `go test` against a real `DATABASE_URL` | Slower; proves the adapter, not just the orchestration logic |
| Contract | An adapter or port implementation conforms to its port's documented behavior, run against every implementation of that port | Shared test-suite function, run against fakes and real adapters alike | See [`contract-tests.md`](contract-tests.md) |
| Failure-injection | Crash, retry, replay, and recovery behavior under simulated faults | Targeted tests against real or near-real infrastructure | See [`failure-injection.md`](failure-injection.md) |
| End-to-end (web) | The full create→start→poll (or push-update) flow through `web` against a real backend | Phase 8 Slice 3, not yet built | Slowest; fewest tests |

No level substitutes for another. A unit test proving Kernel legality does not prove the database enforces the same compare-and-write guarantee; an integration test proving the database does the right thing does not prove a rare concurrency interleaving is handled, because triggering that interleaving reliably against a real database is exactly what unit-level fakes are good at and integration tests are not.

## The fake-repository decision

**Decision: fakes are a sanctioned, documented pattern for concurrency and ordering tests — not a substitute for integration coverage of state-transition correctness, and not something each new module invents independently.**

This is not a default toward "keep everything as-is" — it is a specific, narrower sanction than that, based on what the existing test suite actually demonstrates:

- `internal/application/integration_test.go` already proves `CreateWorkflow`/`StartWorkflow`/`SubmitResult`/`CancelWorkflow`'s state-transition correctness against the real database — this is the authoritative proof of correctness for those use cases. A fake must never be treated as sufficient evidence that a transition is correct; only an integration test against real persistence is.
- `internal/application/pipeline_test.go`'s fake-backed tests prove something integration tests structurally cannot prove reliably: `TestStartWorkflow_ConcurrentSameVersion_OneAcceptedOneConflict` (a genuine race between two concurrent callers) and `TestStartWorkflow_NotifiesOnlyAfterCommit` (ordering between commit and notification) depend on deterministic, controllable timing that a real database and network round-trip make flaky to assert on. These are legitimate, valuable tests that would be lost, not just moved, by blanket retirement.

Therefore: **fakes are retired as a stand-in for "the happy-path transition works"** (that claim now belongs to integration tests only) **and sanctioned as the correct tool for "this concurrency/ordering/timing interaction behaves correctly under adversarial interleaving."** A new module (Research, M&E, Finance, and beyond) follows the same split: write its state-transition correctness tests as integration tests against real persistence first; add a fake-backed unit test only when it needs to force a specific interleaving or timing condition integration tests cannot reliably reproduce, and name that reason in the test's doc comment.

A module's own in-memory fake, if it needs one, follows `fakeRepo`/`fakeExec`'s existing shape (minimal, package-private, documented with which port it stands in for and why) rather than inventing a new convention.

## CI gates

Today (per `.github/workflows/ci.yml`): `companyd` runs `go vet ./...` and `go test ./...` — the latter includes both unit and integration-level tests in the same run, which requires a reachable `DATABASE_URL` in CI. `web` runs `npm install` and `npm run build`. There is no database service in CI yet, no contract-test job, and no e2e job.

This document does not implement CI changes — that is [`ROADMAP.md`](../../ROADMAP.md) Phase 8's scope (`testing/strategy.md` and `testing/contract-tests.md` are its named prerequisites). It fixes the target shape Phase 8 builds toward:

- Phase 8 Slice 1 adds an ephemeral Postgres/Supabase service so integration tests run in CI, not only against a developer's local `DATABASE_URL`.
- Phase 8 Slice 2 adds a contract-test CI job per [`contract-tests.md`](contract-tests.md).
- Phase 8 Slice 3 adds web e2e tests per this document's end-to-end row.

## Responsibilities by owner

- **Kernel/domain code** (`internal/kernel`, `internal/domain`): unit tests only, no persistence dependency — legality and invariants are pure functions.
- **Application use cases** (`internal/application`): both integration (state-transition correctness, mandatory) and, where a concurrency/ordering claim is being made, unit tests against fakes per the decision above.
- **Adapters** (`internal/adapters/persistence`, `internal/adapters/notify`, `internal/adapters/intelligence`): contract tests against their port (see [`contract-tests.md`](contract-tests.md)), plus integration tests where a real external dependency exists.
- **Runtime/Daemon**: failure-injection tests per [`failure-injection.md`](failure-injection.md) for crash/lease/retry behavior; unit tests for backoff/retry-classification pure logic.
- **web**: type-checking and build today; e2e tests from Phase 8 Slice 3 onward.

## Invariants

- A state-transition correctness claim requires an integration test against real persistence; a fake-backed test alone never satisfies that claim.
- A fake, when used, documents which port it stands in for and which specific concurrency/ordering/timing property it exists to test.
- No module invents its own ad hoc testing convention where this document already specifies one; a deviation is named and justified in that module's own test file, not silently done.
- CI gates only what this document and `contract-tests.md`/`failure-injection.md` specify as required; adding a new required gate updates this document first.

## Open questions

- OPEN QUESTION: what coverage threshold, if any, is enforced, and at what level (unit vs. integration)?
- OPEN QUESTION: should integration tests run against a per-developer local Supabase instance, a shared test project, or an ephemeral CI-only instance — Phase 8 Slice 1's own open question, not resolved here.
- OPEN QUESTION: at what point does `web` get a real test runner configured (none exists today beyond `tsc`/`next build`)?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [`ROADMAP.md`](../../ROADMAP.md)
- [`contract-tests.md`](contract-tests.md)
- [`failure-injection.md`](failure-injection.md)
- [Application architecture](../architecture/application.md)
- [Runtime](../architecture/runtime.md)
