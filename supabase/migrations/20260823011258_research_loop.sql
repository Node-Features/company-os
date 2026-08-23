-- Phase 4 Slice 1: Research Signal -> ResearchQuestion -> Evidence ->
-- Finding -> Recommendation, one narrow vertical slice. See
-- docs/workflows/research-loop.md.
--
-- No FK to principals (same reasoning as the organizations/principals
-- migration): submitted_by_principal_id is the real, resolved HumanPrincipal
-- this slice introduces (ROADMAP.md Phase 3 Slices 4/6), always present for
-- these new actions, but kept as a loose uuid for consistency with every
-- other principal_id column in this schema.
--
-- No compare-and-swap version columns: this flow is linear, not concurrently
-- mutated the way workflows is.
--
-- Every table is organization_id-scoped with RLS enabled and zero policies,
-- matching the existing schema convention (see the first-slice migration's
-- header comment).

CREATE TABLE signals (
  signal_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  source_type text NOT NULL,
  description text NOT NULL,
  submitted_by_principal_id uuid NOT NULL,
  submitted_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE signals ENABLE ROW LEVEL SECURITY;

CREATE TABLE research_questions (
  question_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  signal_id uuid NOT NULL REFERENCES signals (signal_id),
  text text NOT NULL,
  status text NOT NULL CHECK (status IN ('OPEN', 'CLOSED')),
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE research_questions ENABLE ROW LEVEL SECURITY;

CREATE TABLE evidence (
  evidence_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  question_id uuid NOT NULL REFERENCES research_questions (question_id),
  source text NOT NULL,
  content text NOT NULL,
  retrieved_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE evidence ENABLE ROW LEVEL SECURITY;

CREATE TABLE findings (
  finding_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  question_id uuid NOT NULL REFERENCES research_questions (question_id),
  claim text NOT NULL,
  -- JSON array of Evidence UUID strings, not a native uuid[] column — this
  -- slice never needs a SQL-side array operation (ANY(), containment,
  -- etc.), so jsonb keeps the Go driver code simple, matching this
  -- schema's existing convention of using jsonb for structured fields
  -- (inputs, constraints, output).
  evidence_ids jsonb NOT NULL,
  status text NOT NULL CHECK (status IN ('PUBLISHED', 'SUPERSEDED')),
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE findings ENABLE ROW LEVEL SECURITY;

CREATE TABLE recommendations (
  recommendation_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  finding_id uuid NOT NULL REFERENCES findings (finding_id),
  proposed_action text NOT NULL,
  status text NOT NULL CHECK (status IN ('PROPOSED', 'EXPIRED')),
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE recommendations ENABLE ROW LEVEL SECURITY;
