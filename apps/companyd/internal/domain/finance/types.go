// Package finance holds Finance's domain types (docs/departments/finance.md,
// docs/domain/resource.md). Minimal field sets only — narrower than those
// docs' full aspirational contracts, same convention every prior Phase 4
// slice used.
package finance

import (
	"time"

	"github.com/google/uuid"
)

// Currency is this slice's currency vocabulary. finance.md's own open
// question on canonical currency/exchange-rate source stays unresolved —
// USD is a placeholder default, not a resolved answer.
type Currency string

const CurrencyUSD Currency = "USD"

// Budget is a versioned resource envelope for one organization+subject
// scope (fixtures.CapabilityID this slice) — narrower than finance.md's
// full envelope (owner, period, reservation policy, contingency, rollover,
// alert thresholds are all out of scope this slice).
type Budget struct {
	BudgetID       uuid.UUID
	OrganizationID uuid.UUID
	SubjectID      uuid.UUID
	LimitAmount    float64
	Currency       Currency
	CreatedAt      time.Time
}

// EnforcementMode is resource.md's hard/advisory distinction. This slice
// only ever creates ADVISORY constraints — HARD enforcement would mean
// gating Runtime dispatch on a Finance check, which is out of scope this
// slice (no code path consults CostConstraint before or during real
// execution).
type EnforcementMode string

const EnforcementAdvisory EnforcementMode = "ADVISORY"

// CostConstraint is resource.md's CostConstraint specialization, scoped to
// one Budget/subject. No versioning/supersession-link machinery this
// slice — calling CreateCostConstraint again simply creates a new row, and
// repository reads return the most recent by CreatedAt.
type CostConstraint struct {
	ConstraintID    uuid.UUID
	OrganizationID  uuid.UUID
	BudgetID        uuid.UUID
	SubjectID       uuid.UUID
	MaxCost         float64
	Currency        Currency
	EnforcementMode EnforcementMode
	CreatedAt       time.Time
}

// ConstraintOutcome mirrors resource.md's "Enforcement outcomes" section
// vocabulary exactly.
type ConstraintOutcome string

const (
	WithinLimit            ConstraintOutcome = "WITHIN_LIMIT"
	ApproachingLimit       ConstraintOutcome = "APPROACHING_LIMIT"
	LimitExceeded          ConstraintOutcome = "LIMIT_EXCEEDED"
	MeasurementUnavailable ConstraintOutcome = "MEASUREMENT_UNAVAILABLE"
	NotApplicable          ConstraintOutcome = "NOT_APPLICABLE"
)

// PriceProfile is migration-seeded (one row per ADR-0004 fallback
// provider: gemini/openai/anthropic), not created through a governed use
// case this slice — same precedent as Phase 3 Slice 6's one-time
// Organization seed. Keyed by ProviderAdapter only, not ModelID: Gemini's
// ModelID is a fixed compile-time constant, but OpenAI's and Anthropic's
// ModelID is echoed back from the live API response and isn't a stable
// string to hardcode a lookup key against. ModelID is still recorded here
// for provenance/display.
type PriceProfile struct {
	PriceProfileID        uuid.UUID
	ProviderAdapter       string
	ModelID               string
	Currency              Currency
	InputPricePerKTokens  float64
	OutputPricePerKTokens float64
	EffectiveAt           time.Time
}

// MeasurementMethod distinguishes resource.md's estimate-vs-actual usage
// records. This slice only ever records ACTUAL, computed post-hoc from a
// real completed Result — no pre-execution reservation/estimate path
// (resource.md's ResourceReservation is out of scope this slice).
type MeasurementMethod string

const MeasurementActual MeasurementMethod = "ACTUAL"

// ResourceUsage is an attributable observation of consumed resources for
// one Result. Succeeded is carried from the source Result's outcome so
// RunResourceEvaluation can compute effective-cost-per-successful-result
// without re-touching the Execution repository — finance.md's invariant
// that failed/retried/rejected work is included in effective-cost
// analysis, not just successful calls.
type ResourceUsage struct {
	UsageID           uuid.UUID
	OrganizationID    uuid.UUID
	ResultID          uuid.UUID
	SubjectID         uuid.UUID
	ProviderAdapter   string
	ModelID           string
	InputTokens       int
	OutputTokens      int
	Cost              float64
	Currency          Currency
	Succeeded         bool
	MeasurementMethod MeasurementMethod
	CreatedAt         time.Time
}

// EvaluationStatus is this slice's ResourceEvaluation outcome vocabulary —
// narrower than resource.md's full lifecycle, matching what
// RunResourceEvaluation can actually determine.
type EvaluationStatus string

const (
	EvaluationComputed     EvaluationStatus = "COMPUTED"
	EvaluationInconclusive EvaluationStatus = "INCONCLUSIVE"
)

// ResourceEvaluation combines Finance-owned cost/usage evidence with
// M&E-owned outcome evidence (finance.md's "Boundary with Research and
// M&E") for one subject. EffectiveCostPerSuccessfulResult is nil when
// SuccessfulCount is zero (never divide by zero, never assume);
// BudgetLimitAmount/BudgetVariance are nil when no Budget exists yet for
// the subject. MEPerformanceOutcome/MESuccessRate are nil when M&E hasn't
// run an Evaluation for the subject yet — Finance's own cost evidence
// still stands independently either way (finance.md: "Finance consumes
// M&E outcome evidence without rewriting it," never gated on it existing).
type ResourceEvaluation struct {
	EvaluationID                     uuid.UUID
	OrganizationID                   uuid.UUID
	SubjectID                        uuid.UUID
	TotalCost                        float64
	Currency                         Currency
	TotalCount                       int
	SuccessfulCount                  int
	EffectiveCostPerSuccessfulResult *float64
	BudgetLimitAmount                *float64
	BudgetVariance                   *float64
	MEPerformanceOutcome             *string
	MESuccessRate                    *float64
	Status                           EvaluationStatus
	CreatedAt                        time.Time
}
