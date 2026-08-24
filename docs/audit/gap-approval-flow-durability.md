# Gap: REQUIRE_APPROVAL Flow Durability

Status: APPROVED (2026-08-24) — problem statement and remediation plan approved. **Partially resolved 2026-08-24**, project-owner-authorized as part of a concurrency-test-matrix pass: the idempotency-key portion below (the swallowed-error bullet, and the concurrent-request half of the "between 2 and 3" crash window) is fixed and tested. The narrower crash-then-retry-after-TTL duplicate-write window, and crash window 1, remain open — see [`docs/testing/concurrency-guarantees.md`](../testing/concurrency-guarantees.md) for the full before/after record and residual-gap accounting.

Severity: P1 — threatens runtime reliability. See [`findings.md`](findings.md) §2 (invariant 5), §3 rows 4-5.

## Problem

The REQUIRE_APPROVAL path writes across three separate transactions with no atomicity between them:

1. `SaveGovernanceDecision` (`evaluateGovernance`, [pipeline.go:58](../../apps/companyd/internal/application/pipeline.go))
2. `CreatePendingApproval` ([pipeline.go:109](../../apps/companyd/internal/application/pipeline.go))
3. `a.store()` (the idempotency-replay-guard write, e.g. [workflow_cancel.go:89/94/110/115](../../apps/companyd/internal/application/workflow_cancel.go))

Two distinct crash windows, both real (not theoretical — any deploy or infra blip during this sequence triggers them):

- **Between 1 and 2**: a `GovernanceDecision` row with `Outcome=REQUIRE_APPROVAL` is durably recorded, but no `PendingCommand`/`Approval` ever exists. **Still open** — untouched by the 2026-08-24 fix, which reservations the idempotency key but does not yet combine `SaveGovernanceDecision` and `CreatePendingApproval` into one transaction. On retry (same key), the reservation is either still live (`Indeterminate`, if within `idempotencyReservationTTL`) or reclaimable (after it) — reclaiming re-runs the flow from scratch, producing a correct new PendingCommand/Approval; the first decision row stays permanently orphaned — harmless, but untracked ([`findings.md`](findings.md) §3 row 3, tracked as backlog, not this gap).
- ~~**Between 2 and 3**: a valid, actionable `PendingCommand`/`Approval` now exists. Retry (idempotency lookup misses) re-executes the use case from scratch.~~ **Concurrent-request case FIXED 2026-08-24**: two requests genuinely racing with the same idempotency key can no longer both reach `CreatePendingApproval` — `IdempotencyReserve`'s atomic upsert (`workflow_repo.go`) means only one caller ever wins the reservation; a racer that arrives after the winner's `finalize` gets the winner's own `APPROVAL_REQUIRED` outcome replayed, not a second execution. Proven: `TestIntegration_CancelWorkflow_ApprovalResolutionRacesCommandRetry` (`apps/companyd/internal/application/idempotency_race_integration_test.go`). **Still open, narrower**: a crash landing in the specific gap between `CreatePendingApproval` committing and `finalize` running, followed by a retry more than `idempotencyReservationTTL` (30s) later, still reclaims and re-executes — a real but much narrower window than "any concurrent request, reliably reproducible." See [`docs/testing/concurrency-guarantees.md`](../testing/concurrency-guarantees.md)'s "Known residual gaps." Fully closing it requires folding `finalize` into `CreateWorkflow`/`CommitTransition`'s own transaction, not done this pass.

~~Separately, in the already-idempotency-guarded path (Workflow ALLOW outcomes), `a.Repo.IdempotencyStore`'s error is explicitly swallowed: `_ = a.Repo.IdempotencyStore(...)` ([application.go:64](../../apps/companyd/internal/application/application.go)).~~ **FIXED 2026-08-24**: `IdempotencyStore`/`IdempotencyLookup` were replaced by `IdempotencyReserve`/`IdempotencyFinalize`; `finalize`'s error is now logged (`log.Printf`), not discarded, and — more importantly — no longer the only thing standing between a retry and re-execution, since the reservation itself (not the final store) is what a retry checks first. `AuthorizeDispatch`'s two `SaveGovernanceDecision` calls **still swallow** their error ([workflow_start.go](../../apps/companyd/internal/application/workflow_start.go)) — untouched by this pass, still inconsistent with `evaluateGovernance`'s sibling call.

