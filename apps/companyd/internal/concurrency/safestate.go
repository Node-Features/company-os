// Package concurrency holds worked illustrations of Go concurrency
// patterns relevant to CompanyOS's architecture. Nothing in this package is
// wired into companyd — each type exists to demonstrate a pattern
// concretely, with a compiled, tested proof, where the real architecture
// doc only states the requirement in prose. It is deliberately not under
// internal/kernel: kernel.md requires Kernel to hold no state between calls
// at all, which every type here necessarily does.
//
// SafeState (safestate.go) illustrates the optimistic-concurrency pattern
// docs/architecture/persistence.md requires for real authoritative state —
// the same pattern ports.AuthoritativeStateRepository.CommitTransition
// already implements for real, against Postgres, for the Workflow
// aggregate (verified against internal/adapters/persistence/supabase/
// workflow_repo.go's "UPDATE workflows SET ... WHERE ... version=$N" for
// ADR-0007). It exists because ADR-0007 needed a concrete, in-process,
// goroutine-level demonstration, and a database round-trip isn't a
// self-contained one.
//
// Bus (eventbus.go) illustrates a correctly-synchronized, topic-based,
// non-blocking in-memory publish/subscribe primitive. It is explicitly not
// CompanyOS's event mechanism — see eventbus.go's doc comment for why, and
// internal/ports.Publisher for the real, outbox-shaped port events.md
// requires instead.
package concurrency

import (
	"errors"
	"sync"
)

// ErrVersionConflict is returned when Adjust's expectedVersion does not
// match the balance's current version. It mirrors ports.ErrConflict
// exactly, for the same reason: a caller must reload current state and
// retry with a fresh expected version, never blindly overwrite.
var ErrVersionConflict = errors.New("concurrency: version conflict")

// SafeState is a versioned, mutex-guarded balance — an in-process stand-in
// for what a real financial-state aggregate's persistence adapter would do
// with a database row and a "WHERE version = $expected" clause.
//
// Why a mutex, not a channel: the operation being protected is a single,
// fast, synchronous compare-then-write with no I/O and no blocking inside
// the critical section — exactly sync.Mutex's designed case. A channel
// earns its cost when one goroutine owns the state and must interleave
// several distinct kinds of work through a select loop (internal/daemon
// does exactly that for supervision); there is only one operation to
// serialize here, so a channel would add indirection without adding
// safety.
//
// Why version-compare-and-write instead of a bare "lock, add, unlock": a
// bare Adjust(delta) would be memory-safe — the mutex alone prevents a data
// race — but it would silently let two callers each apply a delta computed
// from stale information, which is exactly what
// docs/architecture/persistence.md forbids: "conflicting writes fail rather
// than merge implicitly." Requiring the caller to state which version it
// read makes a stale write fail loudly (ErrVersionConflict) instead of
// silently succeeding on outdated assumptions — the same contract
// CommitTransition's expectedVersion parameter enforces. This is the part
// of the pattern that would still be correct if SafeState were ever backed
// by a real database instead of an in-process mutex: the mutex is this
// illustration's implementation detail, not the guarantee itself.
type SafeState struct {
	mu      sync.Mutex
	balance int64
	version int64
}

// NewSafeState returns a SafeState at version 1 — the same starting version
// ports.AuthoritativeStateRepository.CreateWorkflow uses for a new Workflow.
func NewSafeState(initialBalance int64) *SafeState {
	return &SafeState{balance: initialBalance, version: 1}
}

// Read returns the current balance and version as of the call. Like any
// read against real Persistence, it is a snapshot and may already be stale
// by the time the caller acts on it — Read never blocks past another
// in-flight Read; it only contends with a concurrent Adjust.
func (s *SafeState) Read() (balance, version int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.balance, s.version
}

// Adjust applies delta to the balance only if expectedVersion still matches
// the current version, then advances the version by exactly one — the same
// compare-and-write CommitTransition performs against Postgres. It returns
// ErrVersionConflict, unchanged, if another Adjust committed first; the
// caller must Read again and retry with the fresh version, exactly as a
// real caller would after ports.ErrConflict.
//
// Guarantee: under any number of concurrent goroutines, at most one Adjust
// call succeeds per version — no lost update is possible (proved by
// TestSafeState_ConcurrentAdjust_NoLostUpdates under -race).
//
// Non-guarantee: Adjust does not retry internally, and it does not
// guarantee fairness or ordering among callers that lose the race — any of
// the concurrent callers may be the one that wins a given version, in any
// order. A caller that receives ErrVersionConflict and does not retry will
// simply not have its delta applied; silently dropping a write is the
// caller's bug, not a property this type hides.
func (s *SafeState) Adjust(expectedVersion int64, delta int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if expectedVersion != s.version {
		return ErrVersionConflict
	}
	s.balance += delta
	s.version++
	return nil
}
