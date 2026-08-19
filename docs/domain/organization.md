# Organization Domain

Status: DRAFT

## Definition

An `Organization` is the authoritative CompanyOS boundary within which identity, authority, policies, objectives, departments, workflows, resources, knowledge, and audit records have meaning. It is the tenant and governance scope, not a deployment, database, repository, provider account, or department.

## Minimum contract

An Organization contains:

- globally unique stable organization identity and immutable creation identity;
- version, canonical name, lifecycle status, and recorded timestamps;
- mission, vision, principle, and non-goal document references;
- accountable owner Principal reference and governance-policy-set references;
- active department-registry and workflow/capability compatibility references;
- default security classification, residency, retention, and isolation requirements;
- resource-account and reporting-boundary references;
- supersession or closure reason and effective time when applicable.

External aliases, domains, repositories, billing accounts, provider tenants, and deployment identifiers are versioned bindings to the Organization; none replaces its identity.

## Lifecycle

The minimum lifecycle is `PROPOSED → ACTIVE → SUSPENDED → CLOSED`, with reactivation from `SUSPENDED` allowed only through a governed transition. `CLOSED` is terminal. Suspension blocks new governed work but preserves state, evidence, attribution, and recovery access required by policy.

Creation, activation, suspension, reactivation, and closure are Application/Kernel transitions with current Governance evidence and successful persistence.

## Isolation boundary

Every authoritative command, record, event, execution intent, Principal binding, policy evaluation, approval, resource constraint, artifact, and knowledge item names exactly one Organization unless an explicit governed cross-organization contract exists. Authentication and storage adapters fail closed when organization scope is missing, ambiguous, inactive, or inconsistent.

## Invariants

- Organization identity is stable and never inferred from hostname, repository, credential, provider tenant, or process configuration.
- An Organization cannot read, authorize, spend, execute, or publish through another Organization's scope implicitly.
- Provider and infrastructure tenancy cannot weaken CompanyOS organization isolation.
- Suspension or closure never erases historical identities, decisions, events, evidence, or provenance.
- Mission and policy changes are versioned references; they do not silently rewrite prior decisions.
- Cross-organization relationships are explicit, bounded, attributable, and governed on every participating side.

## OPEN QUESTIONS

- Is the first deployment single-organization while retaining mandatory organization IDs?
- Which owner-recovery and Organization-closure rules are required initially?

## Dependencies

- None.
