# Gap: Runtime Dispatch Resilience

Status: APPROVED (2026-08-24) — problem statement and remediation plan approved. Implementation still requires the project owner to explicitly select and authorize this slice before any code changes, per this repository's doc-gate convention.

Severity: P1 — threatens runtime reliability. See [`findings.md`](findings.md) §3 rows 8-9.

## Problem

`Runtime.Sweep` claims a batch of due intents (batch size 10, [runtime.go](../../apps/companyd/internal/runtime/runtime.go)) and spawns one goroutine per claim: `go func(c execution.ClaimedExecution) { defer r.wg.Done(); r.execute(ctx, c.Attempt, c.Intent) }(c)`. Two gaps:

1. **No panic recovery.** A repo-wide grep for `recover()` across `internal/runtime`, `internal/daemon`, and `main.go` found none. Go's default behavior on an unrecovered goroutine panic is to crash the entire process — a single malformed provider response or an unhandled edge case in one dispatch takes down every other in-flight intent along with the daemon itself, compounding [`gap-execution-lease-reclaim.md`](gap-execution-lease-reclaim.md)'s orphaning (nothing was recording terminal state for any of them either).
2. **No concurrency bound.** `Sweep` fires on a `PollInterval` timer (5s, `main.go`) and on every `Wakeup` signal, with no coordination preventing overlapping sweeps. Provider timeout is 30s (`fixtures/firstslice.go`). Under a slow-provider window, multiple sweeps can pile up with no worker pool or semaphore limiting total concurrent provider calls or DB connections in flight.

## Invariant

Restores: a single bad dispatch cannot take down the daemon; dispatch concurrency is bounded and predictable under load, not a function of how many sweeps happen to overlap.

## Proposed approach (plan-level only)

1. Wrap `execute()`'s goroutine body in a `recover()` that logs the panic, marks the corresponding attempt/intent in a state [`gap-execution-lease-reclaim.md`](gap-execution-lease-reclaim.md)'s reclaim mechanism can pick up (rather than leaving it silently stuck), and lets the daemon keep running.
2. Introduce a bounded worker pool or semaphore around dispatch — sized as a config value, not hardcoded — so `Sweep` claims work but execution concurrency has a real ceiling independent of poll cadence.
3. Consider whether overlapping `Sweep` calls themselves need coordination (e.g., a simple in-flight-sweep guard) or whether the concurrency bound in step 2 makes that unnecessary — a design choice for whoever implements this, not fixed here.

## Files likely to change

- [`apps/companyd/internal/runtime/runtime.go`](../../apps/companyd/internal/runtime/runtime.go)
- [`apps/companyd/cmd/companyd/main.go`](../../apps/companyd/cmd/companyd/main.go) — if the concurrency bound becomes a configured value.

## Tests required

**Before (regression baseline):** a test that injects a panic into one dispatch and confirms today's actual behavior — the process/test harness crashes rather than recovering.

**After:**
- Injected panic in one dispatch: daemon survives, other concurrently-dispatching intents complete normally, the panicked intent lands in a state the reclaim mechanism will pick up.
- A load test asserting concurrent provider calls never exceed the configured bound even when many intents are simultaneously due.

## Dependencies

- [`findings.md`](findings.md) §3 rows 8-9.
- [`testing/failure-injection.md`](../testing/failure-injection.md)
- [`architecture/runtime.md`](../architecture/runtime.md)
- [`adr/ADR-0007-concurrency-model.md`](../adr/ADR-0007-concurrency-model.md) (Status: PROPOSED)
- Related: [`gap-execution-lease-reclaim.md`](gap-execution-lease-reclaim.md) — a panicked dispatch needs the same reclaim path a crashed/orphaned intent needs; implementing both together avoids two separate "stuck intent" recovery mechanisms.
