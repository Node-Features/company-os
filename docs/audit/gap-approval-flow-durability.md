# Gap: REQUIRE_APPROVAL Flow Durability

Status: APPROVED (2026-08-24) — problem statement and remediation plan approved. Implementation still requires the project owner to explicitly select and authorize this slice before any code changes, per this repository's doc-gate convention.

Severity: P1 — threatens runtime reliability. See [`findings.md`](findings.md) §2 (invariant 5), §3 rows 4-5.

## Problem

The REQUIRE_APPROVAL path writes across three separate transactions with no atomicity between them:

1. `SaveGovernanceDecision` (`evaluateGovernance`, [pipeline.go:58](../../apps/companyd/internal/application/pipeline.go))
2. `CreatePendingApproval` ([pipeline.go:109](../../apps/companyd/internal/application/pipeline.go))
3. `a.store()` (the idempotency-replay-guard write, e.g. [workflow_cancel.go:89/94/110/115](../../apps/companyd/internal/application/workflow_cancel.go))

Two distinct crash windows, both real (not theoretical — any deploy or infra blip during this sequence triggers them):

- **Between 1 and 2**: a `GovernanceDecision` row with `Outcome=REQUIRE_APPROVAL` is durably recorded, but no `PendingCommand`/`Approval` ever exists. The caller got no response, so on retry `a.replay()` finds nothing (idempotency was never stored) and the whole flow re-runs, producing a correct new PendingCommand/Approval. The first decision row is now permanently orphaned — harmless, but untracked ([`findings.md`](findings.md) §3 row 3, tracked as backlog, not this gap).
- **Between 2 and 3**: a valid, actionable `PendingCommand`/`Approval` now exists. Retry (idempotency lookup misses) re-executes the use case from scratch. `ProposalDigest = canonicalDigest(cmd)` includes a fresh `CommandID`/`DeclaredTime` each call ([`kernel/workflow/digest.go`](../../apps/companyd/internal/kernel/workflow/digest.go)), so the retry produces a **different** digest and a **second, duplicate** `PendingCommand`/`Approval` for what was logically one request. A human approver now sees two entries for the same underlying intent.

Separately, in the already-idempotency-guarded path (Workflow ALLOW outcomes), `a.Repo.IdempotencyStore`'s error is explicitly swallowed: `_ = a.Repo.IdempotencyStore(...)` ([application.go:64](../../apps/companyd/internal/application/application.go)). If that write fails after a successful commit, a later retry finds no replay guard and re-executes the already-completed command. `AuthorizeDispatch`'s two `SaveGovernanceDecision` calls have the same swallow pattern ([workflow_start.go:121-122,141-143](../../apps/companyd/internal/application/workflow_start.go)) — inconsistent with `evaluateGovernance`'s sibling call, which surfaces the identical failure as `Unavailable`.

## Invariant

Restores: a single logical REQUIRE_APPROVAL request produces exactly one durable, actionable Approval; a successful command commit is never silently re-executable due to a swallowed bookkeeping-write failure.

## Proposed approach (plan-level only)

1. Narrow the crash window between `SaveGovernanceDecision` and `CreatePendingApproval` by combining them into one transaction where the repository layer allows it (both already exist as separate `ports` calls into the same underlying store — check whether `ports.AuthoritativeStateRepository` can expose a combined write, following the same shape `CommitTransition` already uses for its own multi-table atomic writes).
2. Close the window between `CreatePendingApproval` and `a.store()` similarly, or — if combining transactions isn't practical here — make retry detection independent of the idempotency-store row succeeding, e.g. by checking for an existing non-terminal `PendingCommand` for the same source/resource before creating a new one (mirrors the dedup pattern `ProposeObjective`'s `GetObjectiveBySource` already uses).
3. Stop swallowing `IdempotencyStore` and `SaveGovernanceDecision` errors in `AuthorizeDispatch` — surface them as `Unavailable`, consistent with every other call site of the same methods.

## Files likely to change

- [`apps/companyd/internal/application/pipeline.go`](../../apps/companyd/internal/application/pipeline.go)
- [`apps/companyd/internal/application/application.go`](../../apps/companyd/internal/application/application.go)
- [`apps/companyd/internal/application/workflow_start.go`](../../apps/companyd/internal/application/workflow_start.go)
- [`apps/companyd/internal/ports`](../../apps/companyd/internal/ports) — repository interface, if a combined-transaction method is added.
- [`apps/companyd/internal/adapters/persistence/supabase`](../../apps/companyd/internal/adapters/persistence/supabase) — corresponding implementation.

## Tests required

**Before (regression baseline):** a fault-injection test (per [`testing/failure-injection.md`](../testing/failure-injection.md)) simulating a crash between `CreatePendingApproval` and `a.store()`, asserting today's actual (undesired) outcome: retry produces two PendingCommand/Approval rows for one logical request.

**After:**
- Same fault-injection scenario, asserting exactly one PendingCommand/Approval survives (or that duplicates are detected and one is resolved as a no-op/conflict).
- A test asserting `IdempotencyStore` failure after a successful commit is now surfaced as `Unavailable`, not silently swallowed, and that a subsequent retry does not re-execute the already-committed command.

## Dependencies

- [`findings.md`](findings.md) §2, §3.
- [`testing/failure-injection.md`](../testing/failure-injection.md) — this is exactly the class of test that document already scopes.
- [`architecture/governance.md`](../architecture/governance.md), [`domain/approval.md`](../domain/approval.md), [`domain/command.md`](../domain/command.md)
