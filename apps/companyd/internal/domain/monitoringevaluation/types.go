// Package monitoringevaluation holds M&E's domain types
// (docs/departments/monitoring-evaluation.md, docs/domain/metric.md,
// docs/domain/evaluation.md). Minimal field sets only — narrower than
// those docs' full aspirational contracts, same convention every prior
// slice used.
package monitoringevaluation

import (
	"time"

	"github.com/google/uuid"
)

// MetricDefinitionID identifies this slice's one hardcoded MetricDefinition
// — "capability attempt succeeded" for the one existing CapabilityDefinition
// (fixtures.CapabilityID). metric.md's full lifecycle (DRAFT/ACTIVE/RETIRED,
// review intervals, segmentation) is out of scope; one ACTIVE definition
// lives in code, the same way fixtures.Registry hardcodes the one
// CapabilityDefinition/WorkflowDefinition.
var MetricDefinitionID = uuid.MustParse("00000000-0000-4000-9000-000000000001")

// Metric is one immutable measurement under MetricDefinitionID.
type Metric struct {
	MetricID       uuid.UUID
	OrganizationID uuid.UUID
	SubjectID      uuid.UUID // fixtures.CapabilityID this slice
	ResultID       uuid.UUID // source Result this Metric was computed from
	Value          float64   // 1.0 (succeeded) or 0.0 (did not)
	CreatedAt      time.Time
}

// Independence is evaluation.md's independence classification. This slice
// only ever produces SelfReported — the same system computing the Metric
// also runs the Evaluation; no INTERNAL_INDEPENDENT/DUAL_REVIEWED claim
// would be true yet (monitoring-evaluation.md's own open question on
// evaluator independence is unresolved, not answered here).
type Independence string

const (
	IndependenceSelfReported Independence = "SELF_REPORTED"
)

// EvaluationOutcome is this slice's outcome vocabulary — narrower than
// evaluation.md's full lifecycle, matching what ClassifyEvaluation can
// actually determine from a success-rate metric.
type EvaluationOutcome string

const (
	EvaluationPass         EvaluationOutcome = "PASS"
	EvaluationFail         EvaluationOutcome = "FAIL"
	EvaluationInconclusive EvaluationOutcome = "INCONCLUSIVE"
)

// Evaluation is an attributable assessment of a set of Metrics for one
// subject.
type Evaluation struct {
	EvaluationID   uuid.UUID
	OrganizationID uuid.UUID
	SubjectID      uuid.UUID
	MetricIDs      []uuid.UUID
	Outcome        EvaluationOutcome
	SuccessRate    float64
	SampleSize     int
	Independence   Independence
	CreatedAt      time.Time
}

// PerformanceProfile is the latest-Evaluation-only projection for one
// subject this slice keeps — not the full versioned, history-aware
// projection evaluation.md describes.
type PerformanceProfile struct {
	SubjectID          uuid.UUID
	OrganizationID     uuid.UUID
	LatestEvaluationID uuid.UUID
	Outcome            EvaluationOutcome
	SuccessRate        float64
	UpdatedAt          time.Time
}
