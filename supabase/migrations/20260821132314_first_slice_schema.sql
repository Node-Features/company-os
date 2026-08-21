-- Phase 1 first vertical slice schema.
-- Workflow is the sole authoritative aggregate (persistence.md); everything
-- else referenced here (Organization, Objective, WorkflowDefinition,
-- CapabilityDefinition, Principal) is an in-memory fixture this slice
-- (internal/fixtures), not a table, per ADR-0004 and this plan's scoping.
--
-- Every table is organization_id-scoped and has RLS enabled with zero
-- policies: companyd's pooled DATABASE_URL role is a trusted backend role
-- and bypasses RLS, so this is a default-deny backstop for the
-- publishable/anon Supabase key `web` uses for reads only, not the
-- authorization mechanism itself (that lives in Application/Kernel).

CREATE SEQUENCE lease_fencing_seq;

-- Authoritative Workflow aggregate. Addressed by (organization_id,
-- workflow_id); version is a compare-and-write guard (persistence.md).
CREATE TABLE workflows (
  organization_id uuid NOT NULL,
  workflow_id uuid NOT NULL,
  version bigint NOT NULL,
  definition_id uuid NOT NULL,
  definition_version int NOT NULL,
  objective_id uuid NOT NULL,
  state text NOT NULL CHECK (state IN ('PLANNED', 'READY', 'COMPLETED', 'FAILED', 'CANCELLED')),
  initiating_principal_id uuid NOT NULL,
  correlation_id uuid NOT NULL,
  causation_id uuid,
  inputs jsonb NOT NULL DEFAULT '{}',
  wait_reason text,
  terminal_reason text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (organization_id, workflow_id)
);
ALTER TABLE workflows ENABLE ROW LEVEL SECURITY;

-- Immutable domain event log (event.md). Committed in the same transaction
-- as the workflows state write (state-plus-outbox per ADR-0004).
CREATE TABLE domain_events (
  event_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  event_type text NOT NULL,
  schema_version int NOT NULL DEFAULT 1,
  subject_type text NOT NULL,
  subject_id uuid NOT NULL,
  subject_version bigint NOT NULL,
  occurred_at timestamptz NOT NULL,
  recorded_at timestamptz NOT NULL DEFAULT now(),
  correlation_id uuid NOT NULL,
  causation_id uuid,
  workflow_id uuid,
  objective_id uuid,
  principal_id uuid,
  payload jsonb NOT NULL
);
ALTER TABLE domain_events ENABLE ROW LEVEL SECURITY;

-- Post-commit publication outbox (persistence.md). This slice's publisher
-- is a no-op stub that marks rows published; Supabase Realtime LISTEN/NOTIFY
-- is deferred fast-follow work once companyd runs as more than one process.
CREATE TABLE event_outbox (
  event_id uuid PRIMARY KEY REFERENCES domain_events (event_id),
  organization_id uuid NOT NULL,
  enqueued_at timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz,
  publish_attempts int NOT NULL DEFAULT 0,
  last_error text
);
CREATE INDEX event_outbox_pending_idx ON event_outbox (enqueued_at) WHERE published_at IS NULL;
ALTER TABLE event_outbox ENABLE ROW LEVEL SECURITY;

-- Persisted request for Runtime to perform one bounded capability step
-- (workflow.md's ExecutionIntent). Written atomically with the causing
-- Workflow transition.
CREATE TABLE execution_intents (
  intent_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  workflow_id uuid NOT NULL,
  workflow_version bigint NOT NULL,
  capability_definition_id uuid NOT NULL,
  capability_definition_version int NOT NULL,
  governance_decision_id uuid NOT NULL,
  idempotency_key text NOT NULL,
  constraints jsonb NOT NULL DEFAULT '{}',
  due_at timestamptz NOT NULL,
  inputs jsonb NOT NULL,
  status text NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'CLAIMED', 'CLOSED')),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (organization_id, idempotency_key)
);
CREATE INDEX execution_intents_due_idx ON execution_intents (due_at) WHERE status = 'PENDING';
ALTER TABLE execution_intents ENABLE ROW LEVEL SECURITY;

-- Runtime execution mechanics (execution.md): attempts, leases, and a
-- checkpoint column folded in since this slice's one capability is a single
-- bounded synchronous call with no partial progress to checkpoint.
CREATE TABLE execution_attempts (
  attempt_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  intent_id uuid NOT NULL REFERENCES execution_intents (intent_id),
  workflow_id uuid NOT NULL,
  workflow_version bigint NOT NULL,
  logical_operation_id text NOT NULL,
  attempt_number int NOT NULL DEFAULT 1,
  capability_request_id uuid NOT NULL,
  status text NOT NULL CHECK (
    status IN (
      'CLAIMED', 'DISPATCHED', 'WAITING', 'SUCCEEDED',
      'FAILED_RETRYABLE', 'FAILED_TERMINAL', 'CANCELLED', 'LEASE_EXPIRED'
    )
  ),
  lease_owner text,
  lease_fencing_token bigint,
  lease_expires_at timestamptz,
  provider_run_id text,
  checkpoint jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  last_heartbeat_at timestamptz,
  next_attempt_due_at timestamptz,
  terminal_at timestamptz,
  result_id uuid
);
CREATE INDEX execution_attempts_lease_idx ON execution_attempts (lease_expires_at)
  WHERE status IN ('CLAIMED', 'DISPATCHED', 'WAITING');
