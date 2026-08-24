package runtime

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/google/uuid"
)

// TestSweep_DatabaseFailure_DoesNotPanic is the one scenario in this
// package that deliberately does NOT use a real database: reliably making
// a real Postgres connection fail on command, mid-test, is itself a
// flakiness risk (exactly what this project's 2026-08-24 test-determinism
// work eliminated elsewhere) — a pure-Go fake that always errors is the
// correct, deterministic tool here, matching this package's own discipline
// of faking only genuinely-hard-to-control dependencies (see fakeProvider's
// doc comment). This proves Sweep and reclaimAbandoned both degrade
// gracefully — log and return — when the database becomes unreachable,
// never panicking and never touching Application (App is left nil; if
// either method incorrectly tried to use it, this test would panic on a
// nil pointer dereference, which is exactly the failure mode being ruled
// out).
func TestSweep_DatabaseFailure_DoesNotPanic(t *testing.T) {
	rt := &Runtime{
		Exec:          fakeFailingExec{err: errors.New("simulated: connection refused")},
		App:           nil,
		Provider:      &fakeProvider{},
		Fixtures:      fixtures.NewRegistryWithOrganization(uuid.New()),
		PollInterval:  time.Hour,
		LeaseDuration: time.Minute,
		Wakeup:        make(chan uuid.UUID, 1),
		Notifier:      &fakeNotifier{},
	}
	rt.ensureWorkContext()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Sweep panicked on database failure: %v", r)
		}
	}()
	rt.Sweep(context.Background())
	rt.Wait()
}

// TestReclaimAbandoned_DatabaseFailure_DoesNotBlockClaiming proves the two
// halves of Sweep are independently resilient: a failure in
// ReclaimExpiredLeases must not prevent ClaimDueIntents from still being
// attempted on the same Sweep call (a transient reclaim-query error
// shouldn't stop normal dispatch). Uses a fake that fails only
// ReclaimExpiredLeases and returns an empty, successful claim otherwise.
func TestReclaimAbandoned_DatabaseFailure_DoesNotBlockClaiming(t *testing.T) {
	claimCalled := false
	f := &partialFailingExec{
		reclaimErr: errors.New("simulated: reclaim query failed"),
		onClaim:    func() { claimCalled = true },
	}
	rt := &Runtime{
		Exec:          f,
		App:           nil,
		Provider:      &fakeProvider{},
		Fixtures:      fixtures.NewRegistryWithOrganization(uuid.New()),
		PollInterval:  time.Hour,
		LeaseDuration: time.Minute,
		Wakeup:        make(chan uuid.UUID, 1),
		Notifier:      &fakeNotifier{},
	}
	rt.ensureWorkContext()

	rt.Sweep(context.Background())
	rt.Wait()

	if !claimCalled {
		t.Fatal("ClaimDueIntents was never called — a failing ReclaimExpiredLeases must not short-circuit the rest of Sweep")
	}
}

// TestDispatchBounded_PanicRecovered_OtherDispatchesStillComplete is
// gap-runtime-resilience.md's "before" regression baseline turned "after"
// assertion: a panic inside one dispatched fn must not crash the test
// process (proving Runtime survives it, unlike before this fix) and must
// not prevent concurrently-running, non-panicking dispatches from
// completing normally. Exercises dispatchBounded directly rather than
// through Sweep/execute() — execute()'s first line calls r.App, a concrete
// *application.Application with no fake-able interface seam, so the
// dispatch-goroutine mechanics (panic recovery, concurrency bound) are unit
// tested at the level they actually live: dispatchBounded, independent of
// what fn happens to be. The DB-backed TestIntegration_Runtime_* tests in
// runtime_test.go cover execute()'s own real behavior end to end.
func TestDispatchBounded_PanicRecovered_OtherDispatchesStillComplete(t *testing.T) {
	rt := &Runtime{Fixtures: fixtures.NewRegistryWithOrganization(uuid.New())}
	rt.ensureWorkContext()

	var completed int32
	const others = 5

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic escaped dispatchBounded and crashed the caller: %v", r)
		}
	}()

	rt.dispatchBounded(func() { panic("simulated: malformed provider response") })
	for i := 0; i < others; i++ {
		rt.dispatchBounded(func() { atomic.AddInt32(&completed, 1) })
	}
	rt.Wait()

	if got := atomic.LoadInt32(&completed); got != others {
		t.Fatalf("completed = %d, want %d — a sibling panic must not prevent other dispatches from finishing", got, others)
	}
}

// TestDispatchBounded_ConcurrencyNeverExceedsBound is
// gap-runtime-resilience.md's load-test requirement: with many dispatches
// due at once, the number running fn concurrently must never exceed
// MaxConcurrentDispatch, regardless of how many were queued at once (the
// overlapping-sweeps scenario the gap doc names).
func TestDispatchBounded_ConcurrencyNeverExceedsBound(t *testing.T) {
	const bound = 3
	const total = 20

	rt := &Runtime{
		Fixtures:              fixtures.NewRegistryWithOrganization(uuid.New()),
		MaxConcurrentDispatch: bound,
	}
	rt.ensureWorkContext()

	var current, maxSeen int32
	started := make(chan struct{}, total)
	release := make(chan struct{})

	for i := 0; i < total; i++ {
		rt.dispatchBounded(func() {
			n := atomic.AddInt32(&current, 1)
			for {
				old := atomic.LoadInt32(&maxSeen)
				if n <= old {
					break
				}
				if atomic.CompareAndSwapInt32(&maxSeen, old, n) {
					break
				}
			}
			started <- struct{}{}
			<-release
			atomic.AddInt32(&current, -1)
		})
	}

	// Deliberately no time.Sleep: at most `bound` goroutines can ever be
	// past the semaphore acquire before release is closed, since none of
	// them frees its slot until <-release unblocks — so draining exactly
	// `bound` signals off started is itself the proof concurrency reached
	// the bound, without racing against a fixed delay. If dispatchBounded
	// ever under-bounds (fewer concurrent slots than configured), this
	// blocks here and the test fails on Go's default timeout rather than
	// asserting a wrong number, which is an acceptable failure mode for a
	// bug that severe.
	for i := 0; i < bound; i++ {
		<-started
	}
	close(release)
	rt.Wait()

	if got := atomic.LoadInt32(&maxSeen); got > bound {
		t.Fatalf("observed %d concurrent dispatches, want at most MaxConcurrentDispatch=%d", got, bound)
	}
}

// TestDispatchBounded_DefaultBoundAppliesWhenUnset proves
// defaultMaxConcurrentDispatch, not an unbounded semaphore, is what a
// zero-value Runtime gets — MaxConcurrentDispatch is easy to forget to set
// (as cmd/companyd/main.go itself would have, before this fix), and an
// unset bound must fail safe (bounded), not fail open (unbounded).
func TestDispatchBounded_DefaultBoundAppliesWhenUnset(t *testing.T) {
	rt := &Runtime{Fixtures: fixtures.NewRegistryWithOrganization(uuid.New())}
	rt.ensureWorkContext()

	if cap(rt.sem) != defaultMaxConcurrentDispatch {
		t.Fatalf("semaphore capacity = %d, want defaultMaxConcurrentDispatch = %d", cap(rt.sem), defaultMaxConcurrentDispatch)
	}
}
