# ADR-0003: Model-Independent Intelligence

Status: PROPOSED

## Context

CompanyOS departments and agents need intelligence capabilities while model providers, model versions, quality, price, latency, privacy posture, tool support, and reliability change continuously. Direct provider selection inside departments or agents would couple organizational behavior to vendors, bypass Governance and Finance, fragment evaluation evidence, and prevent consistent routing.

The repository has not yet defined who may accept ADRs. This record therefore proposes a decision and is not authoritative until accepted through the future ADR process.

## Proposed decision

Introduce a CompanyOS-owned, provider-independent Intelligence boundary:

1. Departments and agents submit an `IntelligenceCapabilityRequest` that describes outcome and constraints without naming a provider or model.
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

Before acceptance:

- the first IntelligenceCapability and request schema are specified;
- Governance eligibility and Finance budget interfaces are defined;
- ModelProfile and ModelEvaluation evidence requirements are testable;
- routing determinism, fallback, and failure semantics are reviewed;
- a vertical-slice test plan demonstrates provider substitution without department changes;
- an authorized ADR approver is identified.

## Open questions

- OPEN QUESTION: Who may accept this ADR?
- OPEN QUESTION: Which first vertical slice will prove the boundary?
- OPEN QUESTION: Which router inputs are hard constraints versus governed preferences?
- OPEN QUESTION: What evidence freshness and cold-start rules apply to new models?
