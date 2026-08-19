# Evaluation Domain

Status: DRAFT

## Ownership

Monitoring & Evaluation (M&E) is the organizational owner of `Evaluation` and `PerformanceProfile` lifecycle, methodology, and quality. This document is their canonical domain contract. Model, coding-agent, department, workflow, and capability architectures may define typed specializations but cannot redefine evaluation confidence, provenance, validity, benchmark, or lifecycle semantics.

## Evaluation

An `Evaluation` is an immutable, attributable assessment of one or more Results, subjects, or interventions against declared questions and criteria using Evidence and [Metrics](metric.md). It contains:

- evaluation identity, type, version, status, and accountable M&E owner;
- organization and evaluated subject identities/types/versions;
- objective, result, workflow, execution, attempt, capability, environment, and comparison references when applicable;
- evaluation question, preregistered design reference, criteria, thresholds, and outcome vocabulary;
- benchmark, test suite, dataset, rubric, baseline, comparator, and evaluator references at exact versions;
- Metric and Evidence identities rather than copied measurements;
- method, sampling, segmentation, aggregation, and causal-claim scope;
- evaluator identities, independence classification, conflicts of interest, and review evidence;
- findings, outcome classification, unintended effects, limitations, and follow-up questions;
- confidence and uncertainty expressed under the referenced method;
- provenance, effective interval, evaluated-at time, expiry/review condition, and supersession links.

An Evaluation interprets evidence. It does not mutate the evaluated Result, redefine its Objective, authorize action, allocate budget, or select a provider.

## Evaluation design and benchmark semantics

An `EvaluationDesign` freezes the question, subject/task class, environment constraints, criteria, thresholds, metrics, evidence plan, sampling, benchmark assets, evaluator method, independence requirement, stopping rule, and analysis plan before results are inspected when feasible.

A `Benchmark` is a versioned evaluation instrument, not a score. It identifies its purpose, task/population scope, fixtures or dataset provenance, licenses and usage restrictions, contamination/leakage controls, execution environment, scoring method, quality thresholds, calibration, known limitations, security classification, and validity/review interval.

Changing fixtures, prompts, evaluator model, scoring code, thresholds, environment, or material sampling assumptions creates a new benchmark or design version. Post-hoc deviations remain recorded and reduce or qualify validity; they are never silently folded into the original design.

## Confidence and uncertainty

Every confidence claim identifies its method and inputs. It may describe statistical uncertainty, calibrated evaluator reliability, evidence strength, or another explicitly named construct; these meanings cannot be mixed into one unlabeled score.

Insufficient evidence produces `INCONCLUSIVE`, a bounded finding, or a failed evaluation—not an assumed pass. M&E defines the allowed outcome vocabulary per Evaluation type and records error bounds, assumptions, data quality, sample size, and limitations.

## Provenance and independence

Evaluation provenance links the exact Results, Metrics, Evidence, benchmark/design versions, environment, tools/models/code, parameters, evaluator actions, reviews, and timestamps used to reach the finding.

Independence is explicit:

- `SELF_REPORTED`: produced solely by the evaluated party or provider;
- `INTERNAL_INDEPENDENT`: produced by an M&E evaluator outside the implementation decision;
- `EXTERNAL_INDEPENDENT`: produced by a governed external evaluator;
- `DUAL_REVIEWED`: independently evaluated and separately reviewed.

The required level belongs to the EvaluationDesign and applicable policy. Self-reported evidence may contribute but cannot satisfy a stronger independence requirement.

## Validity

Validity is bounded by subject/profile version, task class, environment, benchmark/design version, policy context, evidence quality, time, and declared distribution. An Evaluation outside those bounds is historical evidence, not current eligibility evidence.

Expiry, material environment change, benchmark supersession, evidence retraction, detected leakage, or significant production drift triggers re-evaluation or marks the Evaluation invalid for affected decisions. Historical records remain immutable.

## PerformanceProfile

A `PerformanceProfile` is an M&E-owned, versioned projection over compatible Evaluations for one subject and declared operating context. It contains subject and profile identity/version; supported task/capability and environment scope; included Evaluation identities; derived Metric references; applicability and exclusion rules; reliability and quality findings; known failure modes; validity window; freshness state; and projection-method version.

It contains no provider authorization, budget permission, or routing decision. Routers consume a PerformanceProfile as evidence after Governance and resource eligibility checks. Updating the projection never rewrites its source Evaluations.

## Lifecycle

Proposed Evaluation lifecycle:

```text
PLANNED -> RUNNING -> COMPLETED
    |          |------> FAILED
    |          \------> INCONCLUSIVE
    \-----------------> CANCELLED
COMPLETED -> SUPERSEDED | EXPIRED | INVALIDATED
```

Evaluation status and workflow state are separate: the Application/Kernel boundary determines legal transitions, while M&E owns evaluation meaning. A PerformanceProfile is published only from eligible completed Evaluations under its projection rules.

## Specialization rules

A specialization must:

- identify its base `Evaluation` type and subject type;
- add only subject-specific references, dimensions, failure observations, and applicability constraints;
- reuse canonical MetricDefinition, Metric, benchmark, confidence, provenance, independence, validity, and lifecycle semantics;
- remain substitutable wherever a generic Evaluation is accepted;
- avoid embedding provider SDK types or provider-controlled authority.

`ModelEvaluation` and `CodingAgentEvaluation` are specializations governed by these rules.

## Invariants

- M&E owns Evaluation and PerformanceProfile semantics and lifecycle.
- Criteria, benchmark/design versions, evidence, metrics, evaluator, independence, uncertainty, provenance, and validity are explicit.
- Evaluated parties cannot authoritatively approve or edit their own Evaluation.
- Negative, failed, invalidated, and inconclusive outcomes remain discoverable.
- Evaluation cannot authorize action, allocate resources, select providers, or mutate the evaluated Result or Objective.
- PerformanceProfiles are reproducible projections, not independent facts or provider registries.
- Provider-specific specializations cannot weaken generic evaluation requirements.
- Dependent routing or objective decisions use persisted, valid, sufficiently fresh Evaluation/Profile versions.

## OPEN QUESTIONS

- Which M&E roles may approve an EvaluationDesign or publish a PerformanceProfile?
- What independence level is required by risk and decision class?
- Which outcome vocabulary and validity windows apply to the first evaluation slice?
- How are evaluator models calibrated and audited without circular self-evaluation?

## Dependencies

- [Metric](metric.md)
- [Evidence](evidence.md)
- [Result](result.md)
- [Objective](objective.md)
- [Principal](principal.md)
- [Resource](resource.md)
