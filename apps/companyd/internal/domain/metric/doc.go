// Package metric is reserved for the canonical Metric domain contract
// (docs/domain/metric.md, Status: APPROVED) but has no implementation
// yet — no MetricDefinition or Metric type exists here
// (docs/audit/backlog-p2-p4.md's "doc-only stubs" row). The real,
// narrower equivalent is internal/domain/monitoringevaluation.Metric;
// there is no separate MetricDefinition type anywhere — M&E instead
// hardcodes one definition as a package-level var
// (monitoringevaluation.MetricDefinitionID).
package metric
