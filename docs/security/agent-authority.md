# CompanyOS Agent Authority

Status: APPROVED

## Purpose

This document specifies agent permission and authority limits, grounding the Authority check in [`governance.md`](../architecture/governance.md)'s decision pipeline specifically for `AgentPrincipal` requests, which [`identity.md`](../architecture/identity.md)'s Agent flow leaves generic. It narrows [`ADR-0008`](../adr/ADR-0008-authority-capability-model.md)'s `Role`-based policy matching into a concrete, defensible starting posture, and answers that ADR's carried-forward open question: *"`docs/security/agent-authority.md` is where real per-department agent authority should be decided."*

## Default posture: zero standing authority

An agent has **no authority by default**. Every action an agent requests must match an explicit permit under [Governance's default-deny policy composition](../architecture/governance.md#policy-composition) — absence of a matching permit is `DENY`, not an implicit grant. This document does not create authority; it constrains how authority may be granted so no department invents its own ad hoc scheme.

## Scoping dimensions

An agent's effective authority is the intersection of every dimension below. Broadening any one dimension without the others does not broaden actual authority:

| Dimension | Meaning | Owning mechanism |
|---|---|---|
| Role | The named policy-matching identity (`ADR-0008`'s `policy.Role`) | Policy rules keyed by `(Role, Action)` |
| Organization | The active `PrincipalOrganizationBinding` | [Identity — Organization scoping](../architecture/identity.md#organization-scoping) |
| Department | The department whose workflow/objective context originated the request | [Departments — Dependency rule](../architecture/departments.md#dependency-rule) |
| Action class | `automatic`, `approval_required`, or `human_only` autonomy | [Governance — Decision pipeline](../architecture/governance.md#decision-pipeline) |
| Resource | The specific typed target, including organization/tenant ownership | [Governance — Decision request](../architecture/governance.md#decision-request) |
| Time/session | The current `AuthenticationSession`'s validity, not a stale grant | [Identity — Session identity](../architecture/identity.md#session-identity-versus-principal-identity) |

An agent's Role does not carry across departments: a `research_agent` Role has no implicit standing for `finance.*` actions, matching `ADR-0008`'s illustrative table (`research_agent` requesting `finance.transfer_funds` is `DENY` — wrong role, not merely no rule).

## Autonomy-class defaults by action shape

Rather than enumerate every action (which belongs in department-specific policy, not this document), this document fixes the *default classification shape* every department must justify a deviation from:

- **Read within own department scope** (e.g. `research.read_market_data`): eligible for `automatic`, subject to normal policy match.
- **Write/create within own department scope** (e.g. `research.create_report`): eligible for `automatic` only when the action is reversible and does not commit external effect (no message sent, no payment issued, no code merged).
- **External or irreversible effect** (e.g. `finance.transfer_funds`, `customer.send_message`, a merge in [Coding-agents](../architecture/coding-agents.md), a `knowledge.approve`): defaults to `approval_required` or `human_only`, never `automatic`, until a specific accepted ADR narrowly justifies otherwise — mirroring `governance.md`'s existing prohibition on deterministic automatic knowledge approval.
- **Cross-department action:** defaults to `DENY` — an agent does not gain standing in another department merely by referencing its objective or workflow, per [Departments — Dependency rule](../architecture/departments.md#dependency-rule)'s "must not import another department's implementation."

A department proposing a new action must classify it against this shape when writing its policy rules; a classification that deviates (e.g. an irreversible action marked `automatic`) requires its own reasoning recorded alongside the policy, not silent inclusion.

## No self-authorization, ever

Restating [System context — Agent](../architecture/system-context.md#agent) and [Governance invariants](../architecture/governance.md#invariants) as the binding rule for this document: an agent may propose commands and invoke permitted capabilities but can never grant itself authority, mutate authoritative state directly, or approve its own action. This is not a policy configuration an agent-authority rule can override — it is structural, enforced by Governance regardless of Role.

## Escalation and delegation

- An agent's authority can only be **narrowed** by delegation, never amplified — [Governance invariants](../architecture/governance.md#invariants): "Delegation narrows authority; it cannot amplify the delegator's authority."
- An agent requesting an action beyond its current Role's permits does not escalate itself; the request surfaces as `DENY` (no matching permit) or `REQUIRE_APPROVAL` (matched but gated), and only a human with sufficient standing Authority can supply the missing Approval evidence, per [Governance — Approval boundary](../architecture/governance.md#approval-boundary).
- Revocation of an agent's Role or Authority takes effect for the next Governance evaluation; dispatch-time re-evaluation ([`runtime.md`](../architecture/runtime.md)'s Application-mediated dispatch-authorization step) ensures an in-flight execution cannot outlive a revoked grant indefinitely.

## Relationship to ADR-0008's illustrative rules

`ADR-0008`'s `research_agent`/`finance_agent` rows are demonstration data, not this document's actual policy — that ADR says so explicitly. This document's job is the *shape* those rules must follow (default-deny, no cross-department standing, irreversible-effect gating); each department's concrete policy rules are that department's own artifact, reviewed against this shape, not duplicated here.

## Invariants

- No agent has standing authority outside its assigned Role, organization, and department scope.
- No policy may classify an irreversible or externally visible action as `automatic` without a recorded justification.
- No agent can be the sole authorizer of its own request, regardless of Role.
- A Role change or revocation is visible to the next Governance evaluation, not only to new sessions.
- Cross-department action requests default to `DENY` absent an explicit, reviewed cross-department contract.

## Open questions

- OPEN QUESTION (carried from `ADR-0008`): should `Role` become a real, persisted binding resolved through [`principal.md`](../domain/principal.md)'s delegation references before any production policy depends on it, given it is entirely caller-asserted today?
- OPEN QUESTION: what is the reviewed process for a department to justify an `automatic`-classified irreversible action, and who signs off?
- OPEN QUESTION: does a `ServicePrincipal` (non-agent, non-human) need its own narrower authority document, or is this document's shape sufficient by treating it as a degenerate Role?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [`threat-model.md`](threat-model.md)
- [Governance](../architecture/governance.md)
- [Identity](../architecture/identity.md)
- [Departments](../architecture/departments.md)
- [System context](../architecture/system-context.md)
- [`ADR-0008`](../adr/ADR-0008-authority-capability-model.md)
- [Principal domain](../domain/principal.md)
- [Policy domain](../domain/policy.md)
