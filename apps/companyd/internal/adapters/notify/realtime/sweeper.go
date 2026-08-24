package realtime

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/observability"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// Sweeper is the read side of event_outbox: it discovers not-yet-published
// rows and drives them through Publisher, marking each published (or
// recording a failure for retry next Sweep) — the "outbox-equivalent
// mechanism" docs/architecture/events.md's persistence step 3 requires,
// unimplemented until Phase 1 Slice 8. Shaped after
// internal/runtime.Runtime's own Start/Sweep poll loop for the same reason
// that one exists: cheap, always-correct-eventually, no external scheduler.
type Sweeper struct {
	Outbox       ports.OutboxRepository
	Publisher    ports.Publisher
	OrgID        uuid.UUID
	PollInterval time.Duration
	BatchSize    int

	// Metrics is an optional operational-metrics sink
	// (docs/architecture/observability.md). A nil Metrics is never an
	// error — every emission call site checks it first.
	Metrics ports.MetricsRecorder

	// lastBacklog/lastReconcileAtUnixNano back Diagnostics(), read from a
	// different goroutine than Sweep runs on (health.go's HTTP handler),
	// hence atomic rather than plain fields.
	lastBacklog             atomic.Int64
	lastReconcileAtUnixNano atomic.Int64
}

// Start runs the poll loop until ctx is cancelled. There is no wake-up fast
// path (unlike Runtime.Wakeup) — PollInterval alone directly bounds
// publish latency, and is kept short (around 1s) since the query it drives
// is a cheap, indexed read.
func (s *Sweeper) Start(ctx context.Context) {
	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Sweep(ctx)
		}
	}
}

// Diagnostics reports the outbox backlog observed on the most recent Sweep
// and when that Sweep ran — read-only, side-effect-free values for
// health.go, per docs/architecture/daemon.md's "Health probes are
// side-effect free." Never authoritative: a fresh call to Sweep may
// immediately supersede these values.
func (s *Sweeper) Diagnostics() (backlog int, lastReconcileAt time.Time) {
	backlog = int(s.lastBacklog.Load())
	if ns := s.lastReconcileAtUnixNano.Load(); ns != 0 {
		lastReconcileAt = time.Unix(0, ns)
	}
	return backlog, lastReconcileAt
}

func (s *Sweeper) incrCounter(name string, labels map[string]string) {
	if s.Metrics == nil {
		return
	}
	s.Metrics.IncrCounter(name, labels)
}

// Sweep runs exactly one load-publish-mark pass.
func (s *Sweeper) Sweep(ctx context.Context) {
	s.lastReconcileAtUnixNano.Store(time.Now().UnixNano())
	s.incrCounter("reconciliation_runs_total", nil)

	events, err := s.Outbox.LoadUnpublished(ctx, s.OrgID, s.BatchSize)
	if err != nil {
		observability.Logger(ctx).Error("realtime: load unpublished events failed", "error", err.Error())
		return
	}
	s.lastBacklog.Store(int64(len(events)))
	if s.Metrics != nil {
		s.Metrics.SetGauge("outbox_backlog", float64(len(events)), nil)
	}
	if len(events) == 0 {
		return
	}

	ids := make([]uuid.UUID, len(events))
	for i, e := range events {
		ids[i] = e.EventID
	}

	if err := s.Publisher.Publish(ctx, events); err != nil {
		observability.Logger(ctx).Error("realtime: publish events failed", "count", len(events), "error", err.Error())
		if err := s.Outbox.MarkPublishFailed(ctx, ids, err.Error()); err != nil {
			observability.Logger(ctx).Error("realtime: mark publish failed error", "error", err.Error())
		}
		return
	}

	if err := s.Outbox.MarkPublished(ctx, ids); err != nil {
		observability.Logger(ctx).Error("realtime: mark published failed", "error", err.Error())
	}
}
