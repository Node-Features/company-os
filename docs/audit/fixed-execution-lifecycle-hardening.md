# Fixed: Execution Lifecycle Hardening (Lease Reclaim + Shutdown Safety)

Status: IMPLEMENTED (2026-08-24) — implemented and empirically verified against the real database, including under deliberate concurrent-process stress. **Not yet committed to git** as of this writing — check `git status` before assuming this is landed on a branch.

This is the implementation record for [`gap-execution-lease-reclaim.md`](gap-execution-lease-reclaim.md) (P0), expanded into a full execution-lifecycle hardening pass covering all ten hard invariants named in the work's own brief. Read [`gap-execution-lease-reclaim.md`](gap-execution-lease-reclaim.md) first for the original problem statement and plan; this document is the "what actually shipped, what it proves, what's still open" record, in the same shape as [`fixed-test-suite-nondeterminism.md`](fixed-test-suite-nondeterminism.md).

## Lifecycle stage → code, as found

The target model — Goal → Command → Governance → Domain transition → Durable event → Execution intent → Lease → Execution attempt → Checkpoint → Result → Evidence → Evaluation — mapped to real code before this work:

| Stage | Code |
|---|---|
| Goal → Command | `internal/adapters/httpapi/*.go` → `application.Application` methods, each building a `command.WorkflowCommandEnvelope` |
| Governance | `application.evaluateGovernance` ([pipeline.go](../../apps/companyd/internal/application/pipeline.go)) → `governance.Evaluate` — decision always persisted before any branch |
| Domain transition | `internal/kernel/workflow` — pure `Validate*Proposal`/`Finalize*`, no I/O |
| Durable event | `ports.AuthoritativeStateRepository.CommitTransition` — one transaction: workflow row + `domain_events` + `event_outbox` |
| Execution intent | Created inside that same `CommitTransition` call |
| Lease | `ExecutionRepository.ClaimDueIntents` — `FOR UPDATE SKIP LOCKED`; `lease_expires_at`/`lease_fencing_token` already existed in schema, but nothing ever read `lease_expires_at` back to reclaim anything |
| Execution attempt | `execution.ExecutionAttempt` row, created per claim |
| Checkpoint | `execution_attempts.checkpoint jsonb` exists, deliberately unused — this slice's one capability is a single bounded call with no partial progress to checkpoint (confirmed in the type's own doc comment; not invented here) |
| Result | `Runtime.execute` → `SaveResult`, always persisted regardless of outcome |
| Evidence | The `Result` + `domain_events` + `governance_decisions` rows themselves |
| Evaluation | `Application.SubmitResult` → Kernel `Finalize*` → `CommitTransition`'s version CAS — the one place external provider output is validated before becoming Workflow state |

**Confirmed gap**: Lease existed in schema and domain model (`execution.StatusLeaseExpired`, `execution.AttemptStatus.CanTransitionTo`) with zero enforcing code. `ClaimDueIntents` only ever selected `PENDING`. A crash or routine deploy between claim and terminal report stranded work permanently; `Runtime.Start`'s shutdown reused the same cancellable context for in-flight dispatches as for the poll loop itself.

## What was implemented

