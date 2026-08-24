# CompanyOS Authority Model

Status: APPROVED

## Purpose

This document formalizes CompanyOS's own authority model — explicit, deterministic, testable, auditable — as it actually exists in code today, plus the three real gaps closed on 2026-08-24
(`docs/adr/ADR-0010-authority-model-formalization.md`). It does not introduce a new framework: `ADR-0008-authority-capability-model.md` already established, and this document reaffirms, that a generic `Subject`/`Role`/`Capability`/`Action`/`Resource` library is deliberately not built here — CompanyOS's existing `Principal`/`Role`/`Action`/`Resource` types plus `governance.Evaluate` already express everything a generic layer would, without the indirection.

## Three layers, not one

Governance decisions conflate three genuinely different questions unless they are kept explicitly separate:

| Layer | Question | Owned by |
|---|---|---|
| **LEGALITY** | Is this domain transition valid? | Kernel (`internal/kernel/{workflow,objective,knowledge}`) — pure functions, no I/O, no principal/role/ownership inspection. Verified 2026-08-24: no exception found anywhere in these three packages. |
| **AUTHORITY** | Is this actor allowed to request/perform it? | Governance (`internal/governance/evaluate.go`) — default-deny Role×Action policy match, plus resource-instance Constraints (below). |
| **APPROVAL** | Does another human decision have to occur? | The Approval domain (`internal/domain/{command,approval}`, `internal/adapters/persistence/supabase/pending_repo.go`'s `ResolveApproval`) — a separate lifecycle with its own compare-and-swap concurrency control, now also its own structural invariants (self-approval, human decider, expiry — see below). |

Kernel never authorizes; Governance never judges state-machine legality; the Approval domain never re-decides eligibility (that's what re-running `governance.Evaluate` at resume time is for). `kernel/workflow/proposal.go`'s `ValidateCancelProposal` states this explicitly in its own doc comment, as the concrete precedent for the general rule: "the Kernel owns legal transitions, not authorization."

## The Authority tuple

```
Authority =
    Principal
  + Organization
  + Department
  + Role
  + Capability
  + Action
  + Resource
  + Scope
  + Autonomy Class
  + Constraints
  + Evidence
```

Mapped onto real code — component by component, including the two that are honestly not yet real rather than invented:

| Component | Real today | Where |
|---|---|---|
| Principal | Yes | `internal/domain/principal.Principal` (`PrincipalID`, `OrganizationID`, `Kind`, `DisplayName`) |
| Organization | Yes | `internal/domain/organization.Organization`; every governed request carries `OrganizationID` |
| Department | **Not a durable concept** | Represented only via Action-string namespace prefixes (`research.*`, `finance.*`, `knowledge.*`), matching each department's package boundary. No `DepartmentMembership` binding exists — `docs/domain/principal.md`'s own dependency list names "Future Membership, Role, and Provider domain contracts" as not-yet-written. Not invented here. |
| Role | Caller-asserted, not verified | `internal/domain/policy.Role` (ADR-0008) — a plain string, trusted the same way `EvidencePresent` stands in for real evidence verification. `docs/domain/principal.md:82` defines a real, versioned `Authority` delegation relationship (delegator, delegate, allowed Actions/Resources/context, validity, depth, constraints, revocation) that Role would eventually resolve against — not built yet. |
| Capability | **No separate type — deliberately** | ADR-0008 already rejected this: `docs/domain/capability.md`'s `CapabilityDefinition` is a different, already-implemented concept (a provider-independent dispatch contract, e.g. "generate text"), not a permission. What "Capability" would mean in a generic authority model — what a Role may do — is fully expressed by which Action rules match that Role in `governance/policy.go`'s `matchRule`. No fifth type earns its keep here. |
| Action | Yes | A stable string (`Request.Action`, `command.ActionFor`) |
| Resource | Yes | `policy.Resource{Type, ID}` |
| Scope | **Currently == Organization only** | No sub-organization/department-level scope exists; `docs/domain/organization.md`'s own open question ("is the first deployment single-organization...") is still open. `docs/audit/backlog-p2-p4.md`'s "multi-tenant isolation is structural-only-by-absence" finding applies here directly — do not create a second Organization before real sub-scoping exists if this ever needs to mean more than the whole tenant. |
| Autonomy Class | Yes | `policy.AutonomyLevel` — `AUTOMATIC` / `APPROVAL_REQUIRED` / `HUMAN_ONLY`, composed most-restrictive-wins (`governance.moreRestrictive`) |
| Constraints | Yes, per-field | `governance.Request.ResourceOwnerPrincipalID` / `ExcludedPrincipalID` — resource-instance authority narrowing, evaluated before policy-rule matching. Concrete, named fields rather than a generic `[]Constraint` slice, matching the "no abstraction without a second real use" discipline this codebase already follows elsewhere. |
| Evidence | Partial | `Request.EvidencePresent` (a bool stand-in) and `Request.ApprovalEvidence` are real; full `AuthenticatedPrincipalEvidence` threading from the real authenticated caller into Governance is `docs/audit/gap-approval-principal-attribution.md`'s scope, not yet done — every governed use case still evaluates a `fixtures.Registry` principal, or (Objective/Knowledge/Research since Phase 4) the real resolved Principal's ID, but not yet full session evidence. |

## The decision model

```
AUTOMATIC        — proceeds; no human involvement required for this evaluation
REQUIRE_APPROVAL — otherwise eligible, blocked pending a separate human decision
HUMAN_ONLY       — proceeds; specifically because an eligible human requested it directly
DENIED           — does not proceed
```

`AUTOMATIC` and `HUMAN_ONLY` are both "proceed" dispositions (`policy.Decision.Allows()`) — kept distinct because they answer different audit questions, not because they behave differently to a caller deciding whether to dispatch. Renamed 2026-08-24 from the prior `ALLOW`/`DENY`/`REQUIRE_APPROVAL` three-value vocabulary — see `ADR-0010`'s Implementation section for exactly what changed and why historical `governance_decisions` rows were never rewritten.

Before this change, `HUMAN_ONLY` autonomy was declared in the domain model (`policy.AutonomyHumanOnly`) but was dead: no policy rule ever set it, `governance.Request` carried no principal-*kind* signal at all, and reaching that branch unconditionally denied regardless of who was actually asking. It is now real: `governance.Request.RequestingPrincipalKind` carries the requester's `principal.Kind`, and `Evaluate` checks it — a human requester under a `HUMAN_ONLY` requirement gets `DecisionHumanOnly`; anyone/anything else gets `DecisionDenied`.

## The evaluator pipeline

```
Request → Normalize → Resolve Principal → Resolve Resource → Resolve Capability
        → Resolve Policy → Evaluate Authority → Evaluate Autonomy → Evaluate Approval → Decision
```

Not a new nine-stage interface — a naming of stages already present in real code, so the mapping is auditable rather than aspirational:

| Stage | Real code |
|---|---|
| Normalize | Application assembles `command.GovernedCommandProposal` from the canonical-JSON digest helpers (`kernel/*/digest.go`) before Governance ever sees the request |
| Resolve Principal | Whatever `principal.Principal` value the caller currently trusts as the requester — `fixtures.Registry` for Workflow commands, the real resolved Principal (from `httpapi.PrincipalFromContext`) for Objective/Knowledge/Research since 2026-08-24 |
| Resolve Resource | Kernel-resolved facts (e.g. a Workflow's `InitiatingPrincipalID`, loaded from persistence, never client-asserted) |
| Resolve Capability | No separate step — see the Capability row above; folded into Resolve Policy |
| Resolve Policy | `governance/policy.go`'s `matchRule(Role, Action)` |
| Evaluate Authority | `governance.Evaluate`'s evidence/ownership/exclusion/default-deny checks |
| Evaluate Autonomy | Most-restrictive-wins composition of the matched Rule's autonomy with any resource-instance `AdditionalAutonomyRequirement` |
| Evaluate Approval | The `ApprovalEvidence` match against `AutonomyApprovalRequired`, or the real `HUMAN_ONLY` requester-kind check |
| Decision | The final `policy.GovernanceDecision`, persisted unconditionally for every outcome before any dependent execution continues |

Dispatch-time re-evaluation (`docs/architecture/governance.md` step 8) is satisfied by the caller invoking `Evaluate` again immediately before dispatch (`Runtime.execute` → `Application.AuthorizeDispatch`), not by anything inside `Evaluate` itself.

## Approval-domain structural invariants

Before 2026-08-24, three invariants `docs/domain/approval.md` and `docs/architecture/governance.md` both state existed only "by construction" — true today only because the same two fixtures (`fixtures.TriggerPrincipal()` as requester, `fixtures.ApproverPrincipal()` as decider) are used everywhere, not because anything checked them. `ResolveApproval` (`pending_repo.go`) now enforces all three for real, unconditionally, for every `CommandType` — not opt-in per feature, per the explicit "do not scatter authorization logic across individual features" requirement this work was scoped against:

1. **Self-approval prohibition** — the deciding principal may never equal the Approval's `RequestingPrincipalID` (`ports.ErrSelfApproval`).
2. **Human decider** — the deciding principal's `Kind` must be `principal.KindHuman` (`ports.ErrNonHumanDecider`).
3. **Staleness** — a `PendingCommand` whose `ExpiresAt` (new: `approvalTTL`, 24h, set at creation) has passed is rejected and transitioned to `EXPIRED` rather than resolvable forever (`ports.ErrApprovalExpired`).

All three are checked inside one `SELECT ... FOR UPDATE` transaction, so there is no window between checking and writing — a losing concurrent resolver sees the same well-defined outcome a solo caller would, and an illegitimate attempt (self-approval, non-human decider) leaves the Approval untouched (still `PENDING`, resolvable by a legitimate decider later) rather than burning it.

This is layered on top of, not a replacement for, Knowledge's own `ExcludedPrincipalID` producer-exclusion check (evaluated in `governance.Evaluate`, at *request* time) — the two protect genuinely different things: the general self-approval check stops the *requester* of a review from also deciding it; the Knowledge-specific check additionally stops the *content producer* (who may be a different principal than whoever happened to call the request endpoint) from deciding it.

## `knowledge.review.request` (formerly `knowledge.approve`)

`docs/architecture/governance.md`'s "Knowledge approval boundary" section described `knowledge.approve` as a direct, one-step decision: a human calls it, Governance checks "is this principal human," and returns `ALLOW`/`DENY` on the spot. The implementation was a two-step `REQUIRE_APPROVAL` queue: any principal could request review; a separate `/decide` call, routed generically through `ResolveApproval`, resolved it later. These are structurally different models, not a wording mismatch — this was `docs/audit/backlog-p2-p4.md`'s "governance-autonomy drift" finding, open since 2026-08-23.

Resolved by splitting what was actually two operations sharing one name:

- **Requesting review** (`RequestKnowledgeApproval`) is renamed to Action `knowledge.review.request`, `AutonomyApprovalRequired` — any principal may ask that a KnowledgeItem be reviewed, matching what was already built and tested.
- **Deciding** (`POST /v1/approvals/{approvalId}/decide`) is protected by the general, unconditional human-decider invariant above — not a per-action policy rule. This is *more* protective than a per-action `HUMAN_ONLY` classification would have been: it covers every `CommandType`'s decide act (Objective, Workflow-cancel, Knowledge), not just Knowledge's.

## Authorization matrix

The complete, current `firstSlicePolicies` table (`internal/governance/policy.go`) — every rule that exists, not only the illustrative ones. Default-deny governs everything not listed: an Action with no matching Role+Action(Prefix) row is `DENIED` unconditionally. `ResourceOwnerPrincipalID`/`ExcludedPrincipalID` (Constraints column) are evaluated before this table is even consulted and can turn what this table alone would allow into `DENIED`.

| Role | Action | Autonomy | Resulting outcome shape | Constraints | Rule ID |
|---|---|---|---|---|---|
| *(any)* | `workflow.create` / `.start` / `.result.accept` / `.result.reject` | AUTOMATIC | AUTOMATIC | none | `workflow-actions` |
| *(any)* | `workflow.cancel` | AUTOMATIC, escalated to APPROVAL_REQUIRED when the target Workflow is READY | AUTOMATIC (PLANNED) / REQUIRE_APPROVAL→AUTOMATIC on resolve (READY) | `ResourceOwnerPrincipalID` = the Workflow's initiator | `workflow-actions` + `cancelAutonomyRequirement` |
| *(any)* | `capability.intelligence.dispatch` | AUTOMATIC | AUTOMATIC | none | `capability-dispatch` |
| *(any)* | `research.signal.submit` / `.question.open` / `.evidence.record` / `.finding.publish` / `.recommendation.issue` | AUTOMATIC | AUTOMATIC | none | `research-*` |
| *(any)* | `me.metric.record` / `me.evaluation.run` | AUTOMATIC | AUTOMATIC | none | `me-*` |
| *(any)* | `finance.budget.create` / `.constraint.create` / `.usage.record` / `.evaluation.run` | AUTOMATIC | AUTOMATIC | none | `finance-*` |
| *(any)* | `objective.propose` | APPROVAL_REQUIRED, unconditional | REQUIRE_APPROVAL→AUTOMATIC on resolve | decide act: human decider, not the requester (structural, `ResolveApproval`) | `objective-propose` |
| *(any)* | `knowledge.item.capture` | AUTOMATIC | AUTOMATIC | none | `knowledge-item-capture` |
| *(any)* | `knowledge.review.request` | APPROVAL_REQUIRED, unconditional, permanent (no automatic-approval path exists) | REQUIRE_APPROVAL→AUTOMATIC on resolve | request: `ExcludedPrincipalID` = the item's content producer. decide act: human decider, not the requester (structural, `ResolveApproval`) | `knowledge-review-request` |
| `research_agent` | `research.read_market_data` / `.create_report` | AUTOMATIC | AUTOMATIC | none | `research-agent-*` |
| `research_agent` | any other Action (e.g. `finance.transfer_funds`, `customer.send_message`) | — | DENIED (default-deny, wrong role) | — | *(none)* |
| `finance_agent` | `finance.read_financial_data` / `.create_payment_request` | AUTOMATIC | AUTOMATIC | none | `finance-agent-*` |
| `finance_agent` | `finance.transfer_funds` | APPROVAL_REQUIRED | REQUIRE_APPROVAL→AUTOMATIC on resolve | decide act: human decider, not the requester | `finance-agent-transfer-funds` |
| *(unrecognized role)* | *(any Action requiring a role-scoped rule)* | — | DENIED (default-deny, unknown role fails closed) | — | *(none)* |
| *(any)* | *(no matching rule at all — e.g. `customer.send_message`)* | — | DENIED (default-deny) | — | *(none)* |
| *(any, non-human)* | *(any Action classified HUMAN_ONLY by a future rule)* | HUMAN_ONLY | DENIED | `RequestingPrincipalKind != HUMAN` | *(no current rule uses this autonomy — mechanism is real and tested, `TestEvaluate_HumanOnly_*`, awaiting its first real policy use)* |
| *(human)* | *(same, HUMAN_ONLY)* | HUMAN_ONLY | HUMAN_ONLY | `RequestingPrincipalKind == HUMAN` | *(same)* |

`research_agent`/`finance_agent` remain ADR-0008's illustrative role-to-action mapping, not a canonical department authority model — `docs/security/agent-authority.md` (ROADMAP.md Phase 2 Slice 2) is where real per-department agent authority is decided.

## Dependencies

- `docs/architecture/governance.md`, `docs/architecture/kernel.md`
- `docs/domain/principal.md`, `docs/domain/approval.md`, `docs/domain/command.md`, `docs/domain/organization.md`
- `docs/adr/ADR-0008-authority-capability-model.md`, `docs/adr/ADR-0010-authority-model-formalization.md`
- `docs/audit/findings.md`, `docs/audit/backlog-p2-p4.md`, `docs/audit/fixed-authority-model-formalization.md`
