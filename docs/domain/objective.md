# Objective Domain

Status: DRAFT

## Definition

An `Objective` is a governed, organization-scoped commitment to pursue a defined outcome under explicit success evidence, constraints, ownership, and time bounds. A recommendation, prompt, task, workflow, or feedback result is not automatically an Objective.

## Minimum contract

An Objective contains:

- stable objective identity, organization, title, intent, and version;
- accountable owner Principal and responsible department references;
- source recommendation, decision, or parent-objective references;
- desired outcomes, exclusions, priority, and time bounds;
- success MetricDefinition and required Evidence references;
- applicable policy, approval, resource-constraint, and risk references;
- lifecycle status: `PROPOSED`, `APPROVED`, `ACTIVE`, `BLOCKED`, `COMPLETED`, `CANCELLED`, or `SUPERSEDED`;
- created, activated, terminal, and review timestamps with reasons.

Only an authorized Application/Kernel transition may approve, activate, complete, cancel, or supersede an Objective.

## Invariants

- Objective status is authoritative state, not inferred from workflow, agent, or provider activity.
- Completion requires the declared outcome evidence; task completion alone is insufficient.
- Changed outcomes, success semantics, or material constraints create a new version and renewed governance evaluation.
- Resource limits constrain pursuit but do not redefine success.
- Feedback and recommendations may propose an Objective but cannot create one directly.

## OPEN QUESTIONS

- Which Objective aggregate and transition forms the first vertical slice?
- May one Objective have multiple accountable owners?

## Dependencies

- [Metric](metric.md)
- [Evidence](evidence.md)
- [Resource](resource.md)
- [Principal](principal.md)
- [Organization](organization.md)
