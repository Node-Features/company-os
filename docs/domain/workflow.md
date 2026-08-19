# Workflow Domain

Status: DRAFT

## Definition

A `WorkflowDefinition` specifies legal organizational stages, transitions, inputs, outputs, evidence, capabilities, and governance gates. A `Workflow` is one versioned organization-scoped instance progressing under that definition. Runtime executes persisted intent but does not determine legal Workflow transitions.

## Minimum contracts

`WorkflowDefinition` contains stable identity/version, purpose, input/output contracts, states, commands, transition preconditions, required Governance evidence, produced Event types, required capabilities, terminal outcomes, and compatibility rules.

`Workflow` contains stable workflow identity, organization, definition identity/version, Objective and department references, current domain state/version, initiating Principal, correlation/causation identities, accepted inputs, current wait or terminal reason, and created/updated timestamps.

`ExecutionIntent` is the immutable, persisted request for Runtime to perform one bounded capability step. It contains intent identity, workflow/state version, capability request reference, governance-decision reference, idempotency identity, constraints, due time, and expected Result contract. It is intent—not proof of dispatch or completion.

## Transition contract

A command plus authoritative Workflow snapshot produces either stable rejection reasons or a Kernel decision containing next state, DomainEvents, and optional ExecutionIntent. Application persists these atomically before notifying Runtime. Runtime returns execution Evidence and a proposed Result through a new Application request.

## First-slice lifecycle

The first slice defines only these Workflow states:

- `PLANNED`: the Workflow exists with an approved Objective and resolved definition references, but execution is not yet eligible for dispatch;
- `READY`: the `START_WORKFLOW` transition has been accepted and exactly one ExecutionIntent for the first Capability step was committed with the new Workflow version and resulting DomainEvents.

`START_WORKFLOW` has one legal transition: `PLANNED -> READY`. It requires the expected Workflow version; an approved Objective reference; active compatible WorkflowDefinition and CapabilityDefinition references; normalized inputs; and current Governance evidence for the exact proposal. The Kernel enforces these preconditions and decides the transition. Application coordinates atomic persistence of the new Workflow version, DomainEvents, and ExecutionIntent before Runtime is notified.

No other Workflow state or transition is established by this first-slice contract.

## Invariants

- Only Kernel rules decide legal Workflow state transitions.
- Workflow state, Runtime execution state, checkpoints, and conversation state remain distinct.
- A Workflow is bound to one definition version unless an explicit compatible migration is accepted.
- Duplicate execution or delivery cannot repeat an organizational transition.
- Waiting, approval, cancellation, failure, and terminal reasons are explicit and persisted.
- No dependent execution begins before its causing transition and ExecutionIntent commit.

## OPEN QUESTIONS

- What additional states and transitions are required after the first `READY` execution intent completes?
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
