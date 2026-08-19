# Policy Domain

Status: DRAFT

## Definition

A Policy is a versioned organizational rule that contributes to authorization or autonomy classification for a request. It is data governed by CompanyOS, not application code, a prompt instruction, a provider guardrail, or an approval.

## Governance vocabulary

### Principal

Policy evaluation uses the canonical [Principal domain contract](principal.md). Policy may reference a Principal and authenticated evidence about it, but it does not redefine Principal identity, type, organization scope, authentication, delegation, membership, or authority.

### Action

A stable, provider-independent operation that may be performed. An Action defines its input contract, target Resource types, risk classification, and evidence requirements. Tool names and API methods map to Actions but do not define them.

### Resource

The stable CompanyOS identity of the object an Action targets or affects. A Resource has a type, organization scope, ownership or hierarchy relationships, and the minimum attributes required for evaluation.

### Context

Trusted, request-specific facts not inherent to Principal, Action, or Resource. Context may include objective/workflow identity, time, environment, risk, cost, channel, proposed arguments digest, and prior persisted evidence. Untrusted agent claims are not trusted Context until verified.

### Authority

A versioned, revocable delegation defining the maximum Actions, Resources, contexts, budget, duration, and delegation depth available to a Principal. Authority is necessary but not sufficient: policy may further restrict it. Delegation cannot enlarge the delegator's Authority.

### AutonomyLevel

The required execution mode for an otherwise eligible request:

- `automatic`: an eligible Principal may execute after Governance returns `ALLOW`.
- `approval_required`: an eligible request needs valid human Approval before `ALLOW`.
- `human_only`: agents and services cannot execute; an authorized human Principal must initiate or execute the Action.

When multiple levels apply, the effective level is the most restrictive: `human_only` > `approval_required` > `automatic`.

### Decision

An immutable result of evaluating one normalized request against explicit policy, authority, autonomy, entity, and context versions:

- `ALLOW`: eligible for dispatch for its bounded validity period.
- `DENY`: prohibited or insufficiently proven; cannot execute.
- `REQUIRE_APPROVAL`: otherwise eligible, but missing valid approval evidence.

A Decision records reasons and matched rules. It is evidence, not a reusable grant unless explicitly designed as a narrowly bounded authorization token.

## Policy model

A Policy has:

- stable policy ID and immutable version;
- organization scope and status;
- human-readable purpose and owner;
- rule effect: permit, forbid, or autonomy requirement;
- Principal, Action, Resource, and Context constraints;
- priority only for deterministic administration, never to let permit override forbid;
- validity interval and supersession metadata;
- source, review, approval, and deployment evidence.

Policy lifecycle is `DRAFT -> VALIDATED -> ACTIVE -> SUPERSEDED` or `RETIRED`. Invalid or inactive policy cannot contribute to `ALLOW`. Exact lifecycle commands remain an OPEN QUESTION pending the organization and event domains.

## Composition rules

- No matching permit means `DENY`.
- Any matching forbid means `DENY`, regardless of permits or approvals.
- Otherwise, the most restrictive applicable AutonomyLevel governs.
- `approval_required` without matching valid Approval yields `REQUIRE_APPROVAL`.
- `human_only` requested by a non-human Principal yields `DENY`.
- Evaluation or validation uncertainty yields `DENY`.
- Emergency restrictions and organization isolation always apply.

## Invariants

- Policies refer to stable CompanyOS identities and action contracts, not provider-specific strings alone.
- Policy versions used by a Decision are immutable and auditable.
- Policies cannot directly execute actions or mutate workflow state.
- Policy administration is itself a governed Action.
- Authority and Approval can narrow or satisfy requirements but cannot override a forbid.
- Autonomy classification never grants eligibility absent a permit and sufficient Authority.
- A model or agent cannot authoritatively interpret free text as active policy.
- Cached evaluations are invalid after relevant policy, authority, resource, or context changes.

## Relationship to architecture

[Governance](../architecture/governance.md) evaluates this domain model. A Cedar adapter may translate CompanyOS Principal, Action, Resource, Context, entities, and permit/forbid policies into Cedar requests, but the CompanyOS model remains canonical.

## Open questions

- OPEN QUESTION: What is the canonical Policy identifier and version format?
- OPEN QUESTION: Which policy lifecycle roles validate and activate changes?
- OPEN QUESTION: Are autonomy requirements represented in a separate policy set or as CompanyOS policy metadata?
- OPEN QUESTION: Which policy changes invalidate pending Approvals?
- OPEN QUESTION: What static analysis and simulation are mandatory before activation?

## Dependencies

- [Organization](organization.md)
- [Principal](principal.md)
