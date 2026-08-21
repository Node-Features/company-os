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
// bounded in-flight operations ... before closing dependencies").
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
		return ctx.Err()
	}
}
