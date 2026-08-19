# CompanyOS Governance Architecture

Status: DRAFT

## Responsibility

Governance is the mandatory decision boundary between a requested action and its execution. Agents, humans, departments, workflows, and services may request actions. Governance determines whether a specific principal may execute a specific action on a specific resource in the supplied context.

Governance owns:

- constructing and validating authorization requests;
- resolving applicable policy and delegated authority at explicit versions;
- default-deny policy evaluation and restrictive policy composition;
- assigning the effective autonomy level;
- returning exactly `ALLOW`, `DENY`, or `REQUIRE_APPROVAL`;
- creating, resolving, binding, consuming, expiring, and revoking approvals;
- producing durable, attributable decision and approval evidence;
- re-evaluating governed actions immediately before dispatch.

Governance consumes authenticated Principal evidence through the [Identity contract](identity.md). It does not authenticate credentials, own Principal lifecycle, resolve arbitrary external subjects, create trusted claims, or manage authentication sessions.

Governance does not own policy meaning, approval state semantics, identity authentication, workflow legality, application orchestration, execution mechanics, provider behavior, or persistence technology. Domain definitions own meaning; Identity proves the requesting [Principal](../domain/principal.md); the Kernel owns legal transitions; the [Application layer](application.md) coordinates use cases; Runtime executes only persisted, authorized intent.

## Decision request

Every governance request contains:

| Input | Required meaning |
|---|---|
| Principal | Durable human, agent, service, or provider identity and active organization binding resolved through the Principal domain |
| Authentication evidence | Immutable `AuthenticatedPrincipalEvidence` identity from the Identity boundary, bound to Principal, organization, audience, assurance, time, and request/channel context |
| Action | Stable, provider-independent operation being requested |
| Resource | Stable target identity and type, including organization/tenant ownership |
| Context | Trusted request facts such as objective, workflow, time, risk, budget, channel, environment, and proposed arguments digest |
| Authority | Active delegation that bounds what the principal may request |
| Policies | Validated, applicable policy set at recorded versions |
| Approval evidence | Optional resolved approval bound to this exact request |

Absent or unverifiable required input produces `DENY`. Provider names and tool strings may be context or adapter data, but cannot replace stable Action and Resource identities.

## Decision pipeline

1. Validate Identity-issued authentication evidence and resolve the exact active Principal and organization-binding versions; invalid, stale, revoked, ambiguous, or insufficient evidence is `DENY`.
2. Validate Action, Resource, trusted Context/attestations, entity data, policy syntax/schema, and organization scope; uncertainty is `DENY`.
3. Confirm the Principal has active Authority covering the request; missing, expired, revoked, or exceeded authority is `DENY`.
4. Evaluate applicable authorization policies using default deny: at least one permit must match and no forbid may match. Evaluation errors fail closed.
5. Compose applicable autonomy requirements using the most restrictive result: `human_only` > `approval_required` > `automatic`.
6. If effective autonomy is `human_only`, an agent or service request is `DENY`; an eligible human principal continues.
7. If effective autonomy is `approval_required`, return `REQUIRE_APPROVAL` unless valid unconsumed approval evidence exactly matches the request.
8. Re-evaluate policy, authority, context, and approval at dispatch time. Only then return `ALLOW`.

`REQUIRE_APPROVAL` means the request is otherwise eligible but lacks valid approval evidence. It is not a weak allow and must never be sent to an executor.

## Policy composition

- Default is deny: absence of a matching permit never implies access.
- Any matching forbid overrides permits and approvals.
- Policy evaluation errors, missing entities, unsupported actions, stale policy snapshots, and ambiguous identity fail closed.
- Multiple autonomy requirements compose to the most restrictive level.
- Approval cannot override a forbid, create authority, change organization scope, or legalize an invalid Kernel transition.
- Emergency pause or kill is an overriding deny condition.
- Policies are immutable by version once used in a decision; changes create a new version.

## Approval boundary

An approval binds the approver, requesting principal, action, resource, normalized arguments digest, objective/workflow context, policy and authority versions, expiry, and allowed uses. Approval resolution persists before execution resumes.

At execution, Governance verifies that the approval is approved, unexpired, unrevoked, unconsumed when single-use, issued by an authorized human, and still matches the request. Policy and authority are then evaluated again. Approving a request does not execute it; the Application layer records the new `ALLOW` decision for the exact intent before Runtime executes separately.

