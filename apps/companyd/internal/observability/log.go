package observability

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
)

var defaultLogger atomic.Pointer[slog.Logger]

// Init configures the process-wide structured logger — called once from
// cmd/companyd/main.go's boot sequence (ADR-0006 boot stage 3). JSON
// output: this is a service log read by tooling, not a human terminal.
// Safe to call more than once (e.g. from a test); the most recent call
// wins. Before Init is called, Logger falls back to slog.Default() rather
// than panicking or discarding output.
func Init(w io.Writer, level slog.Level) {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	defaultLogger.Store(slog.New(handler))
}

func base() *slog.Logger {
	if l := defaultLogger.Load(); l != nil {
		return l
	}
	return slog.Default()
}

// Logger returns the process logger enriched with whatever
// ExecutionContext correlation identity is attached to ctx — every record
// emitted through it carries structured attributes for the fields known
// at this point in the lifecycle, never a string-interpolated ID.
func Logger(ctx context.Context) *slog.Logger {
	attrs := FromContext(ctx).Attrs()
	if len(attrs) == 0 {
		return base()
	}
	args := make([]any, len(attrs))
	for i, a := range attrs {
		args[i] = a
	}
	return base().With(args...)
}