1. **`ports.ExecutionRepository.ReclaimExpiredLeases`** ([execution_repo.go](../../apps/companyd/internal/adapters/persistence/supabase/execution_repo.go)) — one atomic `FOR UPDATE SKIP LOCKED` + `UPDATE ... RETURNING`, mirroring `ClaimDueIntents`'s exact shape. Transitions expired `CLAIMED`/`DISPATCHED`/`WAITING` attempts to `LEASE_EXPIRED` **and reassigns `lease_fencing_token` in the same statement** — the actual mechanism that makes reclaim safe (see invariant 8 below).
2. **`RecordDispatched`/`RecordTerminal` now return `ports.ErrConflict`** on a fencing-token mismatch (previously a bespoke `errors.New`), consistent with every other compare-and-swap in this codebase.
3. **`Runtime.reclaimAbandoned`**, called every `Sweep`: decides retry-vs-exhausted using the real `RetryPolicy.MaxAttempts` (a lease-expiry retry consumes the same attempt budget a provider failure does — resolves this gap's own "does a reclaim count against MaxAttempts?" open question: yes), reusing the existing backoff computation; when exhausted, submits a synthetic `FAILED` Result through the real governed `SubmitResult` path so the Workflow reaches `FAILED` rather than being left dangling. Deliberately does **not** re-mark the attempt `FAILED_TERMINAL` — it's already correctly terminal at `LEASE_EXPIRED`, which is the truthful status (never heard from again, not "failed").
4. **Shutdown-safe dispatch context**: `Runtime` gained `workCtx`/`workCancel` (lazily, race-safely initialized via `sync.Once`), independent of the `ctx` passed to `Start`. `Sweep`'s claim/reclaim queries still honor shutdown (correct — stop accepting new work); dispatched goroutines run on `workCtx` instead, so SIGTERM no longer aliases into every in-flight dispatch. `Daemon.Shutdown` calls the new `StopWork()` only once its bounded drain wait times out — narrowing, not eliminating, the abandonment window to the shutdown deadline itself; anything still orphaned past that is recovered by the same reclaim mechanism on next boot, same as a hard crash.
5. **A real latent bug fixed along the way, not originally in scope**: `Sweep`'s pre-existing `ClaimDueIntents` call hardcoded the package-level `fixtures.OrganizationID` constant instead of `r.Fixtures.Organization().OrganizationID` — invisible in production (one org exists today, both values identical) but made `Runtime` impossible to test in per-test-org isolation. Fixed in both the pre-existing call and the new one; zero production behavior change.

## Hard invariants — verified, not asserted

| # | Invariant | How it holds |
|---|---|---|
| 1 | DB state authoritative | Every transition is a committed Postgres row; nothing in-memory is treated as truth |
| 2 | Events durable | `domain_events`/`event_outbox` written in the same transaction as the state change (pre-existing, confirmed still true: `TestWorkflowRepository_CommitTransition_AtomicWithEvents`) |
| 3 | Runtime notifications are hints, not truth | `notifyChanged` errors are logged, never surfaced — `TestIntegration_Runtime_NotifierFailure_DoesNotBlockDispatch` (new) |
| 4 | Redis never authoritative | N/A — zero Redis code exists anywhere in this codebase |
| 5 | External provider state validated before becoming org state | `execute()` → `SaveResult` (evidence) → `SubmitResult` → Kernel `Finalize*` → `CommitTransition`; no direct path exists, confirmed by full read of `runtime.go` |
| 6 | Retries don't duplicate effects | `SubmitResult`'s idempotency replay (`TestIntegration_Runtime_DuplicateSubmitResult_Idempotent`, new); fencing token makes a reclaimed worker's late write fail before it ever reaches `SubmitResult` |
| 7 | Lost workers don't permanently strand work | The core fix — proven across a real process boundary (`TestIntegration_Runtime_RecoveryAfterRestart_SecondRuntimeReclaimsAndCompletes`, new). **One narrow residual gap remains — see below.** |
| 8 | Concurrent workers can't both claim the same execution | `FOR UPDATE SKIP LOCKED` for both claim and reclaim (`TestExecutionRepository_ReclaimExpiredLeases_ConcurrentReclaimersNoDuplicate`, new); fencing-token bump closes the "already-reclaimed zombie writes anyway" gap (`TestExecutionRepository_ReclaimExpiredLeases_BumpsFencingTokenSoStaleWorkerFailsClosed`, new) |
| 9 | Stale command can't silently overwrite newer state | Version CAS in `CommitTransition`; proven specifically in the cancellation context by `TestIntegration_Runtime_CancellationDuringDispatch_LateResultRejected` (new) |
| 10 | Every meaningful transition has evidence for reconstruction | `Result`/`GovernanceDecision`/`domain_events` rows always written before any Workflow-level decision — this is what makes the residual gap below "recoverable by inspection," not silent data loss |

This closes two items on `findings.md` §2's "does not hold" list: **item 1** (lease-based reclaim) is now implemented and verified; **item 4** (`CanTransitionTo` never invoked) now has real enforcement — the `ReclaimExpiredLeases` WHERE clause is hand-matched to the exact FROM-state set the table legalizes a transition to `LEASE_EXPIRED` from, so the same transition table's specification governs the SQL even though it isn't a literal Go function call (a literal call isn't possible while keeping the check-and-transition atomic in one statement).

