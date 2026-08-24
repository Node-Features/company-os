# CompanyOS Concurrency Guarantees

Status: APPROVED

## Purpose

[`strategy.md`](strategy.md) sets test levels and [`failure-injection.md`](failure-injection.md)
specifies which failure classes to inject and what must hold when they're injected. Neither states,
in one place, which specific CompanyOS mechanisms are **exactly-once** versus
**at-least-once-with-idempotent-handling** versus **best-effort** — a distinction every caller
integrating against this system needs, and one a full concurrency-test-matrix pass
(`docs/audit/gap-approval-flow-durability.md`'s idempotency-key finding, closed by this pass) is
positioned to state precisely, with the tests to back each claim. This document is that record: the
guarantee each mechanism actually provides, the file:line evidence, and the test that proves it.

It also records one real fix this pass made, not just tests: `internal/application`'s
idempotency-key guard (`replay`/`store`, now `reserveOrReplay`/`finalize`) was a
lookup-then-write-then-store sequence with a real TOCTOU gap — two concurrent requests sharing a
key could both miss the replay check and both execute the underlying use case. See "Exactly-once"
below for what replaced it.

## Guarantee tiers

### Exactly-once (CAS/fencing-enforced)

- **Workflow version transitions.** `WorkflowRepository.CommitTransition`
  (`internal/adapters/persistence/supabase/workflow_repo.go`) — `UPDATE workflows ... WHERE
  organization_id=$X AND workflow_id=$Y AND version=$expected`; `RowsAffected() != 1` →
  `ports.ErrConflict`. Proven: `TestWorkflowRepository_CommitTransition_StaleVersionRejected`
  (`adapters/persistence/supabase/integration_test.go`).
- **Execution claim.** `ExecutionRepository.ClaimDueIntents` — `FOR UPDATE SKIP LOCKED` makes two
  concurrent claimers' row-lock sets disjoint by construction, not merely unlikely to collide.
  Proven: `TestExecutionRepository_ClaimDueIntents_NoDuplicateUnderConcurrency`,
  `TestIntegration_Runtime_ConcurrentSweep_DispatchesExactlyOnce`.
- **Execution dispatch/terminal recording.** `RecordDispatched`/`RecordTerminal`
  (`execution_repo.go`) — both `UPDATE ... WHERE attempt_id=$X AND lease_fencing_token=$Y`; a
  reclaim (`ReclaimExpiredLeases`) atomically re-mints the token in the same `UPDATE` that flips
  status to `LEASE_EXPIRED`, fencing off any write from the worker that held the old token. Proven:
  `TestExecutionRepository_ReclaimExpiredLeases_BumpsFencingTokenSoStaleWorkerFailsClosed`,
  and this pass's `TestIntegration_Runtime_CompletionRacesLeaseExpiry_ExactlyOneWins` (genuinely
  unsynchronized race, not a forced ordering) and
  `TestIntegration_Runtime_RestartMidDispatch_FencedOffAfterSecondProcessCompletes`.
- **Approval resolution.** `PendingCommandRepository.ResolveApproval`
  (`adapters/persistence/supabase/pending_repo.go`) — `SELECT ... FOR UPDATE` then a Go-level
  status guard, same transaction as the decision write. Proven:
  `TestIntegration_ResolveApproval_ConcurrentResolutionOneWins` (5 real racing goroutines).
