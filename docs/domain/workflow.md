# Workflow Domain

Status: APPROVED

## Definition

A `WorkflowDefinition` specifies legal organizational stages, transitions, inputs, outputs, evidence, capabilities, and governance gates. A `Workflow` is one versioned organization-scoped instance progressing under that definition. Runtime executes persisted intent but does not determine legal Workflow transitions.

## Minimum contracts

`WorkflowDefinition` contains stable identity/version, purpose, input/output contracts, states, commands, transition preconditions, required Governance evidence, produced Event types, required capabilities, terminal outcomes, and compatibility rules.

`Workflow` contains stable workflow identity, organization, definition identity/version, Objective and department references, current domain state/version, initiating Principal, correlation/causation identities, accepted inputs, current wait or terminal reason, and created/updated timestamps.

`ExecutionIntent` is the immutable, persisted request for Runtime to perform one bounded capability step. It contains intent identity, workflow/state version, capability request reference, governance-decision reference, idempotency identity, constraints, due time, and expected Result contract. It is intent—not proof of dispatch or completion.

## Transition contract

A command plus authoritative Workflow snapshot produces either stable rejection reasons or a Kernel decision containing next state, DomainEvents, and optional ExecutionIntent. Application persists these atomically before notifying Runtime. Runtime returns execution Evidence and a proposed Result through a new Application request.

## First-slice lifecycle

The first vertical slice defines exactly five authoritative Workflow states:

- `PLANNED`: the Workflow has been created with an approved Objective, an active WorkflowDefinition, resolved first-step CapabilityDefinition, and accepted inputs, but no execution is authorized;
- `READY`: `START_WORKFLOW` was accepted and exactly one ExecutionIntent for the first Capability step committed with the new Workflow version and resulting DomainEvents;
- `COMPLETED`: the successful Result for that ExecutionIntent was accepted through `ACCEPT_WORKFLOW_RESULT` and committed with the final Workflow version and resulting DomainEvents;
- `FAILED`: an unsuccessful Result for that ExecutionIntent was accepted through `REJECT_WORKFLOW_RESULT` and committed with the final Workflow version and resulting DomainEvents;
- `CANCELLED`: an authorized cancellation was accepted through `CANCEL_WORKFLOW`, before a successful or unsuccessful Result was accepted, and committed with the final Workflow version and resulting DomainEvents.

`COMPLETED`, `FAILED`, and `CANCELLED` are terminal; no further command produces a transition from them in this slice.

Runtime attempts, leases, dispatch, provider execution, checkpoints, retries, waits, and returned-but-unaccepted Results do not create Workflow states. A Workflow remains `READY` while Runtime executes its persisted intent. Runtime execution state is never inferred as authoritative Workflow state.

An `INDETERMINATE` Result ([Result domain](result.md#minimum-contract)) causes no Workflow transition in this slice: the Workflow remains at its current state pending reconciliation, and only a later definitive Result under the same ExecutionIntent and execution attempt may accept or reject it via `ACCEPT_WORKFLOW_RESULT` or `REJECT_WORKFLOW_RESULT`. A `PARTIAL` Result is treated as `FAILED` in this slice: the first-slice Workflow authorizes exactly one ExecutionIntent, so partial completion of one atomic step has no distinct organizational meaning yet. Multi-step partial semantics are future work.

## First-slice commands and legal transitions

| Command | Required prior state | Accepted transition | Minimum legality |
|---|---|---|---|
| `CREATE_WORKFLOW` | No Workflow exists for the proposed identity | absent → `PLANNED` | Organization is active; Objective is `APPROVED`; WorkflowDefinition and first CapabilityDefinition are active and compatible; identities, versions, inputs, and Governance evidence match the exact proposal. |
| `START_WORKFLOW` | `PLANNED` at the expected version | `PLANNED` → `READY` | Referenced Objective and definitions remain eligible; accepted inputs remain valid; current Governance evidence matches the proposal; exactly one first-step ExecutionIntent is produced. |
| `ACCEPT_WORKFLOW_RESULT` | `READY` at the expected version | `READY` → `COMPLETED` | Result is immutable, organization-matched, has outcome `SUCCEEDED`, not previously accepted or rejected, and bound to the exact Workflow version, ExecutionIntent, CapabilityRequest, and execution attempt; required Artifact/Evidence and capability acceptance criteria are satisfied; current Governance evidence matches the proposal. |
| `REJECT_WORKFLOW_RESULT` | `READY` at the expected version | `READY` → `FAILED` | Result is immutable, organization-matched, has outcome `FAILED`, `TIMED_OUT`, or `PARTIAL`, not previously accepted or rejected, and bound to the exact Workflow version, ExecutionIntent, CapabilityRequest, and execution attempt; current Governance evidence matches the proposal. |
| `CANCEL_WORKFLOW` | `PLANNED` or `READY` at the expected version | `PLANNED` → `CANCELLED` or `READY` → `CANCELLED` | Requested by an authorized Principal; current Governance evidence matches the exact proposal. If `READY`, cancellation of the outstanding ExecutionIntent is recorded as durable intent per the [Execution domain](execution.md#invariants); it does not assume the in-flight attempt stopped immediately. |

Any failed precondition produces stable rejection reasons and no Workflow transition, DomainEvent, or ExecutionIntent. A stale, duplicate, mismatched, or insufficiently evidenced Result cannot accept or reject the Workflow in this slice — only a `SUCCEEDED` Result may accept it via `ACCEPT_WORKFLOW_RESULT`, and only a `FAILED`, `TIMED_OUT`, or `PARTIAL` Result may reject it via `REJECT_WORKFLOW_RESULT`. An `INDETERMINATE` Result advances neither command.

The Kernel performs preliminary proposal validation and final legality validation for every command. Governance decides `ALLOW`, `DENY`, or `REQUIRE_APPROVAL` for the exact immutable proposal. Application coordinates loading, invocation order, and atomic persistence. Runtime may submit execution evidence and a Result through Application, but cannot issue an authoritative transition or persist Workflow state directly.

## Invariants

- Only Kernel rules decide legal Workflow state transitions.
- Workflow state, Runtime execution state, checkpoints, and conversation state remain distinct.
- A Workflow is bound to one definition version unless an explicit compatible migration is accepted.
- Duplicate execution or delivery cannot repeat an organizational transition.
- Waiting, approval, cancellation, failure, and terminal reasons are explicit and persisted.
- No dependent execution begins before its causing transition and ExecutionIntent commit.

## OPEN QUESTIONS

- Which definition changes are compatible with running Workflows?
- Which failure outcomes require compensation rather than retry?

## Dependencies

- [Objective](objective.md)
- [Event](event.md)
- [Evidence](evidence.md)
- [Result](result.md)
- [Principal](principal.md)
- [Capability](capability.md)
- [Organization](organization.md)
- [Execution](execution.md)
