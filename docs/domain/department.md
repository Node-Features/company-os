# Department Domain

Status: APPROVED

**Implementation status:** this contract is doc-only — `internal/domain/department` has no Go types yet, and no `DepartmentRegistry`/`DepartmentMembership` exists anywhere in the codebase. Today's real pattern is three independent, hand-duplicated package sets (`internal/domain/{research,monitoringevaluation,finance}` + `internal/departments/{research,monitoringevaluation,finance}`) with no shared registry binding them (`ROADMAP.md` Phase 11 Slice 0 is the first slice that would build one; `docs/audit/backlog-p2-p4.md`'s "doc-only stubs" row). `Status: APPROVED` records that this document's design was reviewed and accepted, not that it is implemented.

## Definition

A Department is a stable, organization-scoped unit of responsibility. It owns a bounded mission and participates in CompanyOS through registered contracts. It is not an agent, team chat, provider, package, process, database, workflow, or deployment.

## DepartmentDefinition

`DepartmentDefinition` is the immutable versioned declaration of a Department's identity and public contract. It contains:

- stable department ID, definition version, display name, and mission;
- responsibilities and non-responsibilities;
- roles and membership constraints;
- provided and required DepartmentCapabilities;
- owned and participated workflow contracts;
- governing policy references;
- owned metrics and accountability relationships;
- subscribed and emitted event contracts;
- readable, proposable, curatable, and administrable knowledge scopes;
- CompanyOS contract compatibility and definition digest.

Detailed field validation and registration behavior are owned by [Department Architecture](../architecture/departments.md).

## DepartmentRegistry

`DepartmentRegistry` is the authoritative organization-scoped catalog of DepartmentDefinition versions and lifecycle state. It validates references and compatibility, prevents conflicting ownership, records evidence, and resolves the active definition used by application and Runtime operations.

The registry exposes domain operations rather than mutable maps:

- register an immutable definition version;
- record validation success or rejection reasons;
- activate one compatible version through a governed transition;
- disable or retire a version without erasing history;
- resolve active and historical definitions by stable identity;
- report dependents that block disablement, retirement, or migration.

Package discovery and registry mutation are separate. Discovery proposes a definition; only authorized, persisted operations change registry state.

## DepartmentMembership

`DepartmentMembership` is a versioned relationship assigning a human or agent Principal to a Department role for a bounded organization, time, and authority scope. It records:

- membership ID, Department ID, and definition version;
- Principal ID and role ID;
- status, validity interval, issuer, and revocation evidence;
- delegated Authority reference and budget/resource constraints;
- optional reporting relationship and separation-of-duties constraints.

Membership lifecycle is proposed as `PROPOSED -> ACTIVE -> SUSPENDED -> ENDED`, with `PROPOSED -> REJECTED` and `ACTIVE -> REVOKED` alternatives. Membership alone grants neither Action permission nor capability execution; Governance evaluates Authority and policy for every governed action.

## DepartmentCapability

`DepartmentCapability` declares how a Department relates to a provider-independent Capability contract:

- stable Capability ID and compatible contract version range;
- direction: `PROVIDES` or `REQUIRES`;
- purpose and owning responsibility;
- supported input/output and evidence contract versions;
- required policy, authority, quality, cost, reliability, and data-handling constraints;
- cardinality and availability expectations;
- whether the requirement is mandatory or optional;
- declared degraded behavior when optional capability is unavailable.

`PROVIDES` means the Department accepts responsibility for satisfying or coordinating the contract; it does not name a provider. `REQUIRES` means the Department may request the outcome through CompanyOS routing; it does not create a direct dependency on the providing Department.

## Department lifecycle

The Department identity persists independently of any definition version or implementation package. Proposed lifecycle:

```text
REGISTERED -> VALIDATED -> ACTIVE -> DISABLED -> RETIRED
                  \-----------------> REJECTED
```

`VALIDATED` means the current DepartmentDefinition version has passed schema, reference, and compatibility validation but has not yet been authorized to accept work. Activation requires one validated compatible DepartmentDefinition. Disablement stops new work but preserves existing state and may require Runtime draining. Retirement prevents reactivation under that identity while preserving all historical evidence.

## Coordination rules

- Events communicate persisted facts, not commands disguised as facts.
- Workflow contracts coordinate governed multi-step responsibilities.
- Capability contracts request outcomes without choosing a Department or provider implementation.
- Shared Kernel abstractions carry stable identities and domain envelopes only.
- Artifacts and knowledge cross boundaries through governed references with provenance and access policy.
- A Department cannot read another Department's private state or mutate its aggregates.

## Core Department identities

The initial anticipated identities are Research, Monitoring & Evaluation, Design, Engineering, Deployment, Education & Public Engagement, and Finance. They have no privileged contract form. Future Departments register through the same DepartmentDefinition and DepartmentRegistry.

Names and final stable identifiers remain `OPEN QUESTION` until the organization identity convention is specified.

## Invariants

- Department identity is stable across renames, implementation replacements, and definition upgrades.
- Adding, disabling, or retiring a Department does not change Kernel fundamentals.
- Departments never depend directly on other Department implementations.
- All public dependencies are explicit, versioned contracts resolved during validation.
- Responsibilities have one declared owner or an explicit shared-accountability contract.
- Membership, role, capability, and workflow references must resolve against the active compatible definition.
- DepartmentCapabilities never contain provider SDK types or credentials.
- A required capability does not imply authority to invoke it.
- Events cannot directly mutate another Department's state.
- Historical workflows and evidence retain the DepartmentDefinition version that governed them.
- Registry activation and lifecycle changes are governed, persisted, and auditable.
- Third-party and core Departments obey identical domain validation rules.

## Open questions

- OPEN QUESTION: What canonical identifier format applies to Departments and roles?
- OPEN QUESTION: Can one Principal hold memberships in multiple Departments, and which separation-of-duties rules apply?
- OPEN QUESTION: How is shared accountability represented without ambiguous ownership?
- OPEN QUESTION: Does `PROVIDES` name organizational accountability only, executable routing eligibility, or both?
- OPEN QUESTION: Which lifecycle state owns draining and migration of active workflows?
- OPEN QUESTION: What compatibility policy governs DepartmentDefinition version ranges?

## Dependencies

- [Policy](policy.md)
- [Workflow](workflow.md)
- [Event](event.md)
- [Metric](metric.md)
- [Knowledge](knowledge.md)
- [Organization](organization.md)
- [Capability](capability.md)
- [Agent](agent.md)
