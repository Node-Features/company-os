# Capability Domain

Status: APPROVED

## Definition

A `Capability` is an immutable, versioned, provider-independent contract for an outcome CompanyOS may request. It defines what must be supplied, produced, constrained, evidenced, and failed—not which department, agent, model, tool, SDK, or provider performs it.

## CapabilityDefinition

A CapabilityDefinition contains:

- stable capability identity, semantic version, name, purpose, and accountable contract owner;
- supported input and output schemas with compatibility rules;
- required output, Artifact, and Evidence characteristics;
- preconditions, postconditions, acceptance criteria, and typed failure taxonomy;
- idempotency, cancellation, timeout, retryability, and reconciliation semantics;
- security classification, privacy requirements, risk class, and data-boundary requirements;
- Governance Action/Resource mapping and required authority context;
- supported ResourceConstraint types and required usage evidence;
- lifecycle status, effective interval, deprecation, and supersession references.

Provider-specific options cannot appear in a CapabilityDefinition. A material change to outcome meaning, schemas, evidence, safety, or failure semantics creates a new version.

## CapabilityRequest

A `CapabilityRequest` binds one active CapabilityDefinition version to an organization, objective, workflow, execution intent, requesting Principal, normalized inputs, constraints, governance-decision reference, idempotency identity, due/expiry conditions, and required result evidence. It names no concrete provider or model.

Application coordinates validation of the organizational request. The Kernel decides whether the resulting transition may include an ExecutionIntent, and Application coordinates atomic persistence of that intent with the accepted state and events. Runtime may dispatch only the persisted intent through an eligible implementation selected by the applicable router or registry.

## Implementation eligibility

An implementation registration states which CapabilityDefinition versions an adapter or internal executor can satisfy and references current compatibility, security, reliability, quality, cost, and operational evidence. Registration establishes technical eligibility only; Governance, routing, resources, and Runtime availability still apply.

## Invariants

- Departments, agents, workflows, and routers depend on Capability identities and versions, never provider SDK types.
- Capability definition, implementation registration, routing decision, execution attempt, and accepted Result remain distinct records.
- A provider claim cannot activate a Capability or prove compatibility without validated evidence.
- Runtime cannot relax inputs, constraints, acceptance criteria, authority requirements, or failure meaning during dispatch or retry.
- Fallback preserves the logical request and idempotency identity and creates attributable routing and attempt evidence.
- Capability success is a reported Result until an Application/Kernel transition accepts it.

## OPEN QUESTIONS

- Which single CapabilityDefinition proves the first vertical slice?
- What compatibility policy applies to non-breaking schema evolution?

## Dependencies

- [Organization](organization.md)
- [Artifact](artifact.md)
- [Evidence](evidence.md)
- [Resource](resource.md)
