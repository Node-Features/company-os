# ADR-0003: Model-Independent Intelligence

Status: APPROVED

## Context

CompanyOS departments and agents need intelligence capabilities while model providers, model versions, quality, price, latency, privacy posture, tool support, and reliability change continuously. Direct provider selection inside departments or agents would couple organizational behavior to vendors, bypass Governance and Finance, fragment evaluation evidence, and prevent consistent routing.

This record proposes a decision and is not authoritative until accepted through the ADR process defined in [`docs/adr/README.md`](README.md#acceptance-authority-and-process), which names the project owner (`Node-Features`) as approver.

## Proposed decision

Introduce a CompanyOS-owned, provider-independent Intelligence boundary:

1. Departments and agents submit a canonical `CapabilityRequest` referencing an IntelligenceCapability specialization and describing outcome and constraints without naming a provider or model.
2. Task Analyzer produces versioned TaskComplexity and normalized requirements.
3. Governance and privacy rules determine candidate eligibility; Finance supplies budget and effective-cost constraints.
4. Intelligence Router filters and ranks eligible ModelProfiles using capability fit, quality, cost, latency, privacy, tools, historical reliability, and M&E evidence.
5. A persisted RoutingDecision selects a profile before dispatch through a replaceable ProviderAdapter.
6. Normalized execution and ModelEvaluation evidence returns to M&E, Finance, and Research.

The detailed proposed contract is defined in [Model-Independent Intelligence Architecture](../architecture/intelligence.md).

## Consequences

### Positive

- Departments and agents remain stable as providers and models change.
- Governance, privacy, quality, and budget constraints apply consistently.
- Routing decisions become explainable, reproducible, and auditable.
- Research discovery, M&E benchmarking, and Finance cost analysis feed one evidence model.
- Provider adapters can be added or removed without Kernel or department redesign.

### Costs and risks

- ModelProfile, evaluation, pricing, and reliability evidence must remain current.
- A central routing boundary adds latency, operational complexity, and a potential availability bottleneck.
- Normalization may hide provider-specific features or lowest-common-denominator behavior.
- Task analysis and scoring can introduce bias or circular dependence.
- Reproducibility requires versioned inputs, deterministic tie-breaking, and retained evidence.

## Alternatives rejected by this proposal

- **Departments select providers:** rejected because it creates vendor coupling and inconsistent governance.
- **Agents select providers dynamically:** rejected because agents cannot be the authority for policy, privacy, or spending.
- **One global default model:** rejected because capability, risk, quality, and cost requirements vary.
- **Lowest-cost or lowest-latency routing:** rejected because single metrics cannot satisfy quality, privacy, tools, or reliability requirements.
- **Provider gateway as architecture owner:** rejected because a gateway may implement adapters but cannot own CompanyOS capability semantics or eligibility.
- **Static model tiers only:** rejected because tiers become stale and lack task-specific M&E evidence.

## Acceptance criteria

- [x] **The first IntelligenceCapability and request schema are specified** — see `IntelligenceCapability` and `IntelligenceRequest` in [Core contracts](../architecture/intelligence.md#core-contracts).
- [x] **Governance eligibility and Finance budget interfaces are defined** — see routing-pipeline steps 3–4 in [Routing pipeline](../architecture/intelligence.md#routing-pipeline), consuming the canonical [`CostConstraint`](../domain/resource.md#costconstraint) and `ResourceConstraintSet` without redefining them.
- [x] **ModelProfile and ModelEvaluation evidence requirements are testable** — `ModelProfile` eligibility requires evidence freshness ([ModelProfile](../architecture/intelligence.md#modelprofile)); `ModelEvaluation` inherits the canonical [Evaluation domain](../domain/evaluation.md)'s explicit independence classification (`SELF_REPORTED` / `INTERNAL_INDEPENDENT` / `EXTERNAL_INDEPENDENT` / `DUAL_REVIEWED`), named confidence method, and validity window — falsifiable by construction, not a bare score.
- [x] **Routing determinism, fallback, and failure semantics are reviewed** — see "Failure and fallback semantics" and the reproducibility invariant in [Invariants](../architecture/intelligence.md#invariants) ("Routing is reproducible from the recorded request, profiles, evidence, constraints, router version, and tie-break rule").
- [x] **A vertical-slice test plan demonstrates provider substitution without department changes** — see "Acceptance evidence: provider-substitution test plan" below.
- [x] **An authorized ADR approver is identified** — the project owner (`Node-Features`), per [`docs/adr/README.md`](README.md#acceptance-authority-and-process).

All stated acceptance criteria are met at the contract level. What remains is the project owner's explicit review and the `Status: PROPOSED` → `Status: APPROVED` change.

## Acceptance evidence: provider-substitution test plan

This plan demonstrates the ADR's central claim — that a department can be served by a different model or provider with no department-level change — using only contracts already specified in `intelligence.md`.

1. Register two `ModelProfile` records, A and B, both eligible for the same first `IntelligenceCapability`, differing only in M&E-owned reliability/quality evidence and Finance-owned cost evidence.
2. A department issues an `IntelligenceRequest` for that capability. The request names no provider or model, per the invariant that neither a `CapabilityRequest` nor an `IntelligenceRequest` can contain one.
3. Under evidence set E1 (Profile A cheaper or better-evidenced), the Router selects Profile A. Runtime persists the resulting `RoutingDecision`, dispatches through Profile A's `ProviderAdapter`, and returns normalized execution/evaluation evidence to M&E, Finance, and Research.
4. Only Finance's cost evidence or M&E's `ModelEvaluation` evidence changes to evidence set E2 — not the department's request, code, or contract — so Profile B becomes preferred under the same routing weights.
5. The department reissues the identical `IntelligenceRequest`. The Router now selects Profile B and persists a new, distinct `RoutingDecision`.
6. Both RoutingDecisions are independently reproducible from their recorded request version, candidate profiles, evidence versions, constraint versions, router version, and tie-break rule, per the reproducibility invariant.
7. **Pass condition:** steps 3–6 complete with zero changes to the requesting department's `DepartmentDefinition`, code, or request shape — the only inputs that changed were evidence Research, M&E, and Finance own.

This plan proves substitution at the contract level. It does not select the first concrete `IntelligenceCapability`, `ModelProfile` pair, or provider adapter used to execute it — those remain open, tracked below.

## Open questions

- OPEN QUESTION: Which first vertical slice will prove the boundary?
- OPEN QUESTION: Which router inputs are hard constraints versus governed preferences?
- OPEN QUESTION: What evidence freshness and cold-start rules apply to new models?
