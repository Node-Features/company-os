// Package prometheus implements ports.MetricsRecorder using
// github.com/prometheus/client_golang — the only package in this codebase
// that imports a metrics vendor library directly, per
// docs/architecture/observability.md. Application, Runtime, and the
// outbox Sweeper never import this package's dependency themselves; they
// depend only on ports.MetricsRecorder.
package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// counterSpecs/histogramSpecs/gaugeSpecs are docs/architecture/observability.md's
// metric catalog, one entry per row. Recorder pre-registers exactly these
// at construction — metrics are never created dynamically from request
// data, which is what keeps Prometheus label cardinality bounded.
var (
	counterSpecs = map[string][]string{
		"governance_decisions_total": {"outcome", "action"},
		"executions_total":           {"state"},
		"retries_total":              {},
		"abandoned_total":            {},
		"lease_expirations_total":    {},
		"reconciliation_runs_total":  {},
	}
	histogramSpecs = map[string][]string{
		"provider_latency_seconds": {"provider", "outcome"},
		"approval_latency_seconds": {},
	}
	gaugeSpecs = map[string][]string{
		"outbox_backlog": {},
	}
)

// Recorder implements ports.MetricsRecorder over a private
// prometheus.Registry (not the global default registry, so multiple
// Recorders — e.g. one per test — never collide on registration).
type Recorder struct {
	registry   *prometheus.Registry
	counters   map[string]*prometheus.CounterVec
	histograms map[string]*prometheus.HistogramVec
	gauges     map[string]*prometheus.GaugeVec
}

// New constructs a Recorder with every metric in
// docs/architecture/observability.md's catalog pre-registered.
func New() *Recorder {
	reg := prometheus.NewRegistry()
	r := &Recorder{
		registry:   reg,
		counters:   make(map[string]*prometheus.CounterVec, len(counterSpecs)),
		histograms: make(map[string]*prometheus.HistogramVec, len(histogramSpecs)),
		gauges:     make(map[string]*prometheus.GaugeVec, len(gaugeSpecs)),
	}
	for name, labels := range counterSpecs {
		c := prometheus.NewCounterVec(prometheus.CounterOpts{Name: name}, labels)
		reg.MustRegister(c)
		r.counters[name] = c
	}
	for name, labels := range histogramSpecs {
		h := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name}, labels)
		reg.MustRegister(h)
		r.histograms[name] = h
	}
	for name, labels := range gaugeSpecs {
		g := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name}, labels)
		reg.MustRegister(g)
		r.gauges[name] = g
	}
	return r
}

// Handler exposes the registered metrics in Prometheus text-exposition
// format for GET /metrics.
func (r *Recorder) Handler() http.Handler {
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{})
}

// IncrCounter is a no-op for an unregistered name rather than a panic or
// error — a typo'd metric name should never be able to take down a
// dispatch path, matching this codebase's "diagnostic output is never a
// dependency" posture (observability.md's non-authority invariant).
func (r *Recorder) IncrCounter(name string, labels map[string]string) {
	c, ok := r.counters[name]
	if !ok {
		return
	}
	c.With(labels).Inc()
}

func (r *Recorder) ObserveHistogram(name string, seconds float64, labels map[string]string) {
	h, ok := r.histograms[name]
	if !ok {
		return
	}
	h.With(labels).Observe(seconds)
}

func (r *Recorder) SetGauge(name string, value float64, labels map[string]string) {
	g, ok := r.gauges[name]
	if !ok {
		return
	}
	g.With(labels).Set(value)
}