- **Idempotency-key reservation (this pass's fix).** `WorkflowRepository.IdempotencyReserve` —
  one atomic `INSERT ... ON CONFLICT (organization_id, idempotency_key) DO UPDATE ... WHERE
  outcome='IN_PROGRESS' AND created_at < $staleThreshold RETURNING outcome`. A row comes back from
  `RETURNING` exactly when this caller now owns the key (fresh insert or a stale-reservation
  reclaim); no row comes back when a live or terminal reservation already exists, in which case a
  second, purely informational `SELECT` decides what to tell the caller (never who owns the key —
  ownership was already decided atomically). Replaces the old `IdempotencyLookup`
  (`SELECT`)-then-domain-write-then-`IdempotencyStore` (`INSERT ... ON CONFLICT DO NOTHING`, error
  silently discarded) sequence, which had no atomicity between the lookup and the write: two
  concurrent callers could both miss the lookup and both execute the use case. Proven:
  `TestIntegration_CreateWorkflow_ConcurrentSameIdempotencyKey_OneWorkflow` (8 real racing
  goroutines; asserts exactly one real Workflow row created, verified by direct `SELECT count(*)`
  against `workflows`, not just in-memory `Result`s).

### At-least-once, idempotent handling required

- **Outbox event delivery.** `event_outbox` rows are written in the same transaction as the domain
  write (`insertEvents`, called inside `CreateWorkflow`/`CommitTransition` before `tx.Commit`) —
  structurally impossible for a notification to exist for a write that didn't commit, or vice
  versa; there is no race to synchronize because there is no separate step. Proven:
  `TestWorkflowRepository_CommitTransition_AtomicWithEvents` (positive case) and this pass's
  `TestWorkflowRepository_CreateWorkflow_RollbackLeavesNoOrphanEvents` (negative case: forces a
  same-transaction failure after one of two events already "inserted" pre-commit, proves neither
  survives the rollback). Delivery itself (outbox row → `realtime.Sweeper` → Supabase Realtime) is
  poll-driven, at-least-once, retried via `MarkPublishFailed` until `MarkPublished` succeeds.
  Proven at the repository level by `TestOutboxRepository_MarkPublishFailed_LeavesRowUnpublished`,
  and — new in this pass, closing a real gap (`internal/adapters/notify/realtime` had no test file
  at all) — at the `Sweeper` orchestration level by
  `TestIntegration_Sweeper_PublishFailure_RetriedOnNextSweep` and
  `TestIntegration_Sweeper_PermanentPublishFailure_DoesNotLoopForever`.
- **Dispatch retry after lease reclaim.** A reclaimed abandoned attempt is rescheduled
  (`ScheduleRetry`) up to the capability's `MaxAttempts`, then fails terminally — never silently
  dropped, never retried unboundedly. Proven: `TestIntegration_Runtime_WorkerCrash_ReclaimedAndRetried`,
  `TestIntegration_Runtime_LeaseReclaim_ExhaustsAfterMaxAttempts`.
- **Idempotency-key replay of a terminal outcome.** A retry with the *same* key after the
  reservation holder already called `finalize` gets that exact outcome reissued — cheaply (`Result{
  Outcome: ...}` only, no re-fetched `Workflow`/`ApprovalID`, a deliberate fast-path tradeoff, see
  `TestIntegration_CreateWorkflow_IdempotentReplayReturnsSameOutcome`'s own comment). A retry that
  races the still-live reservation gets `Indeterminate` and is expected to retry with the same key
  — this is a real signal, not an error, and the caller must not treat it as failure. **A different
  key for what a human considers "the same" request is never deduplicated** — that's the client's
  contract to uphold (reuse the key on retry), not something the server can infer. Proven:
  `TestIntegration_CreateWorkflow_DifferentIdempotencyKeys_TwoWorkflows` and
  `TestIntegration_CancelWorkflow_ApprovalResolutionRacesCommandRetry`.

### Best-effort, no durability guarantee

- **In-process `Notify` channel** (`Application.notify`) — non-blocking send, dropped silently if
  full or unset; the polling `Runtime.Sweep`/`Sweeper.Sweep` loops are the durable fallback for
  everything this hints at. Proven never load-bearing:
  `TestIntegration_Runtime_NotifierFailure_DoesNotBlockDispatch`.
- **`internal/concurrency.Bus`** — in-memory pub/sub, explicitly documented as best-effort ("a
  subscriber that isn't listening simply never sees it," `docs/audit/findings.md` §5). Not used by
  any governed write path.

## Known residual gaps

Named plainly, not silently left implicit — narrowed by this pass, not eliminated, matching this
repo's own precedent for partial fixes (`docs/audit/fixed-execution-lifecycle-hardening.md`'s
"Residual gap" section):

- **`IdempotencyFinalize` is not in the same DB transaction as the domain write.** A crash between
  the domain write committing and `finalize`'s `UPDATE` leaves the reservation stuck at
  `IN_PROGRESS`. A caller racing that key *within* `idempotencyReservationTTL` (30s) gets
  `Indeterminate` — safe, the original is presumed still in flight. A caller racing it *after* the
  TTL reclaims the key and re-executes the use case from scratch: safe in the common case (the
  original crashed before its domain write ever ran), but if the crash landed in the narrow window
  *after* the domain write committed and *before* `finalize` ran — no I/O separates them, so this
  requires a hard process kill at a specific instant, not a network blip, but it is not impossible —
  the reclaim-and-retry creates a second, independent domain effect (e.g. a second Workflow row).
  This pass narrows that exposure (from "any concurrent request during the whole use case,
  reliably reproducible" down to "a retry landing more than 30s after a crash in a sub-millisecond
  window") but does not fully eliminate it. Fully closing it requires folding `finalize` into the
  same transaction as `CreateWorkflow`/`CommitTransition`, touching those methods' signatures
  across all four call sites — deliberately out of scope this pass; see "Explicitly out of scope"
  in this fix's own record for the full reasoning.
- **The pre-existing `RecordTerminal`-success-then-`Application.SubmitResult`-crash window**
  (`docs/audit/gap-execution-lease-reclaim.md`'s "Residual gap," unchanged by this pass): a crash
  between an attempt's terminal status committing and the corresponding `SubmitResult` call
  completing leaves a `Result` row nothing automatically resubmits.
- **Lease renewal.** No `RenewLease` method exists anywhere in `ports.ExecutionRepository` — leases
  are fixed-duration from claim, never extended in place. Not a bug: there is no code path to race,
  so scenario 6 of the matrix below is structurally N/A rather than untested. If a renewal mechanism
  is ever added, it must be given the same fencing-token CAS `RecordDispatched`/`RecordTerminal`
  already use, not a separate ad hoc guard.
- **`realtime.Sweeper`'s retry path** was untested before this pass (no test file existed at all in
  `internal/adapters/notify/realtime`) — now covered, see "At-least-once" above.

## Test matrix

| # | Scenario | Guarantee proven | Test |
|---|---|---|---|
| 1 | Two workers claim the same execution | Exactly-once | `TestExecutionRepository_ClaimDueIntents_NoDuplicateUnderConcurrency`, `TestIntegration_Runtime_ConcurrentSweep_DispatchesExactlyOnce` |
| 2 | Two requests, same idempotency key | Exactly-once (this pass's fix) | `TestIntegration_CreateWorkflow_ConcurrentSameIdempotencyKey_OneWorkflow` |
| 3 | Two requests, different idempotency keys, same logical command | At-least-once by design — not deduplicated across keys | `TestIntegration_CreateWorkflow_DifferentIdempotencyKeys_TwoWorkflows` |
| 4 | Two concurrent approval resolutions | Exactly-once | `TestIntegration_ResolveApproval_ConcurrentResolutionOneWins` |
| 5 | Approval resolution racing command retry | Exactly-once (idempotency layer) + isolation from a concurrent unrelated transaction | `TestIntegration_CancelWorkflow_ApprovalResolutionRacesCommandRetry` |
| 6 | Lease renewal racing lease expiration | N/A — no renewal mechanism exists | documented above, no test (nothing to test) |
| 7 | Worker completion racing lease expiry | Exactly-once (fencing), genuinely unsynchronized race | `TestIntegration_Runtime_CompletionRacesLeaseExpiry_ExactlyOneWins` |
| 8 | Cancellation racing execution | Exactly-once (version CAS) under genuine in-flight overlap | `TestIntegration_Runtime_CancellationRacesInFlightDispatch_LateResultRejected` |
| 9 | Runtime notification racing DB commit | Structurally impossible (same transaction) | `TestWorkflowRepository_CommitTransition_AtomicWithEvents`, `TestWorkflowRepository_CreateWorkflow_RollbackLeavesNoOrphanEvents` |
| 10 | DB commit succeeds, notification fails | At-least-once, retried | `TestIntegration_Runtime_NotifierFailure_DoesNotBlockDispatch`, `TestIntegration_Sweeper_PublishFailure_RetriedOnNextSweep` |
| 11 | Notification succeeds, worker crashes before completion | No double side effect from a stale notification | `TestIntegration_Runtime_RestartMidDispatch_FencedOffAfterSecondProcessCompletes` |
| 12 | Process restart during execution | Exactly-once (fencing) across a real process boundary | `TestIntegration_Runtime_RecoveryAfterRestart_SecondRuntimeReclaimsAndCompletes` (pre-dispatch crash), `TestIntegration_Runtime_RestartMidDispatch_FencedOffAfterSecondProcessCompletes` (mid-dispatch crash) |

Every test above uses real goroutines/real Postgres, not `time.Sleep`-based timing, not a
retry-until-green assertion loop, and not a process-local mutex standing in for cross-process
coordination — races are forced deterministically via backdated `due_at`/`lease_expires_at`,
blocking channels (`fakeProvider.block`, `fakeNotifier.notified`), or, where the point is exactly
that no synchronization exists between two operations (scenario 7), left genuinely unsynchronized.

## Race-detector status

`go test -race` is unusable on this development machine: the `cgo.exe` link step fails with a
zero-diagnostic exit status 2 building GOROOT's own `runtime/cgo` bootstrap package, reproduced
fresh in this pass and identical to the finding already documented in
[`docs/audit/fixed-runtime-resilience.md`](../audit/fixed-runtime-resilience.md) (which itself
reconfirmed the same failure even after installing a working MSYS2 `gcc.exe` — this is toolchain
plumbing, not a missing compiler). Per that document's precedent, this pass substitutes
high-repetition real-concurrency runs (`go test ./... -count=3`) rather than attempting to fix the
toolchain gap, which remains a pre-existing environment limitation, not a blocker for correctness
work.

## Invariants

- A mechanism is only classified exactly-once here if a real-concurrency test (real goroutines,
  real Postgres) proves it — not because its code *looks* like it should be safe.
- A residual gap is named in this document's own section, not left for a future session to
  rediscover from scratch.
- Adding a new CAS/fencing mechanism to this codebase updates this document's exactly-once section
  in the same change, per `strategy.md`'s "no ad hoc testing convention" invariant.

## Dependencies

- [`strategy.md`](strategy.md)
- [`failure-injection.md`](failure-injection.md)
- [`docs/audit/findings.md`](../audit/findings.md) §2, §5
- [`docs/audit/gap-approval-flow-durability.md`](../audit/gap-approval-flow-durability.md) — the
  idempotency-key finding this pass closes the concurrent-duplicate-write portion of
- [`docs/audit/fixed-execution-lifecycle-hardening.md`](../audit/fixed-execution-lifecycle-hardening.md) —
  fencing-token mechanism this document's exactly-once claims build on
- [Application architecture](../architecture/application.md)
- [Runtime](../architecture/runtime.md)
- [Execution domain](../domain/execution.md)
