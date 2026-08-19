# Metric Domain

Status: DRAFT

## Ownership

Monitoring & Evaluation (M&E) is the organizational owner of `MetricDefinition` and `Metric` lifecycle and quality. This document is their canonical domain contract. Departments, routers, Finance, providers, and adapters may produce observations or consume metrics; they cannot redefine metric meaning.

## MetricDefinition

A `MetricDefinition` is an immutable, versioned specification of one measurement. It contains:

- stable definition identity, version, name, purpose, and accountable M&E owner;
- subject types and population or scope to which the measurement applies;
- value type, unit, directionality, allowed range, and missing-value semantics;
- source-evidence requirements and inclusion/exclusion rules;
- computation, normalization, aggregation, and weighting rules;
- observation window, event-time treatment, lateness, and minimum sample requirements;
- baseline and target references without embedding mutable current values;
- uncertainty method, confidence-level convention, and quality flags;
- segmentation dimensions and comparison compatibility rules;
- security classification, retention, review interval, and known limitations;
- lifecycle status, effective interval, and supersession relationship.

A changed formula, unit, source requirement, population, uncertainty method, or interpretation creates a new definition version. Display-label corrections that cannot affect interpretation may be non-semantic corrections.

## Metric

A `Metric` is an immutable measurement produced under one exact `MetricDefinition` version. It contains:

- metric identity and definition identity/version;
- organization, subject identity/type, and relevant objective, workflow, execution, attempt, department, capability, or provider-profile references;
- measured value and unit;
- population, segments, observation window, and event/recorded timestamps;
- source Evidence identities and data-quality assessment;
- computation implementation/version and reproducibility reference;
- sample size, missing/excluded counts, and quality flags;
- uncertainty result expressed according to the definition;
- producer and responsible M&E evaluator identities;
- supersession or correction relationship when applicable.

A Metric reports a measurement. It does not establish causality, approve a result, authorize action, change an Objective, or select a provider.

## Confidence and uncertainty

Confidence is interpreted only through the referenced MetricDefinition's uncertainty method. A bare confidence number is invalid. Depending on the definition, uncertainty may be represented by an interval, distribution, error bound, calibration result, or an explicit statement that the sample cannot support an estimate.

Missing evidence yields a quality flag or no metric; it never defaults to success. Estimates, projections, provider-reported values, independently observed values, and invoice-confirmed values remain distinguishable through source and quality metadata.

## Provenance

Metric provenance is the complete, attributable chain from source Evidence through selection, transformation, computation, and publication. It identifies evidence versions, collection method, code/query/model/tool versions, parameters, exclusions, producer, timestamps, and integrity references sufficient to reproduce or explain the value.

Provider telemetry and self-reported results may be source Evidence but are not independently validated Metrics merely because they use the expected schema.

## Validity and compatibility

A Metric is valid only for its definition version, subject, population, window, evidence quality, and declared limitations. Validity does not imply that the value is current or suitable for another decision.

Metrics are comparable only when their definition versions declare compatibility or an explicit, reviewed normalization exists. M&E records incompatibility rather than silently combining differing formulas, environments, task classes, populations, or windows.

## Lifecycle

Proposed lifecycle for MetricDefinitions:

```text
DRAFT -> ACTIVE -> RETIRED
   \------> REJECTED
ACTIVE -> SUPERSEDED
```

Activation and supersession are governed, persisted M&E operations. Historical Metrics retain the exact definition version used. Metrics themselves are append-only; corrections create a new Metric linked to the corrected record.

## Invariants

- M&E owns MetricDefinition and Metric semantics; producers cannot redefine them locally.
- Every Metric references exactly one immutable MetricDefinition version.
- Unit, subject, window, evidence, computation provenance, and uncertainty interpretation are explicit.
- A Metric never grants authority, establishes causality by itself, or mutates its subject.
- Missing, late, excluded, estimated, and failed observations remain visible.
- Provider or evaluated-party self-report is labelled and cannot satisfy independent-observation requirements.
- Historical metric values and definitions are not overwritten.
- Comparison across incompatible definitions or contexts is rejected or explicitly normalized under a versioned method.

## OPEN QUESTIONS

- Which M&E role may activate or supersede a MetricDefinition?
- What common units and subject identifiers are required for the first vertical slice?
- Which metric classes require tamper-evident provenance or independent reproduction?
- What minimum sample and uncertainty conventions apply to provider eligibility decisions?

## Dependencies

- Future Evidence, Objective, Result, and Principal domain definitions
