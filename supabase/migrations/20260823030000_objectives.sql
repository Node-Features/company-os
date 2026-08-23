-- Phase 4 Slice 4: Objective-creation gate. A Finding, Recommendation,
-- Evaluation, or ResourceEvaluation may propose an Objective through a
-- distinct, governed Application use case (internal/application's
-- ProposeObjective) — this table is the first real Objective persistence
-- in the codebase; fixtures.ObjectiveID's hardcoded fixture Objective is
-- untouched and unrelated (Workflow's CreateWorkflow keeps validating
-- against it exactly as before).
--
-- The unique constraint on (organization_id, source_type, source_id) is
-- DB-level defense-in-depth for the dedup check
-- internal/kernel/objective.ValidateProposal already performs at the
-- application layer (docs/architecture/departments.md: "duplicate
-- feedback... cannot create another... Objective") — a race between two
-- concurrent proposals for the same source is rejected here with a
-- unique_violation, mapped to ports.ErrConflict by the Supabase adapter.
--
-- No FK to principals (same reasoning as prior migrations). RLS enabled
-- with zero policies, matching the existing schema convention.

CREATE TABLE objectives (
  objective_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  title text NOT NULL,
  intent text NOT NULL,
  status text NOT NULL CHECK (status IN ('PROPOSED', 'APPROVED', 'RETIRED')),
  owner_principal_id uuid NOT NULL,
  source_type text NOT NULL CHECK (source_type IN ('FINDING', 'RECOMMENDATION', 'EVALUATION', 'RESOURCE_EVALUATION')),
  source_id uuid NOT NULL,
  governance_decision_id uuid NOT NULL,
  approval_id uuid,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id, source_type, source_id)
);
ALTER TABLE objectives ENABLE ROW LEVEL SECURITY;