The Application layer constructs and submits the Governance request and persists the decision reference with the resulting state or intent. It cannot modify the request after evaluation, reinterpret the outcome, or make an authorization decision. Immediately before a governed external effect, Runtime requests dispatch authorization through an Application use case so Governance evaluates current evidence for the exact persisted intent.

## Knowledge approval boundary

Knowledge approval is a governed `knowledge.approve` Action on a Resource containing the exact organization, knowledge scope, KnowledgeItem identity/version, and content/source digest. While deterministic automatic approval is disabled, applicable policy classifies this Action as `human_only` and Governance returns `DENY` for agent, service, or provider Principals.

For a human reviewer, Governance verifies current `AuthenticatedPrincipalEvidence`; active reviewer Authority for the knowledge class and scope; authorship and separation-of-duties constraints; required expertise or role evidence; item status and immutable digest; source/evidence requirements; organization boundary; policy versions; and any additional Approval evidence. Only `ALLOW` may support the separate Kernel transition to `APPROVED`.

`REQUIRE_APPROVAL` means the review action needs further human authorization and leaves the KnowledgeItem unapproved. `DENY`, stale evidence, item mutation, changed sources, or changed reviewer/Authority/policy context requires a new evaluation. Governance records reviewer, authentication evidence, Authority, policies, determining rules, item digest, outcome, reasons, and time.

Deterministic automatic knowledge approval remains prohibited until a dedicated ADR is accepted and applicable policy is activated. The ADR must narrowly identify eligible knowledge classes and establish reproducibility, source eligibility, validation, failure, accountability, rollback, and audit requirements. A general `automatic` autonomy level, model confidence, test result, deterministic derivation, or provider assertion is insufficient.

## Auditability

Each evaluation records a decision identifier; request and correlation identifiers; principal/action/resource; trusted context digest; policy, authority, and autonomy versions; outcome; matched permit/forbid/requirement identifiers; evaluation errors; approval identifier and approver when applicable; timestamps; and eventual execution evidence correlation.

Sensitive arguments are represented by canonical digests plus governed references rather than copied into audit logs. Audit records are append-only organizational evidence; application logs and provider traces do not replace them.

## Invariants

- Agents may request actions; Governance determines whether they may execute.
- No governed action reaches an executor without a current persisted `ALLOW` decision.
- `REQUIRE_APPROVAL` and `DENY` never authorize execution.
- No matching permit, any matching forbid, or any evaluation uncertainty results in `DENY`.
- Approval is explicit human evidence, never inferred from silence, prior conversation, or repeated requests.
- The requester cannot approve its own action unless a future policy explicitly permits a human acting in both roles; agents can never self-approve.
- Approval is bound to immutable request content and cannot be reused after material changes.
- Approval cannot exceed the approver's authority.
- Policy, authority, approval, and decision records are persisted before dependent execution continues.
- Execution uses the same organization scope and request identity that Governance evaluated.
- Every decision references the authenticated-evidence, durable-Principal, organization-binding, and assurance versions evaluated.
- Authentication success never implies authorization, and Governance cannot weaken Identity assurance or accept raw caller claims as trusted evidence.
- Delegation narrows authority; it cannot amplify the delegator's authority.
- Policy or authority changes invalidate cached decisions and require re-evaluation.
- Provider guardrails may add protection but cannot replace CompanyOS Governance.
- Knowledge becomes `APPROVED` only from a Governance-authorized review of the exact immutable item version; agents, services, providers, and models cannot serve as the reviewer.
- Deterministic automatic knowledge approval is disabled until a dedicated accepted ADR and active policy explicitly permit narrowly defined cases.

## OSS evidence

### Cedar

At pinned revision `d67b1721273951c0eddd9bafb7a5b8c77ee161f4`, Cedar's [`Request`](https://github.com/cedar-policy/cedar/blob/d67b1721273951c0eddd9bafb7a5b8c77ee161f4/cedar-policy-core/src/ast/request.rs) models principal, action, resource, and context. The [`Authorizer`](https://github.com/cedar-policy/cedar/blob/d67b1721273951c0eddd9bafb7a5b8c77ee161f4/cedar-policy-core/src/authorizer.rs) evaluates a `PolicySet` against entity data and returns binary allow/deny plus diagnostics. Its tests in [`public_interface.rs`](https://github.com/cedar-policy/cedar/blob/d67b1721273951c0eddd9bafb7a5b8c77ee161f4/cedar-policy/tests/public_interface.rs) demonstrate request construction, default deny, permits, forbids, context, and determining-policy diagnostics. Schema and validator code under [`validator`](https://github.com/cedar-policy/cedar/tree/d67b1721273951c0eddd9bafb7a5b8c77ee161f4/cedar-policy-core/src/validator) constrain entity/action shapes before deployment.

