# CompanyOS Contract Tests

Status: APPROVED

## Purpose

This document specifies shared-contract and adapter verification: how CompanyOS proves that every implementation of a port (`internal/ports`) — a real adapter and any in-memory fake alike — actually satisfies that port's documented behavior, not just its method signatures. It is a named prerequisite of `ROADMAP.md` Phase 8's contract-test CI job and of Phase 6's `WorkspaceProvider`/`CodingAgentRuntime` adapters, which need the same discipline applied to genuinely new ports.

## The problem this solves

Go's type system verifies that a struct satisfies an interface's method signatures. It verifies nothing about behavior: a `ports.ExecutionRepository` implementation could return `ErrConflict` on the wrong condition, silently permit a double-claim under concurrent callers, or violate an ordering guarantee the port's doc comment promises — and compile cleanly. Today, [`internal/application/fake_repo_test.go`](../../apps/companyd/internal/application/fake_repo_test.go)'s fakes and the real `apps/companyd/internal/adapters/persistence/supabase` adapters both implement the same ports with no shared test proving they agree on behavior.

## Pattern: one conformance suite per port, run against every implementation

For each port in `internal/ports` with more than one implementation (or that a new adapter will be added against), define one exported test-suite function taking the port interface as a parameter:

```go
// package portstest (or colocated with the port)
func RunExecutionRepositoryConformance(t *testing.T, repo ports.ExecutionRepository, orgID uuid.UUID) {
    t.Run("ClaimDueIntents does not double-claim under concurrent callers", func(t *testing.T) { ... })
    t.Run("RecordDispatched fails when fencing token is stale", func(t *testing.T) { ... })
    t.Run("RecordTerminal is idempotent under retry", func(t *testing.T) { ... })
    // ...
}
```

Each adapter's own test file calls this suite against its concrete instance:

```go
func TestSupabaseExecutionRepository_Conformance(t *testing.T) {
    repo := requireRealExecutionRepository(t)
    portstest.RunExecutionRepositoryConformance(t, repo, testOrgID)
}
```

A fake that claims to implement the same port runs the identical suite. If the fake diverges from the real adapter's behavior, the conformance suite catches it directly — this is what makes a fake trustworthy as a stand-in per [`strategy.md`](strategy.md)'s sanctioned-fakes decision, rather than merely convenient.

## What a conformance suite asserts

At minimum, for each port method: its documented error conditions (`ErrNotFound`, `ErrConflict`, lease-fencing failures), its documented concurrency guarantees (compare-and-write semantics, `FOR UPDATE SKIP LOCKED`-style claim exclusivity), its documented idempotency behavior (a retried call with the same identity does not duplicate an effect), and its documented ordering guarantees (e.g. `OutboxRepository.LoadUnpublished` returns oldest-enqueued-first).

A conformance suite does not assert implementation details (query shape, table structure) — only the behavioral contract the port's doc comment already promises. This keeps the suite reusable across genuinely different backends (e.g. a future non-Postgres persistence adapter) without becoming a change-detector for internals.

## Priority order

Per `ROADMAP.md` Phase 8 Slice 2: start with today's persistence adapters (`AuthoritativeStateRepository`, `ExecutionRepository`, `OutboxRepository`, `PendingCommandRepository`), since they already have two implementations each (real + fake) with no shared proof of agreement. Extend to the Identity `Authenticator` adapter when Phase 3 Slice 3 lands, and to `WorkspaceProvider`/`CodingAgentRuntime` adapters when Phase 6 adds them — those are exactly the ports `ROADMAP.md` names as needing "the same discipline."

## Relationship to integration and unit tests

A conformance suite is not a replacement for [`strategy.md`](strategy.md)'s integration tests — an integration test proves one specific use case's end-to-end correctness; a conformance suite proves a port's *general* contract holds for *any* caller, independent of any one use case. Both are required where they overlap; neither subsumes the other.

## Invariants

- A port with more than one implementation has exactly one conformance suite, run against every implementation, not one ad hoc test per adapter.
- A conformance suite asserts only what the port's own doc comment documents — it does not encode one implementation's internal behavior as if it were the contract.
- A new adapter for an existing port is not considered complete until it passes that port's conformance suite.
- A fake sanctioned under [`strategy.md`](strategy.md)'s decision passes the same conformance suite as the real adapter it stands in for; if it cannot, it is not a valid stand-in for that port's contract-relevant behavior, only for its narrower unit-test purpose.

## Open questions

- OPEN QUESTION: where do conformance suites live — a shared `internal/ports/portstest` package, or colocated with each port's own file? (This document proposes the former for reuse across adapter packages, but leaves it open pending implementation.)
- OPEN QUESTION: does the CI contract-test job (`ROADMAP.md` Phase 8 Slice 2) require a real database service (same as integration tests), or can some conformance assertions run against an in-memory-only substitute for speed?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [`strategy.md`](strategy.md)
- [`ROADMAP.md`](../../ROADMAP.md)
- [Persistence architecture](../architecture/persistence.md)
- [Application architecture](../architecture/application.md)
