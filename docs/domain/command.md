# Command Domain

Status: APPROVED

## Definition

The command contract is the neutral, provider-independent handoff among Application, Kernel, and Governance for a requested organizational state change. It defines immutable envelopes and their integrity relationships; it does not assign orchestration, authorization, domain-decision, execution, or persistence responsibilities.

## WorkflowCommandEnvelope

`WorkflowCommandEnvelope` is the normalized request for one Workflow command. It contains:

- envelope schema version; command, request, and idempotency identities;
- command type and organization identity;
- Workflow identity plus expected version, or an explicit absent-aggregate expectation for creation;
- referenced Objective, WorkflowDefinition, and required contract identities/versions;
- requesting Principal and authenticated-evidence reference;
- normalized inputs plus Artifact, Evidence, and ResourceConstraint references;
- declared time, correlation and causation identities, and security classification.

## First-slice command vocabulary

The first slice permits exactly these Workflow command types:

| Command | Intent | Type-specific payload |
|---|---|---|
| `CREATE_WORKFLOW` | Request creation of one Workflow in its canonical initial state | Proposed Workflow identity; Objective, WorkflowDefinition, and first CapabilityDefinition identities/versions; initiating Principal; accepted input Artifact/Evidence references; applicable constraints. |
| `START_WORKFLOW` | Request that one existing Workflow become executable | Workflow identity and expected version plus the exact referenced versions and inputs required by the canonical transition. |
| `ACCEPT_WORKFLOW_RESULT` | Request authoritative acceptance of one Runtime-returned Result | Workflow identity and expected version; immutable Result identity/version/digest; ExecutionIntent, CapabilityRequest, and execution-attempt identities; Artifact/Evidence references required for acceptance. |
| `REJECT_WORKFLOW_RESULT` | Request authoritative rejection of one Runtime-returned unsuccessful Result | Workflow identity and expected version; immutable Result identity/version/digest and its `FAILED`, `TIMED_OUT`, or `PARTIAL` outcome; ExecutionIntent, CapabilityRequest, and execution-attempt identities. |
| `CANCEL_WORKFLOW` | Request authorized cancellation of one Workflow before it completes or fails | Workflow identity and expected version; requesting Principal; cancellation reason reference. |

The canonical [Workflow lifecycle](workflow.md#first-slice-commands-and-legal-transitions) owns the prior states, legal transitions, and preconditions for these commands. A command expresses requested intent; its name never proves that the transition was accepted. Adding another command type requires its legal semantics to be defined by the owning domain first.

## GovernedCommandProposal

`GovernedCommandProposal` is the immutable subject produced by preliminary Kernel validation for Governance evaluation. It contains:

- proposal identity, schema version, and canonical proposal digest;
- the exact `WorkflowCommandEnvelope` identity and digest;
- stable Action and Resource;
- organization and expected authoritative-state version;
- referenced policy, Authority, identity, approval, and constraint versions when applicable;
- normalized arguments and trusted-context digest;
- proposed effect classification and expiry when applicable.

It contains no next state, DomainEvent, Approval outcome, or ExecutionIntent. Governance evaluates the proposal without rewriting it.

For the first slice, proposal validation maps commands to stable Actions without changing their domain meaning:

- `CREATE_WORKFLOW` → `workflow.create` on the proposed organization-scoped Workflow Resource;
- `START_WORKFLOW` → `workflow.start` on the existing Workflow Resource;
- `ACCEPT_WORKFLOW_RESULT` → `workflow.result.accept` on the existing Workflow and referenced Result Resource;
- `REJECT_WORKFLOW_RESULT` → `workflow.result.reject` on the existing Workflow and referenced Result Resource;
- `CANCEL_WORKFLOW` → `workflow.cancel` on the existing Workflow Resource.

## KernelDecisionEnvelope

`KernelDecisionEnvelope` is the immutable result of final Kernel validation. It contains:

- decision identity and schema version;
- proposal identity and digest;
- organization, aggregate identity, and prior authoritative version or explicit prior absence for creation;
- either stable rejection reasons, or the next aggregate state/version, DomainEvents, and optional ExecutionIntent;
- Governance Decision and Approval references consumed by final validation;
- declared decision time and correlation/causation identities.

Only the Kernel may produce this envelope. The envelope is a proposed atomic write, not proof that persistence or execution succeeded.

## PendingCommand

`PendingCommand` durably preserves an unexecuted command awaiting approval. It contains:

- pending-command identity, schema version, organization, and status;
- immutable command and proposal identities, digests, and governed payload reference;
- expected aggregate, policy, Authority, identity, and constraint versions;
- Governance `REQUIRE_APPROVAL` Decision and `PENDING` Approval references;
- request, idempotency, correlation, and causation identities;
- creation and expiry times plus closure reason and resulting decision reference when closed.

A PendingCommand is not aggregate state, an Approval, an ExecutionIntent, or evidence that the requested action was accepted. Application coordinates its atomic creation and closure; the contract does not persist itself.

## Invariants

- Every envelope is organization-scoped, versioned, immutable after hashing, and provider-independent.
- Every consumer verifies schema version, organization, referenced versions, digests, and correlation identities.
- Material command or governed-context change requires a new command and proposal identity.
- Governance evaluates the exact proposal digest that final Kernel validation consumes.
- `REQUIRE_APPROVAL` creates no target-domain transition or ExecutionIntent.
- A KernelDecisionEnvelope becomes authoritative only through a successful Application-coordinated atomic commit.
- Duplicate requests and resumes are reconciled by stable idempotency identity; they cannot create a second legal transition.

## Open questions

- OPEN QUESTION: Which canonical serialization and digest algorithm will the first implementation use?
- OPEN QUESTION: Which reusable rejection-reason vocabulary belongs in the first slice?

## Dependencies

- [Organization](organization.md)
- [Workflow](workflow.md)
- [Principal](principal.md)
- [Policy](policy.md)
- [Approval](approval.md)
- [Artifact](artifact.md)
- [Evidence](evidence.md)
- [Event](event.md)
- [Resource](resource.md)
