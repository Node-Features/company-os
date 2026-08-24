package daemon

import (
	"context"

	"github.com/Node-Features/company-os/apps/companyd/internal/runtime"
)

// Daemon is the minimal first-slice process-supervision component per
// docs/architecture/daemon.md: it starts Runtime's dispatch loop and
// exposes health via the existing /health handler — not full multi-worker
// coordination, which this single-process topology (ADR-0004) doesn't
// need yet.
type Daemon struct {
	rt *runtime.Runtime
}

func New(rt *runtime.Runtime) *Daemon { return &Daemon{rt: rt} }

// Start spawns Runtime's poll-and-wake loop; it returns immediately, the
// loop runs until ctx is cancelled.
func (d *Daemon) Start(ctx context.Context) error {
	go d.rt.Start(ctx)
	return nil
}

// Shutdown waits, bounded by ctx, for in-flight execute() calls to drain
// (docs/architecture/daemon.md#lifecycle: "stop accepting work, drain
// bounded in-flight operations ... before closing dependencies"). If ctx
// expires first, it calls Runtime.StopWork so anything still in flight past
// the deadline at least observes cancellation going forward — it cannot be
// force-killed, only asked to stop, so this narrows the abandonment window
// (bounded by ctx's own deadline) rather than eliminating it; the lease on
// whatever it was doing will still expire normally and ReclaimExpiredLeases
// recovers it on a later Sweep the same as any other lost worker.
func (d *Daemon) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		d.rt.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		d.rt.StopWork()
		return ctx.Err()
	}
}
