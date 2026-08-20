# Agent Domain

Status: APPROVED

## Definition

An `Agent` is a CompanyOS-managed computational participant assigned a bounded organizational purpose and operating through an `AgentPrincipal`. The Agent describes accountable behavior and participation; its Principal supplies durable identity. An Agent is not a model, provider, prompt, conversation, session, Runtime attempt, department, or source of authority.

## AgentDefinition

An immutable AgentDefinition contains:

- stable definition identity/version, name, purpose, responsibilities, and non-responsibilities;
- eligible organization and role or department-membership references;
- required and provided Capability identities/versions;
- accepted task/input and produced Result/Evidence contracts;
- allowed tool/action categories and mandatory Governance mappings;
- default ResourceConstraint, data-classification, workspace, and supervision requirements;
- accountable owner role, escalation conditions, and required human-review boundaries;
- lifecycle status, compatibility range, and supersession references.

AgentDefinition declares limits and requirements. It does not embed credentials, provider selection, mutable authority, or unrestricted tool lists.

## Agent record

An Agent record binds one Organization, AgentPrincipal, AgentDefinition version, accountable owner Principal, memberships, active lifecycle version, and provenance. Optional objective or workflow assignment is a separate bounded relationship and cannot broaden the definition or delegated Authority.

The minimum lifecycle is `REGISTERED → ACTIVE → SUSPENDED → RETIRED`. Reactivation from `SUSPENDED` requires current definition compatibility, identity evidence, and a governed transition. Retirement is terminal but preserves attribution.

## Execution boundary

An Agent may propose commands and request Capabilities through Application use cases. Governance evaluates every governed Action using current Principal, Authority, policy, Approval, and context evidence. Runtime and provider sessions execute bounded work but cannot change the Agent record or authoritative workflow state directly.

## Invariants

- Every Agent maps to exactly one durable AgentPrincipal and Organization binding at a time.
- Agent identity, definition, membership, Authority, assignment, and execution session remain distinct versioned records.
- An Agent cannot create or enlarge its Authority, approve its own action, choose a prohibited provider, or bypass Application orchestration.
- Prompt text, model output, provider configuration, or human initiation cannot change Agent identity or authority.
- Suspension, revocation, cancellation, or lease expiry blocks new execution under the affected scope; late results remain evidence only.
- Agents communicate through shared contracts; direct department implementation dependencies are prohibited.
- Agent messages and memory are non-authoritative unless accepted by the owning domain operation.

## OPEN QUESTIONS

- Which AgentDefinition, if any, is required by the first vertical slice?
- Which accountable-owner and supervision requirements vary by risk class?

## Dependencies

- [Organization](organization.md)
- [Principal](principal.md)
- [Capability](capability.md)
- [Evidence](evidence.md)
- [Result](result.md)
- [Resource](resource.md)
