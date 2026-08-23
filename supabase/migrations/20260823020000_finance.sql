-- Phase 4 Slice 3: Finance Budget -> CostConstraint -> PriceProfile ->
-- ResourceUsage -> ResourceEvaluation, one narrow vertical slice against
-- the one existing CapabilityDefinition and the three real ADR-0004
-- fallback providers (gemini/openai/anthropic).
-- See docs/departments/finance.md, docs/domain/resource.md.
--
-- No FK to principals (same reasoning as prior migrations); no
-- ResourceReservation table (out of scope this slice — no pre-execution
-- reservation/estimate path).
--
-- Every organization-scoped table has RLS enabled and zero policies,
-- matching the existing schema convention. price_profiles is a global
-- catalog, not organization-scoped, seeded below.

CREATE TABLE budgets (
  budget_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  subject_id uuid NOT NULL,
  limit_amount double precision NOT NULL,
  currency text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE budgets ENABLE ROW LEVEL SECURITY;

CREATE TABLE cost_constraints (
  constraint_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  budget_id uuid NOT NULL REFERENCES budgets (budget_id),
  subject_id uuid NOT NULL,
  max_cost double precision NOT NULL,
  currency text NOT NULL,
  enforcement_mode text NOT NULL CHECK (enforcement_mode IN ('ADVISORY')),
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE cost_constraints ENABLE ROW LEVEL SECURITY;

-- Global catalog: one seeded row per ADR-0004 fallback provider, keyed by
-- provider_adapter only (not model_id — see finance.PriceProfile's doc
-- comment for why). Illustrative placeholder $/1K-token rates, not
-- contractually sourced pricing — finance.md's own open question on
-- canonical pricing sourcing stays unresolved, not answered here.
CREATE TABLE price_profiles (
  price_profile_id uuid PRIMARY KEY,
  provider_adapter text NOT NULL UNIQUE,
  model_id text NOT NULL,
  currency text NOT NULL,
  input_price_per_k_tokens double precision NOT NULL,
  output_price_per_k_tokens double precision NOT NULL,
  effective_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE price_profiles ENABLE ROW LEVEL SECURITY;

INSERT INTO price_profiles (price_profile_id, provider_adapter, model_id, currency, input_price_per_k_tokens, output_price_per_k_tokens) VALUES
  ('00000000-0000-4000-a000-000000000001', 'gemini', 'gemini-2.5-flash', 'USD', 0.0001, 0.0004),
  ('00000000-0000-4000-a000-000000000002', 'openai', 'gpt-5-nano', 'USD', 0.00015, 0.0006),
  ('00000000-0000-4000-a000-000000000003', 'anthropic', 'claude-haiku-4-5', 'USD', 0.0002, 0.0010);

CREATE TABLE resource_usage (
  usage_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  result_id uuid NOT NULL,
  subject_id uuid NOT NULL,
  provider_adapter text NOT NULL,
  model_id text NOT NULL,
  input_tokens int NOT NULL,
  output_tokens int NOT NULL,
  cost double precision NOT NULL,
  currency text NOT NULL,
  succeeded boolean NOT NULL,
  measurement_method text NOT NULL CHECK (measurement_method IN ('ACTUAL')),
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE resource_usage ENABLE ROW LEVEL SECURITY;

CREATE TABLE resource_evaluations (
  evaluation_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  subject_id uuid NOT NULL,
  total_cost double precision NOT NULL,
  currency text NOT NULL,
  total_count int NOT NULL,
  successful_count int NOT NULL,
  effective_cost_per_successful_result double precision,
  budget_limit_amount double precision,
  budget_variance double precision,
  -- informational M&E cross-department evidence (finance.md's "Boundary
  -- with M&E") -- nil when M&E hasn't run an Evaluation for the subject
  -- yet, never required for this evaluation to complete.
  me_performance_outcome text,
  me_success_rate double precision,
  status text NOT NULL CHECK (status IN ('COMPUTED', 'INCONCLUSIVE')),
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE resource_evaluations ENABLE ROW LEVEL SECURITY;
