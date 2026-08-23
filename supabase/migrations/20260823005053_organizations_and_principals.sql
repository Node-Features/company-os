-- Phase 3 Slice 6: real Organization/Principal persistence.
-- Before this migration, Organization/Principal were pure Go literals in
-- internal/fixtures (first-slice plan decision #6). This is the first real
-- persistence for them — see docs/domain/organization.md and
-- docs/domain/principal.md.
--
-- Single-organization deployment (organization.md's own open question,
-- answered here): there is exactly one Organization, and it must keep
-- internal/fixtures.OrganizationID's existing value, since every historical
-- workflows/domain_events/etc. row already references that ID. So this is a
-- one-time seed of the existing ID into a real table, not a general
-- multi-tenant create-org path.
--
-- No FK from workflows/execution_intents/etc. to principals: those keep
-- using loose UUIDs exactly as before this migration (the Service/Human
-- fixture Principals have no row here and aren't meant to). Only a real
-- authenticated HumanPrincipal gets a row, via the onboarding flow this
-- migration supports.

CREATE TABLE organizations (
  organization_id uuid PRIMARY KEY,
  name text NOT NULL,
  status text NOT NULL CHECK (status IN ('ACTIVE', 'INACTIVE')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE organizations ENABLE ROW LEVEL SECURITY;

-- Seed the one real Organization at the same ID every existing Workflow
-- already references (internal/fixtures.OrganizationID), matching the
-- values internal/fixtures.NewRegistry's literal has used since Phase 1.
INSERT INTO organizations (organization_id, name, status)
VALUES ('00000000-0000-4000-8000-000000000001', 'CompanyOS Dev', 'ACTIVE')
ON CONFLICT (organization_id) DO NOTHING;

-- Durable Principal record (principal.md). Only HumanPrincipals created
-- through real sign-in get a row this slice — the Service/Human fixtures
-- (internal/fixtures.PrincipalID/ApproverPrincipalID) remain pure Go
-- fixtures, unpersisted.
CREATE TABLE principals (
  principal_id uuid PRIMARY KEY,
  organization_id uuid NOT NULL REFERENCES organizations (organization_id),
  kind text NOT NULL CHECK (kind IN ('HUMAN', 'AGENT', 'SERVICE', 'PROVIDER')),
  display_name text NOT NULL,
  external_issuer text NOT NULL,
  external_subject text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (external_issuer, external_subject)
);
ALTER TABLE principals ENABLE ROW LEVEL SECURITY;

-- PrincipalOrganizationBinding (principal.md's "Organization binding").
-- Minimal fields only — identity-assurance requirements, permitted
-- authentication methods, and role/membership references from the full
-- doc contract are not persisted this slice.
CREATE TABLE principal_organization_bindings (
  binding_id uuid PRIMARY KEY,
  principal_id uuid NOT NULL REFERENCES principals (principal_id),
  organization_id uuid NOT NULL REFERENCES organizations (organization_id),
  status text NOT NULL CHECK (status IN ('ACTIVE', 'SUSPENDED', 'REVOKED')),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (principal_id, organization_id)
);
ALTER TABLE principal_organization_bindings ENABLE ROW LEVEL SECURITY;