## Invariant

Restores: a single logical REQUIRE_APPROVAL request produces exactly one durable, actionable Approval; a successful command commit is never silently re-executable due to a swallowed bookkeeping-write failure.

## Proposed approach (plan-level only)

1. **Still open.** Narrow the crash window between `SaveGovernanceDecision` and `CreatePendingApproval` by combining them into one transaction where the repository layer allows it.
2. **Done differently than originally proposed, 2026-08-24.** Rather than combining `CreatePendingApproval` and the idempotency write into one transaction, or checking for an existing non-terminal `PendingCommand`, the fix moved the idempotency guard's *reservation* ahead of the whole use case (`IdempotencyReserve`, an atomic upsert, called before governance/kernel ever run) instead of only guarding the final write. This closes the reliably-reproducible concurrent-request case without a transaction-combining change to `pipeline.go`. The narrower crash-then-retry-after-TTL window this doesn't close still matches this step's original spirit — still open, see the "Between 2 and 3" note above.
3. **Half done, 2026-08-24.** `IdempotencyStore`'s swallowed error is fixed (now logged, and structurally less load-bearing — see above). `AuthorizeDispatch`'s two `SaveGovernanceDecision` swallows are untouched — still open.

## Files likely to change

Actually changed, 2026-08-24 (items 2 and 3's `IdempotencyStore` half only):

- [`apps/companyd/internal/application/application.go`](../../apps/companyd/internal/application/application.go) — `replay`/`store` → `reserveOrReplay`/`finalize`
- [`apps/companyd/internal/application/workflow_create.go`](../../apps/companyd/internal/application/workflow_create.go), [`workflow_cancel.go`](../../apps/companyd/internal/application/workflow_cancel.go), [`workflow_start.go`](../../apps/companyd/internal/application/workflow_start.go), [`result_submit.go`](../../apps/companyd/internal/application/result_submit.go) — call-site rename
- [`apps/companyd/internal/adapters/persistence/supabase/workflow_repo.go`](../../apps/companyd/internal/adapters/persistence/supabase/workflow_repo.go) — `IdempotencyLookup`/`IdempotencyStore` → `IdempotencyReserve`/`IdempotencyFinalize`
- [`apps/companyd/internal/ports/persistence.go`](../../apps/companyd/internal/ports/persistence.go) — interface + `IdempotencyInProgress` sentinel
- [`apps/companyd/internal/application/fake_repo_test.go`](../../apps/companyd/internal/application/fake_repo_test.go) — matching test double

Still to change for the remaining open items: `pipeline.go` (crash window 1), `workflow_start.go`'s `AuthorizeDispatch` (item 3's other half).

## Tests required

**Done, 2026-08-24:** `TestIntegration_CreateWorkflow_ConcurrentSameIdempotencyKey_OneWorkflow` and `TestIntegration_CancelWorkflow_ApprovalResolutionRacesCommandRetry` (`apps/companyd/internal/application/idempotency_race_integration_test.go`) — real concurrent goroutines, real Postgres, proving the concurrent-request case is closed. See [`docs/testing/concurrency-guarantees.md`](../testing/concurrency-guarantees.md) for the full record.

**Still needed for the remaining open items:** a fault-injection test (per [`testing/failure-injection.md`](../testing/failure-injection.md)) simulating a crash between `SaveGovernanceDecision` and `CreatePendingApproval` (window 1), and one simulating a crash between the domain write and `finalize` followed by a retry after `idempotencyReservationTTL` (window 2's narrowed remainder) — both still produce a real, if narrow, duplicate today.

## Dependencies

- [`findings.md`](findings.md) §2, §3.
- [`testing/failure-injection.md`](../testing/failure-injection.md) — this is exactly the class of test that document already scopes.
- [`architecture/governance.md`](../architecture/governance.md), [`domain/approval.md`](../domain/approval.md), [`domain/command.md`](../domain/command.md)
