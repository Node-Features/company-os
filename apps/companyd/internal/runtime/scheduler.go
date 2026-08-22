package runtime

import "context"

// Scheduler is the contract Daemon depends on to supervise dispatch of
// already-persisted, Kernel-authorized work. *Runtime satisfies it today
// without any modification — see the compile-time assertion below.
//
// It deliberately has no Schedule method: per docs/architecture/kernel.md
// and docs/architecture/runtime.md, "scheduling" in the sense of deciding
// that work is due happens upstream, when Kernel's FinalizeStart produces an
// ExecutionIntent with a due_at time and Application persists it —
// Scheduler only discovers and dispatches intents that are already due and
// already durable. It has no Cancel method either: nothing in this
// codebase can cancel a pending or in-flight ExecutionIntent yet
// (ROADMAP.md Phase 1 Slice 6, CANCEL_WORKFLOW, is not implemented) —
// adding one here would document a capability that does not exist. "Status"
// is likewise not a Scheduler-level query today; the closest existing thing
// is ports.ExecutionRepository.GetLatestResult, queried directly.
//
// Contract: Start claims and dispatches due work in its own goroutines,
// exactly once each, until ctx is cancelled. Start does NOT guarantee
// dispatch ordering across intents — multiple due intents may run
// concurrently, in any order — and does NOT guarantee a dispatch completes
// before the next poll tick fires. Start does NOT retry internally; retry
// is a property of the persisted ExecutionIntent/attempt record
// (docs/domain/execution.md), driven by a later Sweep reclaiming it, not by
// calling Start again. Sweep runs exactly one claim-and-dispatch pass and
// returns once dispatches are launched, not once they finish — a returned
// Sweep call gives no completion guarantee for the work it started. Wait
// blocks until every dispatch launched by the most recently returned Sweep
// call has finished; it does NOT wait for dispatches launched by a Sweep
// that starts after Wait is called, so relying on it is only safe during
// shutdown, after the Start loop has already stopped accepting new ticks.
type Scheduler interface {
	// Start runs the poll-and-wake dispatch loop until ctx is cancelled.
	Start(ctx context.Context)

	// Sweep claims currently-due work and dispatches it once, outside the
	// regular poll interval — used to react immediately to an external
	// wake-up signal instead of waiting for the next tick.
	Sweep(ctx context.Context)

	// Wait blocks until every dispatch launched by the most recent Sweep
	// has returned.
	Wait()
}

// *Runtime already satisfies Scheduler without modification — this line is
// a compile-time proof of that, not new behavior.
var _ Scheduler = (*Runtime)(nil)
