# Result Domain

Status: APPROVED

## Definition

A `Result` is an immutable, attributable report of the outcome of one bounded workflow step, capability request, or governed action. Provider success, process exit, or Artifact creation is not an accepted organizational Result until validated through an Application/Kernel operation.

## Minimum contract

A Result contains:

- stable result identity, organization, result type, and schema version;
- workflow, objective, execution intent, attempt, capability request, and idempotency references;
- producing Principal/provider and relevant adapter/model/tool versions;
- outcome: `SUCCEEDED`, `FAILED`, `CANCELLED`, `TIMED_OUT`, `PARTIAL`, or `INDETERMINATE`;
- bounded output data and Artifact references;
- supporting Evidence references and observed resource usage;
- started, observed, and reported timestamps;
- error classification, retryability claim, and reconciliation reference when applicable;
- integrity, provenance, and security classification.

Result acceptance and rejection are separate Kernel decisions. The first slice submits [`ACCEPT_WORKFLOW_RESULT`](command.md#first-slice-command-vocabulary) for a `SUCCEEDED` outcome or [`REJECT_WORKFLOW_RESULT`](command.md#first-slice-command-vocabulary) for a `FAILED`, `TIMED_OUT`, or `PARTIAL` outcome, using the legal transitions owned by the [Workflow domain](workflow.md#first-slice-commands-and-legal-transitions). An `INDETERMINATE` outcome advances neither command in this slice. Requesting more evidence or creating further intent from a Result remains future work once an owning domain defines those semantics.

## Invariants

- Results are immutable observations; corrections and reconciliation create linked records.
- Provider outcome and CompanyOS acceptance remain distinct.
- A Result cannot mutate Workflow or Objective state directly.
- `INDETERMINATE` blocks assumptions of success or safe retry until reconciled.
- Artifacts, Evidence, Metrics, and resource usage retain their own identities when referenced.
- Duplicate reports with one idempotency identity cannot create duplicate transitions.

## OPEN QUESTIONS

- Which Result types and error taxonomy are required for the first slice?
- Which Result classes require independent verification before acceptance?

## Dependencies

- [Artifact](artifact.md)
- [Evidence](evidence.md)
- [Objective](objective.md)
- [Principal](principal.md)
- [Capability](capability.md)
- [Organization](organization.md)
