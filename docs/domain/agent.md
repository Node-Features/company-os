# Agent Domain

Status: APPROVED (2026-08-23, project owner `Node-Features`)

Reopened from `APPROVED` to `DRAFT` on 2026-08-23 for one material addition — a display avatar on
`AgentDefinition` (see "Display identity" below) — per this repo's own rule that a material
contract change returns a document to `DRAFT` until re-reviewed, then re-approved the same day
after reconciling the addition against `artifact.md` and `capability.md`'s actual contracts. Every
other section is unchanged from the version the project owner originally approved 2026-08-20.

## Definition

An `Agent` is a CompanyOS-managed computational participant assigned a bounded organizational purpose and operating through an `AgentPrincipal`. The Agent describes accountable behavior and participation; its Principal supplies durable identity. An Agent is not a model, provider, prompt, conversation, session, Runtime attempt, department, or source of authority.

## AgentDefinition

An immutable AgentDefinition contains:

- stable definition identity/version, name, purpose, responsibilities, and non-responsibilities;
- a display avatar: an [Artifact](artifact.md) reference to a generated image representing the
  Agent in UI surfaces (see "Display identity" below) — optional, presentational only;
- eligible organization and role or department-membership references;
- required and provided Capability identities/versions;
- accepted task/input and produced Result/Evidence contracts;
- allowed tool/action categories and mandatory Governance mappings;
- default ResourceConstraint, data-classification, workspace, and supervision requirements;
- accountable owner role, escalation conditions, and required human-review boundaries;
- lifecycle status, compatibility range, and supersession references.

AgentDefinition declares limits and requirements. It does not embed credentials, provider selection, mutable authority, or unrestricted tool lists.

## Display identity

An Agent's `name` and avatar exist to make it recognizable to humans in UI surfaces — the
[Department & Agent directory](../architecture/ui-ux.md) is the intended consumer — and carry no
governance meaning. The avatar is produced like any other AI-generated work product: through a
governed [Capability](capability.md) request, with its output persisted as an
[Artifact](artifact.md) and referenced by ID from `AgentDefinition`, the same pattern this system
already uses for every other generated asset. `AgentDefinition` stores the reference, never image
bytes or a raw provider URL directly — and only an `ACCEPTED` Artifact version may be referenced,
per `artifact.md`'s own invariant that "storage does not imply acceptance": a merely `CANDIDATE`
generated image is not yet a valid avatar reference.

Name and avatar are versioned with `AgentDefinition` like every other field — changing either is a
new `AgentDefinition` version, not a mutation of the active one, so historical UI/audit views can
still render an Agent exactly as it appeared when a given action was taken.

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
- Name and avatar are presentational only — changing either can never change Authority, Governance evaluation, execution eligibility, or which Principal an Agent operates through.

## OPEN QUESTIONS

- Which AgentDefinition, if any, is required by the first vertical slice?
- Which accountable-owner and supervision requirements vary by risk class?
- OPEN QUESTION: Is an avatar mandatory at Agent registration, or may an Agent go `ACTIVE` without
  one (falling back to a generic UI placeholder)? Not decided here.
- OPEN QUESTION: Which Capability generates the avatar image, and under what governed Action —
  reuses the existing image-generation-capability question `ROADMAP.md` Phase 11 Slice 2 will need
  to resolve when it implements this field for real.

## Dependencies

- [Organization](organization.md)
- [Principal](principal.md)
- [Capability](capability.md)
- [Evidence](evidence.md)
- [Result](result.md)
- [Resource](resource.md)
- [Artifact](artifact.md) — avatar image storage
