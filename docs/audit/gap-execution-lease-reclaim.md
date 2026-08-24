# Gap: Execution-Intent Lease Reclaim

Status: IMPLEMENTED (2026-08-24) — implemented and empirically verified against the real database, including under deliberate concurrent-process stress. **Not yet committed to git** as of this writing — check `git status` before assuming this is landed on a branch. See [Implementation](#implementation-2026-08-24) below for what actually shipped, how it differs from the plan, and the resolved open questions.

Severity: P0 — threatens correctness/data integrity. The single highest-confidence finding of the 2026-08-24 audit; independently confirmed by two separate investigation passes (persistence and runtime). See [`findings.md`](findings.md) §1 ("Execution intents / Leases"), §3 rows 1-2.

The rest of this document (problem statement, original plan) is left as originally written, for the historical record of what was found and proposed before implementation — read [Implementation](#implementation-2026-08-24) for what's actually true of the code today.

## Problem

The execution model names Lease as a first-class step (Execution intent → **Lease** → Execution attempt → Checkpoint → Result). The schema anticipates it (`execution_attempts.lease_expires_at`, indexed via `execution_attempts_lease_idx WHERE status IN ('CLAIMED','DISPATCHED','WAITING')`, first-slice migration). The domain model defines the transition (`execution.StatusLeaseExpired`, `execution.AttemptStatus.CanTransitionTo`, [types.go:22-46](../../apps/companyd/internal/domain/execution/types.go)). None of it runs.

`ClaimDueIntents` ([execution_repo.go:24-123](../../apps/companyd/internal/adapters/persistence/supabase/execution_repo.go)) only ever selects `status='PENDING' AND due_at<=now()`. It never re-selects `CLAIMED`/`DISPATCHED` rows whose lease has expired. `CanTransitionTo` has zero call sites anywhere in the repository — the entire legal-transition table is unenforced documentation.

Two independent ways to reach the same stuck state:
- **Crash**: process dies after the claim transaction commits (`execution_repo.go:37-41`) but before `RecordTerminal`/`SaveResult` runs.
- **Routine deploy**: `main.go`'s shutdown handler reuses the same root `ctx` (already `Done()` at signal time) for in-flight work. `Runtime.Start`'s claim loop does stop cleanly (`runtime.go:76-77`), but any DB/provider call already in flight — `AuthorizeDispatch`'s `LoadWorkflow`, `RecordDispatched`, `RecordTerminal`, `SaveResult`, the provider call itself — executes against a cancelled context and is likely to fail via `context.Canceled` mid-write. `daemon.Shutdown` only waits on a `sync.WaitGroup` bounded by a 5s timeout ([daemon.go:30-42](../../apps/companyd/internal/daemon/daemon.go)); it does not stop new work from starting mid-drain or give in-flight work an uncancelled context to finish on.

Either path leaves `execution_intents.status='CLAIMED'` permanently. The owning Workflow stays `READY` forever with no operator-visible signal (no metrics/tracing exist — see [`gap-ci-integration-coverage.md`](gap-ci-integration-coverage.md) and [`backlog-p2-p4.md`](backlog-p2-p4.md) for the observability gap).

## Invariant

Restores: "Lease" as a real, reclaimable step in the execution model, and "does not assume the in-flight attempt stopped immediately" (already stated for `CANCEL_WORKFLOW` in [`domain/execution.md`](../domain/execution.md) — the same caution needs to extend to crash/shutdown recovery generally, not just explicit cancellation).

## Proposed approach (plan-level only)

1. A reclaim query: select `execution_intents`/`execution_attempts` rows in `CLAIMED`/`DISPATCHED`/`WAITING` whose `lease_expires_at < now()`, transition the attempt to `LEASE_EXPIRED` (using the already-defined `CanTransitionTo` table — finally give it a caller), and make the underlying intent claimable again (either directly, or by feeding it back through the existing retry/`ScheduleRetry` path, subject to `MaxAttempts`).
2. Run this reclaim as part of (or alongside) the existing `Sweep` cycle, or as its own periodic pass — a design choice for whoever picks this up, not fixed here.
3. Shutdown: give `Runtime.Start`'s claim loop a cancellable context tied to shutdown, but let already-claimed, in-flight `execute()` calls run to completion (or to their own dispatch timeout) on an *uncancelled* context, bounded by `daemon.Shutdown`'s existing wait — so a graceful deploy drains rather than orphans. Anything still in flight past the drain deadline should be left in a state the reclaim mechanism from step 1 will pick up on the next boot, not silently abandoned.
4. Decide and document a fencing-token policy: `execution_attempts.lease_fencing_token` already exists ([execution_repo.go:126-148](../../apps/companyd/internal/adapters/persistence/supabase/execution_repo.go)) and is checked on `RecordDispatched`/`RecordTerminal` — confirm the reclaim path bumps or respects this token so a reclaimed intent's original (zombie) claimer can never win a race to write a terminal result after reclaim.

## Files likely to change

- [`apps/companyd/internal/adapters/persistence/supabase/execution_repo.go`](../../apps/companyd/internal/adapters/persistence/supabase/execution_repo.go) — new reclaim query.
- [`apps/companyd/internal/runtime/runtime.go`](../../apps/companyd/internal/runtime/runtime.go) — invoke reclaim; separate shutdown-cancel context from in-flight-work context.
- [`apps/companyd/internal/daemon/daemon.go`](../../apps/companyd/internal/daemon/daemon.go) — drain semantics.
- [`apps/companyd/cmd/companyd/main.go`](../../apps/companyd/cmd/companyd/main.go) — context wiring at signal handling.
- [`apps/companyd/internal/domain/execution/types.go`](../../apps/companyd/internal/domain/execution/types.go) — give `CanTransitionTo` a real caller (no type change expected; it should already fit).

## Tests required

**Before (regression baseline, expected to fail today):** an integration test that claims an intent (via `ClaimDueIntents`), never calls `RecordTerminal`, backdates `lease_expires_at`, and asserts the intent is *not* currently reclaimable by any existing mechanism.

**After:**
- The same scenario, now asserting the abandoned intent becomes claimable again exactly once (mirror the existing concurrency shape of `TestExecutionRepository_ClaimDueIntents_NoDuplicateUnderConcurrency`).
- A process-level or simulated-shutdown test: send the shutdown signal mid-dispatch, assert the in-flight write either completes or the intent lands in a state the reclaim mechanism will pick up — never a silent, undetectable orphan.
- A fencing-token test proving a reclaimed intent's original claimer cannot write a terminal result after reclaim.

## Dependencies

- [`findings.md`](findings.md) §1, §2 (invariant 1 in the "does not hold" list), §3 rows 1-2.
- [`domain/execution.md`](../domain/execution.md)
- [`architecture/runtime.md`](../architecture/runtime.md), [`architecture/daemon.md`](../architecture/daemon.md)
- [`adr/ADR-0007-concurrency-model.md`](../adr/ADR-0007-concurrency-model.md) (Status: PROPOSED — relevant, not yet APPROVED; this gap doc does not depend on it being approved first, but a reclaim design should stay consistent with it).

## Open questions (as originally written — see resolutions below)

- Does a reclaimed intent count against `MaxAttempts`, or is a lease-expiry retry free (distinct from a provider-failure retry)? Not answered by any existing doc.
- Single-process topology today (ADR-0004) means only one Runtime ever claims — reclaim logic should still be written for the eventual multi-process case per `architecture/node.md`'s open question, but should not block on it.

## Implementation (2026-08-24)

Resolved open questions: a reclaimed intent **does** count against `MaxAttempts` — a lease-expiry retry shares the same `attempt_number`/`RetryPolicy.MaxAttempts` budget a provider-failure retry does, so an environment that can never actually complete a dispatch (not just an unlucky provider) still terminates in `FAILED` rather than retrying forever. Reclaim logic **is** written multi-process-safe (`FOR UPDATE SKIP LOCKED` + fencing-token bump), verified directly under two genuinely concurrent `companyd`-shaped processes even though only one runs in production today.

**`ports.ExecutionRepository.ReclaimExpiredLeases(ctx, orgID, limit)`** ([persistence.go](../../apps/companyd/internal/ports/persistence.go)), implemented in [execution_repo.go](../../apps/companyd/internal/adapters/persistence/supabase/execution_repo.go): one atomic `SELECT ... FOR UPDATE SKIP LOCKED` + `UPDATE ... RETURNING`, mirroring `ClaimDueIntents`'s existing shape exactly. Transitions each found `CLAIMED`/`DISPATCHED`/`WAITING` + lease-expired attempt to `LEASE_EXPIRED` — finally giving `execution.AttemptStatus.CanTransitionTo`'s table a real (if indirect — matched via the WHERE clause's status set, not a literal function call, since the transition must happen atomically in SQL) enforcement — and, critically, **reassigns `lease_fencing_token` in the same statement**. This is the actual safety mechanism for the "reclaimed intent's original claimer cannot win a race" requirement: the original worker is never forcibly stopped (Go cannot do that), but its old token no longer matches the row, so `RecordDispatched`/`RecordTerminal` fail closed (now returning `ports.ErrConflict`, fixed alongside this from a bespoke error type) if it calls back after being reclaimed.

`Runtime.reclaimAbandoned` ([runtime.go](../../apps/companyd/internal/runtime/runtime.go)) calls this every `Sweep`, then — using `capability.RetryPolicy` from `Fixtures` (a Go value the persistence layer correctly has no business deciding with) — either `ScheduleRetry`s the intent (attempts remain) or synthesizes a `FAILED` `Result` and calls `Application.SubmitResult` directly (attempts exhausted), deliberately *not* re-marking the attempt `FAILED_TERMINAL` — it's already correctly terminal at `LEASE_EXPIRED`, and that status is the truthful one (never heard from again, not "failed").

Shutdown: `Runtime` gained a `workCtx`/`workCancel` pair (lazily, race-safely initialized via `sync.Once`) independent of the `ctx` passed to `Start`. `Sweep`'s claim/reclaim queries still use the shutdown-bound `ctx` (correct — decline new work immediately); dispatched goroutines run on `workCtx` instead, so a SIGTERM's cancellation no longer aliases into every in-flight dispatch. `Daemon.Shutdown` calls the new `Runtime.StopWork()` only once its bounded wait already timed out, narrowing (not eliminating) the abandonment window to the shutdown deadline itself — anything still orphaned past that is recovered by the same reclaim mechanism on next boot, same as a hard crash.

One latent bug found and fixed *because of* this work, not originally in scope: `Runtime.Sweep`'s existing `ClaimDueIntents` call (pre-existing code, not newly written) hardcoded the package-level `fixtures.OrganizationID` constant rather than reading `r.Fixtures.Organization().OrganizationID` — invisible in production (both values are currently identical, one org exists) but made Runtime impossible to test in per-test-org isolation. Fixed in both the pre-existing `Sweep` call and the new `reclaimAbandoned` call for consistency; zero production behavior change.

**Verified:** new tests in [`internal/adapters/persistence/supabase/integration_test.go`](../../apps/companyd/internal/adapters/persistence/supabase/integration_test.go) (atomicity, fencing-token bump proof, non-expired-untouched, concurrent-reclaimers-no-duplicate) and [`internal/runtime/runtime_test.go`](../../apps/companyd/internal/runtime/runtime_test.go) (13 real-database scenario tests plus 2 pure-Go database-failure unit tests) — all passing individually, together, and under two fully concurrent process-level stress runs. Full details, root-cause reasoning, the state-transition diagram, and the failure-correctness walkthrough are now their own doc: [`fixed-execution-lifecycle-hardening.md`](fixed-execution-lifecycle-hardening.md).

**Known residual gap, not fixed here (small, explicitly out of scope):** a process crash in the narrow window between `RecordTerminal` succeeding and `Application.SubmitResult` completing (a handful of function calls, no additional I/O between them in the success path) leaves a fully-evidenced but never-submitted `Result` — the attempt is correctly terminal so `ReclaimExpiredLeases` will never re-touch it, and nothing else currently retries the `SubmitResult` call. Invariant 10 (evidence sufficient for reconstruction) holds — the `Result` row has everything needed — but invariant 7 (lost workers must not permanently strand work) is not fully closed for this one specific, much narrower window. Recommended as a future increment (a periodic scan for `results` rows with `accepted IS NULL` past some grace period, replaying `SubmitResult`), not implemented now — it's a different, smaller mechanism than lease reclaim and deserves its own justification rather than being folded in speculatively.