CompanyOS should borrow the typed request tuple, explicit entities and hierarchies, default deny, forbid-overrides, policy validation, pure evaluation, and determining-policy diagnostics. It should not encode approval as a Cedar `permit`, treat partial/unknown evaluation as approval eligibility, or make Cedar entities the authoritative CompanyOS domain model. Cedar has only permit/forbid effects; `REQUIRE_APPROVAL` is a CompanyOS lifecycle outcome composed after successful eligibility evaluation. Cedar remains a candidate evaluator, not an approved dependency.

### Approval and guardrail references

- OpenAI Agents SDK separates guardrail tripwires in [`guardrail.ts`](https://github.com/openai/openai-agents-js/blob/2d68a10f8c1593f37a8e291e7bce00634ba3e5dd/packages/agents-core/src/guardrail.ts), per-tool `needsApproval` in [`tool.ts`](https://github.com/openai/openai-agents-js/blob/2d68a10f8c1593f37a8e291e7bce00634ba3e5dd/packages/agents-core/src/tool.ts), approval interruptions in [`items.ts`](https://github.com/openai/openai-agents-js/blob/2d68a10f8c1593f37a8e291e7bce00634ba3e5dd/packages/agents-core/src/items.ts), and serializable approve/reject state in [`runState.ts`](https://github.com/openai/openai-agents-js/blob/2d68a10f8c1593f37a8e291e7bce00634ba3e5dd/packages/agents-core/src/runState.ts). Borrow interruption and resumable approval records; reject agent-run state as organizational authority and provider guardrails as policy.
- LangGraph.js [`interrupt.ts`](https://github.com/langchain-ai/langgraphjs/blob/a86f813954e010fbf30711c37baa5c53444613d5/libs/langgraph-core/src/interrupt.ts) and [interrupt tests](https://github.com/langchain-ai/langgraphjs/blob/a86f813954e010fbf30711c37baa5c53444613d5/libs/langgraph-core/src/tests/python_port/interrupt.test.ts) show checkpointed pause and explicit `Command` resume. Borrow durable pause/resume mechanics; reject arbitrary resume values as sufficient approval evidence.
- JARVIS separates an [`AuthorityEngine`](https://github.com/vierisid/jarvis/blob/6e144520c747a6e0b8673ba9b75769d5d5f10a9c/src/authority/engine.ts), persisted [`ApprovalManager`](https://github.com/vierisid/jarvis/blob/6e144520c747a6e0b8673ba9b75769d5d5f10a9c/src/authority/approval.ts), [`AuditTrail`](https://github.com/vierisid/jarvis/blob/6e144520c747a6e0b8673ba9b75769d5d5f10a9c/src/authority/audit.ts), and deferred execution. Borrow three outcomes, conditional approval transitions, emergency gates, and decision/execution correlation. Reject authority floors that can broaden agent privilege, first-match rule ambiguity, mutable in-memory policy as authority, and approval records containing unrestricted raw tool arguments.

## Open questions

- OPEN QUESTION: Is Cedar the first policy evaluator, or should the initial slice implement the CompanyOS contract without it?
- OPEN QUESTION: Which roles may approve each action/resource class, and may dual control be required?
- OPEN QUESTION: Which context fields are trusted facts, who attests them, and how are canonical argument digests produced?
- OPEN QUESTION: Are approvals single-use by default, and which low-risk cases permit bounded reuse?
- OPEN QUESTION: How are policy deployment, rollback, simulation, and separation of duties governed?
- OPEN QUESTION: What retention and confidentiality rules apply to governance evidence?
- OPEN QUESTION: Should any deterministic knowledge class be eligible for automatic approval under a future dedicated ADR?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [System context](system-context.md)
- [Kernel](kernel.md)
- [Identity](identity.md)
- [Principal domain](../domain/principal.md)
- [Policy domain](../domain/policy.md)
- [Approval domain](../domain/approval.md)
- Future agent and security contracts
