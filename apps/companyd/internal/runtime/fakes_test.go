package runtime

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/execution"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

var errNotifyFailed = errors.New("fakeNotifier: simulated notification failure")

// fakeProvider is a test double for ports.ProviderAdapter — the one
// genuinely external dependency Runtime has (an LLM provider over the
// network). Real-database tests in this package fake this and only this,
// keeping everything else (Exec, App/Repo) backed by the real Postgres
// schema — the same "fake only the true external boundary" discipline
// internal/application's own integration tests already use for provider
// calls (see submitFakeResult's doc comment there).
type fakeProvider struct {
	mu    sync.Mutex
	calls int

	// result/err: if err is non-nil, Generate returns it (classified as
	// retryable via fakeClassifiedErr below when retryable is true).
	// Otherwise Generate returns result.
	result    ports.IntelligenceResult
	err       error
	retryable bool

	// block, if non-nil, is closed by the test to release a Generate call
	// that's deliberately hanging (waiting on ctx.Done() or block) — used
	// to test cancellation: Generate blocks until either the caller's ctx
	// is cancelled (returns ctx.Err()) or the test closes block (returns
	// the configured result/err), whichever happens first.
	block chan struct{}
}

func (f *fakeProvider) Generate(ctx context.Context, _ ports.IntelligenceRequest) (ports.IntelligenceResult, error) {
	f.mu.Lock()
	f.calls++
	block := f.block
	res, err, retryable := f.result, f.err, f.retryable
	f.mu.Unlock()

	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return ports.IntelligenceResult{}, ctx.Err()
		}
	}
	if err != nil {
		if retryable {
			return ports.IntelligenceResult{}, fakeClassifiedErr{err: err, retryable: true}
		}
		return ports.IntelligenceResult{}, fakeClassifiedErr{err: err, retryable: false}
	}
	return res, nil
}

func (f *fakeProvider) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// fakeClassifiedErr satisfies ports.IsRetryable's duck-typed Retryable()
// bool contract, mirroring each real provider adapter's own unexported
// classifiedError shape.
type fakeClassifiedErr struct {
	err       error
	retryable bool
}

func (e fakeClassifiedErr) Error() string   { return e.err.Error() }
func (e fakeClassifiedErr) Retryable() bool { return e.retryable }

// fakeNotifier is a test double for ports.ChangeNotifier — a best-effort
// hint by contract (see Runtime.notifyChanged's doc comment), so a test
// configuring it to always fail is exactly how "runtime notification
// failure must not block dispatch" gets proven.
type fakeNotifier struct {
	mu      sync.Mutex
	calls   int
	failAll bool
}

func (f *fakeNotifier) NotifyWorkflowChanged(context.Context, uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.failAll {
		return errNotifyFailed
	}
	return nil
}

func (f *fakeNotifier) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// fakeFailingExec is a ports.ExecutionRepository whose claim/reclaim
// methods always fail — used only by TestSweep_DatabaseFailure_DoesNotPanic
// (runtime_unit_test.go) to prove Runtime degrades gracefully when its
// database becomes unreachable, without needing a real database to go down
// on command (which a real-Postgres integration test can't do safely or
// deterministically). Every method Runtime would only reach after a
// successful claim panics if called — this fake exists to prove those
// paths are never reached when the claim itself fails, not to be a
// general-purpose ExecutionRepository double.
type fakeFailingExec struct{ err error }

func (f fakeFailingExec) ClaimDueIntents(context.Context, uuid.UUID, int, time.Duration, string) ([]execution.ClaimedExecution, error) {
	return nil, f.err
}
func (f fakeFailingExec) ReclaimExpiredLeases(context.Context, uuid.UUID, int) ([]execution.ClaimedExecution, error) {
	return nil, f.err
}
func (fakeFailingExec) RecordDispatched(context.Context, uuid.UUID, int64, string) error {
	panic("fakeFailingExec: RecordDispatched must not be reached when Claim/Reclaim already failed")
}
func (fakeFailingExec) RecordTerminal(context.Context, uuid.UUID, int64, execution.AttemptStatus, *uuid.UUID) error {
	panic("fakeFailingExec: RecordTerminal must not be reached when Claim/Reclaim already failed")
}
func (fakeFailingExec) ScheduleRetry(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	panic("fakeFailingExec: ScheduleRetry must not be reached when Claim/Reclaim already failed")
}
func (fakeFailingExec) SaveResult(context.Context, *result.Result) error {
	panic("fakeFailingExec: SaveResult must not be reached when Claim/Reclaim already failed")
}
func (fakeFailingExec) GetResult(context.Context, uuid.UUID, uuid.UUID) (*result.Result, error) {
	panic("fakeFailingExec: GetResult must not be reached when Claim/Reclaim already failed")
}
func (fakeFailingExec) GetLatestResult(context.Context, uuid.UUID, uuid.UUID) (*result.Result, error) {
	panic("fakeFailingExec: GetLatestResult must not be reached when Claim/Reclaim already failed")
}
func (fakeFailingExec) ListExecutionUnits(context.Context, uuid.UUID, uuid.UUID) ([]execution.ExecutionUnit, error) {
	panic("fakeFailingExec: ListExecutionUnits must not be reached when Claim/Reclaim already failed")
}

// partialFailingExec fails ReclaimExpiredLeases only, succeeding (with an
// empty result) on ClaimDueIntents — used to prove Sweep's two halves are
// independently resilient (see TestReclaimAbandoned_DatabaseFailure_DoesNotBlockClaiming).
type partialFailingExec struct {
	fakeFailingExec
	reclaimErr error
	onClaim    func()
}

func (f *partialFailingExec) ReclaimExpiredLeases(context.Context, uuid.UUID, int) ([]execution.ClaimedExecution, error) {
	return nil, f.reclaimErr
}
func (f *partialFailingExec) ClaimDueIntents(context.Context, uuid.UUID, int, time.Duration, string) ([]execution.ClaimedExecution, error) {
	if f.onClaim != nil {
		f.onClaim()
	}
	return nil, nil
}
