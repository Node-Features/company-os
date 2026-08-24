// MetricsRecorder port for operational metrics over the execution
// lifecycle. See docs/architecture/observability.md.
package ports

// MetricsRecorder records operational metrics against a fixed,
// pre-declared name/label vocabulary (docs/architecture/observability.md's
// metric catalog) — never a dynamically-constructed metric name, which is
// what keeps label cardinality bounded. Implemented by
// internal/adapters/observability/prometheus.Recorder; Application,
// Runtime, and the outbox Sweeper depend only on this interface, never on
// a concrete metrics vendor. A nil MetricsRecorder is never required —
// every call site treats it as optional, the same nil-safe pattern
// ports.ChangeNotifier already establishes for Runtime's Notifier.
type MetricsRecorder interface {
	IncrCounter(name string, labels map[string]string)
	ObserveHistogram(name string, seconds float64, labels map[string]string)
	SetGauge(name string, value float64, labels map[string]string)
}
