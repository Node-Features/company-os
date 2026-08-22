# CompanyOS Threat Model

Status: APPROVED

## Purpose

This document enumerates assets, trust boundaries, threats, and mitigations for CompanyOS. It does not introduce new architecture — every mitigation here cites an already-approved document that owns the actual mechanism. This document's job is to make the threat reasoning explicit and traceable, so [`governance.md`](../architecture/governance.md)'s `DENY` semantics and [`workspaces.md`](../architecture/workspaces.md)'s isolation claims rest on a stated threat model rather than ad hoc judgment, per `ROADMAP.md` Phase 2.

## Assets

| Asset | Why it matters | Owning document |
|---|---|---|
| Authoritative organizational state (Organization, Objective, Workflow, Policy, Approval, results, events) | The organization's legal record of what it is, intends, permitted, and did | [System context](../architecture/system-context.md#authoritative-organizational-state) |
| Governance decisions and Approval evidence | The record that an action was actually authorized | [Governance](../architecture/governance.md) |
| Principal identity and authentication evidence | Who did what — attributability depends on it | [Identity](../architecture/identity.md) |
| Provider/LLM credentials, workspace credentials, service credentials | Compromise gives an attacker the ability to act as CompanyOS externally | [Workspaces — Isolation and authority boundaries](../architecture/workspaces.md#isolation-and-authority-boundaries) |
| Source repositories and produced artifacts (Engineering) | Integrity of what CompanyOS ships | [Coding-agent architecture](../architecture/coding-agents.md) |
| Knowledge base content | Organizational learning that other decisions get built on | [Knowledge](../architecture/knowledge.md) |
| Audit/event history | The only durable evidence of what happened, for both security review and M&E | [Events](../architecture/events.md) |

## Trust boundaries

Reusing [System context — External actors and systems](../architecture/system-context.md#external-actors-and-systems) as the canonical boundary inventory rather than re-deriving one:

| Boundary | Untrusted side | Trusted side | Crossing mechanism |
|---|---|---|---|
| Human owner/operators → CompanyOS | Request content, claimed identity | Authenticated `HumanPrincipal` evidence | [Identity — Human flow](../architecture/identity.md#human) |
| Agent → CompanyOS | Agent-asserted role/intent, prompt content | Authenticated `AgentPrincipal` evidence, Governance decision | [Identity — Agent flow](../architecture/identity.md#agent), [Governance](../architecture/governance.md) |
| AI model providers → CompanyOS | Model output, token content | Nothing automatically — model output is untrusted input until validated | [System context — Authoritative organizational state](../architecture/system-context.md#authoritative-organizational-state) |
| Coding-agent providers → CompanyOS | Agent plans, commands, claimed success | Independently verified diff/test/build evidence | [Coding-agents — Invariants](../architecture/coding-agents.md#invariants) |
| Workspace execution → CompanyOS | Anything a task's code or tooling does inside the workspace | Filesystem/process/network/credential confinement | [Workspaces — Isolation and authority boundaries](../architecture/workspaces.md#isolation-and-authority-boundaries) |
| External data/research sources → CompanyOS | Any fetched content, feed, or document | Evaluated, provenance-tagged input | [System context](../architecture/system-context.md#external-actors-and-systems) |
| GitHub / deployment infra / publishing platforms → CompanyOS | Callback payloads, webhook events | Verified signatures/mutual auth per `ProviderPrincipal` flow | [Identity — Provider flow](../architecture/identity.md#provider) |
| companyd process boundary (in-process) | Nothing crosses this today — Kernel/Runtime/Daemon co-located per ADR-0004 | N/A | [ADR-0004](../adr/ADR-0004-first-slice-technology-stack.md) |

## Threats and mitigations

Organized by boundary, using a STRIDE-derived lens (Spoofing, Tampering, Repudiation, Information disclosure, Denial of service, Elevation of privilege) without inventing new controls beyond what's already specified.

### Spoofing (false identity)

- **Threat:** A request claims to be a different Principal, or an agent claims a role/authority it wasn't issued.
- **Mitigation:** Authentication evidence is type-specific and unforgeable by claim alone — model identity, prompt contents, conversation ID, process location, and environment variables are explicitly insufficient per [Identity — Authentication flows by Principal type](../architecture/identity.md#authentication-flows-by-principal-type). Unknown/indeterminate verification fails closed.
- **Residual risk:** [`ADR-0008`](../adr/ADR-0008-authority-capability-model.md)'s `Role` is caller-asserted, not yet bound to a persisted delegation record — tracked as that ADR's own open question, not re-litigated here.

### Tampering (unauthorized modification)

- **Threat:** A compromised or buggy component writes authoritative state without going through Governance, or a workspace/coding-agent session modifies something outside its scope.
- **Mitigation:** No governed action reaches an executor without a current persisted `ALLOW` decision ([Governance invariants](../architecture/governance.md#invariants)); persistence of accepted state, events, and execution intent happens atomically before dependent execution ([System context — Trust and authority boundaries](../architecture/system-context.md#trust-and-authority-boundaries)); workspace filesystem/process confinement limits blast radius ([Workspaces](../architecture/workspaces.md#isolation-and-authority-boundaries)).
- **Residual risk:** Containerization alone is not proof of isolation ([`workspaces.md`](../architecture/workspaces.md#isolation-and-authority-boundaries)) — the concrete provider mechanism must be verified against these controls before a workspace becomes `Ready`. See [`workspace-isolation.md`](workspace-isolation.md) for which mechanism is selected first.

### Repudiation (denying an action happened)

- **Threat:** An action occurs with no attributable record, or an audit record can be altered after the fact.
- **Mitigation:** Every external action is attributable to an objective, workflow, actor, policy decision, and persisted intent ([System context](../architecture/system-context.md#trust-and-authority-boundaries)); Governance auditability records decision identifiers, principal/action/resource, policy/authority/autonomy versions, and outcome ([Governance — Auditability](../architecture/governance.md#auditability)); audit records are append-only.
- **Residual risk:** Retention and confidentiality rules for governance evidence are an open question in `governance.md` itself — carried forward, not resolved here.

### Information disclosure

- **Threat:** Secrets, cross-organization data, or cross-tenant workspace content leak.
- **Mitigation:** Raw credentials, secrets, and reusable tokens never enter Principal, Governance, domain command, event, or audit payloads ([Identity invariants](../architecture/identity.md#invariants)); cross-organization actions require explicit contract and separate evidence per scope ([Identity — Organization scoping](../architecture/identity.md#organization-scoping)); cache/image sharing must not expose source, secrets, logs, or artifacts across organizations ([Workspaces — Tenancy](../architecture/workspaces.md#isolation-and-authority-boundaries)).
- **Residual risk:** Phase 9 Slice 4 (Supabase RLS for multi-tenant isolation) is not yet implemented — first slice runs with zero RLS policies, safe only because a single hardcoded Organization exists. This is a known, roadmap-tracked gap, not a silent one.

### Denial of service

- **Threat:** Resource exhaustion (budget, compute, workspace lifetime) degrades or blocks legitimate work.
- **Mitigation:** Finance ResourceConstraint outcomes gate routing/eligibility ([Coding-agents — CodingAgentRouter](../architecture/coding-agents.md#codingagentrouter)); execution has explicit timeouts, attempt limits, backoff, and dead-letter routing ([Runtime — Failure semantics](../architecture/runtime.md#failure-semantics)); emergency pause is an overriding deny condition ([Governance — Policy composition](../architecture/governance.md#policy-composition)).
- **Residual risk:** Fairness/concurrency guarantees across organizations, departments, and objectives are an explicit open question in `runtime.md` — not resolved by this document.

### Elevation of privilege

- **Threat:** An agent or delegate gains authority beyond what was granted, or self-approves its own action.
- **Mitigation:** Agents can never self-authorize or grant themselves authority ([System context — Agent](../architecture/system-context.md#agent)); no self-approval, ever, for agents; delegation narrows authority and cannot amplify the delegator's ([Governance invariants](../architecture/governance.md#invariants)); the implementation agent cannot satisfy the independent-review requirement for its own Engineering result ([Coding-agents invariants](../architecture/coding-agents.md#invariants)).
- **Residual risk:** Which roles may approve each action/resource class, and whether dual control is required, is an open question in `governance.md` — see [`agent-authority.md`](agent-authority.md) for the agent-specific narrowing this document hands off to.

## Non-goals

This document does not re-specify Governance's decision pipeline, Identity's authentication flows, or Workspace isolation mechanics — it references them. It does not cover physical/hosting-provider security (Supabase, DigitalOcean, Vercel) beyond noting they are trusted infrastructure whose own security posture CompanyOS does not control, per [System context](../architecture/system-context.md#external-actors-and-systems).

## Invariants

- Every mitigation in this document must trace to an owning architecture document; this document does not invent independent security mechanisms.
- A new external actor or system added to `system-context.md` requires a corresponding row in this document's trust-boundary table before it is treated as reviewed.
- A CRITICAL or MAJOR finding against any cited mitigation (per [`AGENTS.md`](../../AGENTS.md#audit-finding-severity)) invalidates this document's coverage of that boundary until resolved.

## Open questions

- OPEN QUESTION: Who owns continuous threat-model maintenance as new external actors/departments/adapters are added — is this reviewed per-ADR, per-phase, or on a fixed cadence?
- OPEN QUESTION: What is the incident-response process once a threat here is realized (see `SECURITY.md`'s own open triage/disclosure-timeline questions)?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [`SECURITY.md`](../../SECURITY.md)
- [System context](../architecture/system-context.md)
- [Governance](../architecture/governance.md)
- [Identity](../architecture/identity.md)
- [Workspaces](../architecture/workspaces.md)
- [Coding-agents](../architecture/coding-agents.md)
- [Runtime](../architecture/runtime.md)
- [`agent-authority.md`](agent-authority.md)
- [`tool-security.md`](tool-security.md)
- [`workspace-isolation.md`](workspace-isolation.md)
