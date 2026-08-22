package concurrency

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// TestSafeState_ConcurrentAdjust_NoLostUpdates launches many goroutines that
// each read-then-adjust concurrently, retrying on ErrVersionConflict like a
// real caller would, and proves no update is silently lost. Run with -race
// to also prove no data race exists on the underlying fields.
func TestSafeState_ConcurrentAdjust_NoLostUpdates(t *testing.T) {
	const goroutines = 100
	const delta = int64(1)

	s := NewSafeState(0)
	var wg sync.WaitGroup
	var conflicts int64

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				_, version := s.Read()
				runtime.Gosched() // widen the race window: without this, the
				// scheduler can complete each goroutine's Read+Adjust pair
				// back-to-back and never actually interleave two of them,
				// which would let this test pass without ever exercising
				// the conflict-and-retry path it exists to prove.
				err := s.Adjust(version, delta)
				if err == nil {
					return
				}
				if err != ErrVersionConflict {
					t.Errorf("unexpected error: %v", err)
					return
				}
				atomic.AddInt64(&conflicts, 1)
				// A real caller reloads and retries here, exactly like a
				// ports.ErrConflict caller would.
			}
		}()
	}
	wg.Wait()

	balance, version := s.Read()
	if balance != goroutines*delta {
		t.Fatalf("lost update: balance = %d, want %d", balance, goroutines*delta)
	}
	if version != int64(goroutines)+1 {
		t.Fatalf("version = %d, want %d", version, goroutines+1)
	}
	t.Logf("%d goroutines, %d version conflicts observed and retried, zero lost updates", goroutines, conflicts)
}

// TestSafeState_Adjust_RejectsStaleVersion proves a stale expectedVersion is
// rejected rather than silently applied — the guarantee that distinguishes
// this from a bare mutex-protected Adjust(delta).
func TestSafeState_Adjust_RejectsStaleVersion(t *testing.T) {
	s := NewSafeState(100)

	if err := s.Adjust(1, 50); err != nil {
		t.Fatalf("first Adjust: unexpected error: %v", err)
	}
	// version is now 2; retrying with the stale version 1 must fail.
	if err := s.Adjust(1, 50); err != ErrVersionConflict {
		t.Fatalf("stale Adjust: got %v, want ErrVersionConflict", err)
	}

	balance, version := s.Read()
	if balance != 150 {
		t.Fatalf("balance = %d, want 150 (stale write must not apply)", balance)
	}
	if version != 2 {
		t.Fatalf("version = %d, want 2", version)
	}
}