## State-transition diagram

```
                    ┌─────────────────────────────────────────────────────┐
                    │              ExecutionAttempt (per-claim)             │
                    └─────────────────────────────────────────────────────┘

        ClaimDueIntents                    RecordDispatched         provider call
   PENDING ──────────────► CLAIMED ──────────────────► DISPATCHED ──────┬────────► SUCCEEDED
   (intent)                  │                              │           │        RecordTerminal
                              │                              │           ├──► FAILED_RETRYABLE
                              │                              │           │    (attempt < MaxAttempts)
                              │                              │           │       └─ ScheduleRetry ─► intent PENDING again
                              │                              │           │
                              │                              │           └──► FAILED_TERMINAL
                              │                              │                (not retryable, or exhausted)
                              │                              │
                              │                              ▼
                              │                          WAITING ──► DISPATCHED | CANCELLED | LEASE_EXPIRED
                              │
              lease_expires_at < now()        lease_expires_at < now()
                              │                              │
                              ▼                              ▼
                    ┌───────────────────────────────────────────────┐
                    │   LEASE_EXPIRED  (terminal — fencing token     │
                    │   bumped here; old worker's late write now     │
                    │   fails RecordDispatched/RecordTerminal with   │
                    │   ports.ErrConflict, never reaches SubmitResult)│
                    └───────────────────────────────────────────────┘
                                       │
                        attempt_number < MaxAttempts?
                              ┌────────┴────────┐
                             yes                no
                              │                  │
                    ScheduleRetry          failExhausted:
                    intent → PENDING       SaveResult(FAILED) →
                    (new attempt on        Application.SubmitResult →
                     next Sweep)           Workflow → FAILED


        ┌──────────────────────────────────────────────────────────────┐
        │                    ExecutionIntent                            │
        └──────────────────────────────────────────────────────────────┘

   (created by CommitTransition on START_WORKFLOW)
        PENDING ──ClaimDueIntents──► CLAIMED ──(attempt resolves)──► [stays CLAIMED — matches
           ▲                                                          the pre-existing pattern for
           │                                                          a normally-completed attempt;
           └──────────ScheduleRetry (retry, or reclaim-then-retry)────┘ see note below]

   (CLOSED only via CommitTransition's bulk-close on an owning
    Workflow ACCEPT/REJECT/CANCEL transition — pre-existing, unchanged)
```

`execution_intents` staying `CLAIMED` after a normally-resolved attempt matches the codebase's own pre-existing pattern (never explicitly flipped to `CLOSED` outside the Workflow-cancel bulk-close path) — harmless, since `ClaimDueIntents` only ever selects `PENDING`; not changed here.

## Correctness after process failure, transition by transition

| Crash point | What's on disk | Recovery |
|---|---|---|
| Before `CommitTransition` commits | Nothing (or an orphaned `GovernanceDecision`, pre-existing/harmless) | Client retry is a fresh, clean attempt |
| Inside `CommitTransition`'s transaction | Postgres transaction atomicity — all-or-nothing | N/A, can't half-happen |
| After commit, before Runtime ever sees the intent | Intent durably `PENDING` | Next `Sweep` (this process or a fresh one) claims it — ordinary operation |
| After claim commits, before `RecordDispatched` | Attempt `CLAIMED`, lease ticking | **`reclaimAbandoned` finds it once the lease expires** — this is the fix |
| After `RecordDispatched`, during/after the provider call, before `RecordTerminal` | Attempt `DISPATCHED` | Same reclaim path — WHERE clause and legal-transition table don't distinguish `CLAIMED` from `DISPATCHED` |
| Original (zombie) worker wakes up later and calls back anyway | Fencing token already bumped by reclaim | Fails closed with `ports.ErrConflict` — proven directly |
| Two reclaimers race on the same expired lease | `FOR UPDATE SKIP LOCKED` | Exactly one wins the row — proven directly |
| After `SaveResult`, before `RecordTerminal` | Result row persisted (evidence, harmless if orphaned), attempt still `DISPATCHED` | Reclaim still finds and recovers it |
| **After `RecordTerminal`, before `App.SubmitResult`** | Attempt correctly terminal, Result fully persisted, Workflow never actually transitioned | **Not automatically recovered — see below** |
| SIGTERM during a routine deploy, mid-dispatch | In-flight goroutine now on `workCtx`, not the poll loop's `ctx` | `Daemon.Shutdown` gives it up to its bound to finish normally; if it doesn't, `StopWork()` cancels it and lease-based reclaim recovers it on next boot, same as a hard crash |

