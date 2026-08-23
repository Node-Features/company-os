-- Phase 4 Slice 2: M&E Result -> Metric -> Evaluation -> PerformanceProfile,
-- one narrow vertical slice against the one existing CapabilityDefinition.
-- See docs/departments/monitoring-evaluation.md, docs/domain/metric.md,
-- docs/domain/evaluation.md.
--
-- No metric_definitions table: this slice's one MetricDefinition
-- ("capability attempt succeeded") is hardcoded in
-- internal/domain/monitoringevaluation, the same way fixtures.Registry
-- hardcodes the one CapabilityDefinition/WorkflowDefinition.
--
-- No FK to principals (same reasoning as prior migrations); metric_ids/
-- evidence-equivalent references stay loose uuids.
--
-- Every table is organization_id-scoped with RLS enabled and zero
-- policies, matching the existing schema convention.

CREATE TABLE metrics (
  metric_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  subject_id uuid NOT NULL,
  result_id uuid NOT NULL,
  value double precision NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE metrics ENABLE ROW LEVEL SECURITY;

CREATE TABLE evaluations (
  evaluation_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  subject_id uuid NOT NULL,
  -- JSON array of Metric UUID strings, not a native uuid[] column — same
  -- reasoning as findings.evidence_ids in the research_loop migration:
  -- this slice never needs a SQL-side array operation on it.
  metric_ids jsonb NOT NULL,
  outcome text NOT NULL CHECK (outcome IN ('PASS', 'FAIL', 'INCONCLUSIVE')),
  success_rate double precision NOT NULL,
  sample_size int NOT NULL,
  independence text NOT NULL CHECK (independence IN ('SELF_REPORTED')),
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE evaluations ENABLE ROW LEVEL SECURITY;

-- Latest-Evaluation-only projection per subject (upserted, not
-- history-preserving) — narrower than evaluation.md's full versioned
-- PerformanceProfile.
CREATE TABLE performance_profiles (
  subject_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  latest_evaluation_id uuid NOT NULL REFERENCES evaluations (evaluation_id),
  outcome text NOT NULL CHECK (outcome IN ('PASS', 'FAIL', 'INCONCLUSIVE')),
  success_rate double precision NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE performance_profiles ENABLE ROW LEVEL SECURITY;