ALTER TABLE execution_attempts ENABLE ROW LEVEL SECURITY;

-- Reported capability outcome (result.md). SUCCEEDED -> ACCEPT_WORKFLOW_RESULT;
-- FAILED/TIMED_OUT/PARTIAL -> REJECT_WORKFLOW_RESULT; INDETERMINATE -> neither.
CREATE TABLE results (
  result_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  result_type text NOT NULL DEFAULT 'INTELLIGENCE_TEXT_GENERATION',
  workflow_id uuid NOT NULL,
  objective_id uuid NOT NULL,
  intent_id uuid NOT NULL REFERENCES execution_intents (intent_id),
  attempt_id uuid NOT NULL REFERENCES execution_attempts (attempt_id),
  capability_request_id uuid NOT NULL,
  idempotency_key text NOT NULL,
  producing_principal_id uuid NOT NULL,
  provider_adapter text NOT NULL,
  model_id text NOT NULL,
  outcome text NOT NULL CHECK (outcome IN ('SUCCEEDED', 'FAILED', 'CANCELLED', 'TIMED_OUT', 'PARTIAL', 'INDETERMINATE')),
  output jsonb,
  error_classification text,
  retryable boolean,
  started_at timestamptz NOT NULL,
  observed_at timestamptz NOT NULL,
  reported_at timestamptz NOT NULL DEFAULT now(),
  accepted boolean,
  decided_at timestamptz,
  UNIQUE (organization_id, idempotency_key)
);
ALTER TABLE results ENABLE ROW LEVEL SECURITY;

-- Governance decision record (governance.md): exactly one of
-- ALLOW/DENY/REQUIRE_APPROVAL per governed command or dispatch re-check.
CREATE TABLE governance_decisions (
  decision_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  request_id uuid NOT NULL,
  correlation_id uuid NOT NULL,
  principal_id uuid NOT NULL,
  action text NOT NULL,
  resource_type text NOT NULL,
  resource_id text NOT NULL,
  proposal_digest text NOT NULL,
  trusted_context_digest text NOT NULL,
  policy_version text NOT NULL,
  autonomy_level text NOT NULL,
  outcome text NOT NULL CHECK (outcome IN ('ALLOW', 'DENY', 'REQUIRE_APPROVAL')),
  matched_rule_id text,
  reason text,
  decided_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE governance_decisions ENABLE ROW LEVEL SECURITY;

-- Command awaiting a REQUIRE_APPROVAL resolution (command.md). Unexercised
-- by this slice's always-ALLOW policy, but implemented per application.md's
-- "the stage may be trivial, but it is never bypassed."
CREATE TABLE pending_commands (
  pending_command_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  status text NOT NULL CHECK (status IN ('PENDING', 'RESOLVED', 'EXPIRED', 'CANCELLED')),
  command_type text NOT NULL,
  proposal_digest text NOT NULL,
  governed_payload jsonb NOT NULL,
  expected_workflow_id uuid,
  expected_workflow_version bigint,
  governance_decision_id uuid NOT NULL REFERENCES governance_decisions (decision_id),
  approval_id uuid,
  request_id uuid NOT NULL,
  idempotency_key text NOT NULL,
  correlation_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz,
  closed_at timestamptz,
  closure_reason text
);
ALTER TABLE pending_commands ENABLE ROW LEVEL SECURITY;

CREATE TABLE approvals (
  approval_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL,
  pending_command_id uuid NOT NULL REFERENCES pending_commands (pending_command_id),
  requesting_principal_id uuid NOT NULL,
  action text NOT NULL,
  resource_type text NOT NULL,
  resource_id text NOT NULL,
  proposal_digest text NOT NULL,
  status text NOT NULL CHECK (
    status IN ('PENDING', 'APPROVED', 'CONSUMED', 'REJECTED', 'EXPIRED', 'CANCELLED', 'REVOKED')
  ),
  decided_by_principal_id uuid,
  decided_at timestamptz,
  reason text,
  created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE approvals ENABLE ROW LEVEL SECURITY;

-- Idempotent-replay store: retrying the same logical request must return
-- the original outcome, never create a second transition (application.md).
CREATE TABLE idempotency_keys (
  organization_id uuid NOT NULL,
  idempotency_key text NOT NULL,
  request_id uuid NOT NULL,
  outcome text NOT NULL,
  result_ref jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (organization_id, idempotency_key)
);
ALTER TABLE idempotency_keys ENABLE ROW LEVEL SECURITY;