## Residual gap, deliberately not fixed here

A process crash in the window between `RecordTerminal` succeeding and `Application.SubmitResult` completing — a handful of function calls, no further I/O in the success path — leaves a fully-evidenced but never-submitted `Result`. The attempt is genuinely terminal, so `ReclaimExpiredLeases` correctly won't touch it, and nothing today automatically retries the `SubmitResult` call. Invariant 10 holds (the `Result` row has everything needed to manually reconstruct); invariant 7 isn't fully closed for this one narrow case. A periodic scan for `results` rows with `accepted IS NULL` past a grace period, replaying `SubmitResult`, would close it — not built here, since it's a materially different, smaller mechanism deserving its own justification rather than folding it in speculatively. Recommended as a future increment.

## Files changed

`ports/persistence.go` (interface) · `adapters/persistence/supabase/execution_repo.go` (`ReclaimExpiredLeases`, `ErrConflict` fix) · `runtime/runtime.go` (`workCtx`/`StopWork`/`reclaimAbandoned`/`failExhausted`, org-scoping fix) · `daemon/daemon.go` (`StopWork` call on drain timeout) · `application/fake_repo_test.go` (interface-satisfaction addition, no behavior change) · new: `runtime/fakes_test.go`, `runtime/runtime_test.go`, `runtime/runtime_unit_test.go` · extended: `adapters/persistence/supabase/integration_test.go`.

## Tests added

Mapped to the twelve requested scenarios — normal execution, duplicate delivery, worker crash, lease expiration, concurrent claim, retry, provider failure, database failure, runtime notification failure, stale version, cancellation, recovery after restart — each covered by name in `internal/runtime/runtime_test.go` / `runtime_unit_test.go`, plus a thirteenth (`StopWork` cancels an in-flight dispatch) proving the shutdown-mechanics fix directly, and four persistence-layer tests (atomicity, fencing-token-bump proof, non-expired-untouched, concurrent-reclaimers-no-duplicate) in `adapters/persistence/supabase/integration_test.go`.

Two test-writing mistakes worth recording, both caught by running the tests rather than trusting the reasoning: an early cancellation-test assertion expected `Conflict` where the real, correct outcome is `Rejected` (the Kernel's state check catches a fully-completed cancellation before the version CAS is ever reached — same "two individually-correct rejection categories" class of finding the 2026-08-24 test-determinism work already documented, just deterministic here rather than a race); and two tests asserted `due_at` lands strictly in the future after a reclaim-triggered retry, which is flaky by construction since `computeBackoff`'s full jitter can legitimately return 0 — both fixed by asserting the real invariant (`PENDING`, not the specific jittered timing) instead.

**Verified**: all 17 new tests pass individually, together, and under two fully concurrent process-level stress runs (real DB, adversarial concurrency — same standard as [`fixed-test-suite-nondeterminism.md`](fixed-test-suite-nondeterminism.md)). `go test -race` remains unavailable in this environment (pre-existing cgo/gcc toolchain gap, documented there and in ADR-0007's own notes) — high-repetition and concurrent real-process stress substituted where it would normally apply. Full `go test ./...`: every package green, exit 0.

## Dependencies

- [`gap-execution-lease-reclaim.md`](gap-execution-lease-reclaim.md) — the original problem statement and plan this implements
- [`fixed-test-suite-nondeterminism.md`](fixed-test-suite-nondeterminism.md) — the fresh-per-test-organization pattern this work's tests reuse directly (`requireRealRuntime`)
- [`findings.md`](findings.md) §2 (invariants 1 and 4 resolved), §3 (failure-mode matrix rows 1-2)
- [`domain/execution.md`](../domain/execution.md), [`architecture/runtime.md`](../architecture/runtime.md), [`architecture/daemon.md`](../architecture/daemon.md)
- [`gap-ci-integration-coverage.md`](gap-ci-integration-coverage.md) — these fixes have no CI regression protection until that gap closes
