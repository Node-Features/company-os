# ADR-0002: Pluggable Departments

Status: APPROVED

## Context

CompanyOS organizes work into departments (Research, Monitoring & Evaluation, Finance, Design, Engineering, Deployment, Education & Public Engagement, and future ones) that must be addable, replaceable, disabled, or retired without redesigning the Kernel each time, and that must not gain privileged coupling to each other's internals merely by shipping in the core distribution. [`departments.md`](../architecture/departments.md) and [`department.md`](../domain/department.md) already specify this extension contract in detail as drafts; this record proposes accepting it.

## Proposed decision

Adopt the `DepartmentDefinition` / `DepartmentRegistry` extension-contract model already specified across the referenced documents:

1. A department is declared through an immutable, versioned `DepartmentDefinition` manifest (mission, responsibilities/non-responsibilities, roles, capabilities, workflows, policies, metrics, event subscriptions/emissions, knowledge scope, compatibility) — never through an executable provider object, credential, or direct reference to another department's implementation.
2. `DepartmentRegistry` is the sole authoritative catalog: it validates a definition, resolves its referenced contracts, rejects conflicting or cyclic ownership, and only then activates it through a governed Kernel transition. The canonical lifecycle is `REGISTERED -> VALIDATED -> ACTIVE -> DISABLED -> RETIRED`, with a `REGISTERED -> REJECTED` branch, per the lifecycle reconciled between `department.md` and `departments.md` on 2026-08-20.
3. Departments coordinate only through shared, versioned contracts — events, workflow contracts, capability requests, artifact/evidence references, and the canonical Evaluation/Metric/ResourceConstraint contracts — never by importing another department's implementation, storage schema, internal commands, agents, or provider adapters.
4. Research, Monitoring & Evaluation, and Finance form the first cross-department coordination pattern (the adaptive feedback loop), gated by the Objective-creation gate: a Finding, Recommendation, Evaluation, or ResourceEvaluation may only *propose* an Objective through a distinct, separately governed Application request — none may create one directly.

## Consequences

### Positive

- A new department — core or third-party — registers through the same contract and governance gates as every existing one; adding it does not require changing Kernel fundamentals.
- Disabling or retiring a department preserves its historical evidence, approvals, and knowledge provenance rather than erasing organizational memory.
- The adaptive loop (Research discovers, M&E evaluates, Finance prices) lets evidence from execution influence future objectives without granting any single department, agent, or model unilateral organizational authority.
- Department boundaries are independently auditable: because coordination is contract-only, a reviewer can check "did this department import another's implementation?" mechanically rather than by inspecting business logic.

### Costs and risks

- Registry validation and governed activation add process overhead compared to ad hoc module loading or a simple plugin-discovery mechanism.
- Shared-contract-only coordination is slower to build against than direct in-process calls, especially before the shared contracts (events, workflows, capabilities) a new department needs already exist.
- A carelessly scoped `DepartmentCapability` could still create de facto coupling between two departments' behavior even while nominally respecting the contract boundary, if reviews aren't disciplined about it.

## Alternatives rejected by this proposal

- **Departments as ordinary software plugins with direct extension points:** rejected — `departments.md`'s OSS evidence explicitly borrows Backstage's and VS Code's declarative-registration patterns while rejecting the idea that department ownership is equivalent to a software plugin exposing arbitrary extension points.
- **One shared module or team codebase with informal boundaries:** rejected because it permits silent privileged coupling and violates the invariant that departments never depend directly on another department's implementation.
- **A central orchestrator agent coordinates departments directly:** rejected — `departments.md`'s OSS evidence explicitly rejects JARVIS's centralized-orchestrator-as-source-of-truth pattern; the orchestrator would become the de facto authority instead of the Kernel.
- **Feedback results (Findings, Evaluations, ResourceEvaluations) create Objectives automatically:** rejected because it would let Research, M&E, or Finance unilaterally commit the organization to a course of action without a distinct governed decision; the Objective-creation gate exists specifically to prevent this.

## Acceptance criteria

- [x] `departments.md` and `department.md` are internally consistent — verified by the fresh read-only architecture audit completed 2026-08-20, which found no `CRITICAL` or `MAJOR` contradiction between them after the department-lifecycle reconciliation (see [Audit finding severity](../../AGENTS.md#audit-finding-severity)).
- [x] `departments.md` and `department.md` are themselves `APPROVED` architecture and domain documents (2026-08-20), not merely mutually consistent drafts.
- [x] the project owner reviews and explicitly changes `Status: PROPOSED` to `Status: APPROVED`.

Which concrete department(s) prove this contract in the first vertical slice remains an open implementation choice and is not required before accepting the extension-contract shape itself.

## Open questions

- OPEN QUESTION: Are `DepartmentDefinitions` loaded from signed packages, repository manifests, persisted configuration, or more than one source?
- OPEN QUESTION: Which registry owns each shared contract type and its compatibility rules?
- OPEN QUESTION: Can multiple implementation packages satisfy one `DepartmentDefinition`, or is definition plus implementation a deployable unit?
- OPEN QUESTION: Which failures disable only one department versus making CompanyOS unready?
