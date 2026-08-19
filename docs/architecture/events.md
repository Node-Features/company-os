# Event Architecture

Status: DRAFT

## Responsibility

CompanyOS events are immutable, versioned records that something relevant occurred. Events support workflow coordination, audit, projections, metrics, and knowledge derivation without transferring ownership of organizational state to a transport or agent conversation.

This architecture owns event meaning, envelopes, identity, ordering expectations, publication guarantees, consumption rules, and the boundary between domain, workflow, integration, and communication records. It does not select a broker, define database tables, or make event delivery itself authoritative.

## Distinct records

| Concept | Meaning | Authority |
|---|---|---|
| `AgentMessage` | Communication content exchanged by a human, agent, provider, or tool within a session or channel | Non-authoritative input or evidence until validated by a domain operation |
| `WorkflowEvent` | Immutable fact emitted by CompanyOS about a workflow or execution occurrence | Authoritative evidence of the recorded occurrence, but not the current workflow state |
| `WorkflowState` | Current legal organizational workflow state at a specific version | Authoritative only when produced by Kernel rules and durably persisted |
| `Checkpoint` | Versioned recovery material sufficient to resume execution mechanics | Runtime recovery evidence; never organizational truth by itself |
| `Artifact` | Addressable output such as a report, patch, design, dataset, or deployment manifest | Content plus provenance; acceptance status determines organizational standing |
| `Evidence` | Attributed observation used to justify a decision, evaluation, transition, or claim | Supports authority but does not grant it |
| `Metric` | Versioned measurement derived from defined evidence over a scope and time window | Analytical result, not a state transition or policy decision |
| `Knowledge` | Provenance-bearing organizational claim or synthesis curated for retrieval and reuse | Draft unless explicitly reviewed or derived under an approved rule |

Therefore `AgentMessage != WorkflowEvent`, `WorkflowEvent != WorkflowState`, and conversation history is never authoritative state.

## Event classes

- **Domain event:** emitted with a persisted Kernel transition; states an organizational fact such as an objective transition or approval resolution.
- **Workflow event:** records execution lifecycle facts such as intent scheduled, attempt started, wait entered, checkpoint created, capability returned, or attempt failed.
- **Integration event:** a stable, minimized event published for another bounded context or external integration after its source fact is persisted.
- **Audit event:** append-only evidence of a security- or governance-relevant request, decision, access, or action.

An agent message may cause an application command. Only a successful Kernel decision coordinated by the [Application layer](application.md) and committed through Persistence may produce a domain event. Adapters translate external callbacks into untrusted inputs before validation; they do not relabel callbacks as CompanyOS facts.

## Required envelope

Every `WorkflowEvent` contains:

| Field | Requirement |
|---|---|
| `eventId` | Globally unique, stable event identity used for deduplication |
| `eventType` | Stable CompanyOS-owned semantic name |
| `occurredAt` | Declared occurrence timestamp |
| `recordedAt` | Persistence timestamp assigned by CompanyOS |
| `correlationId` | Identity joining one objective/request flow across components |
| `causationId` | Command or prior event that directly caused this event |
| `workflowId` | Organizational workflow identity; required for workflow events |
| `executionId` / `attemptId` | Runtime identities when applicable |
| `organizationId` | Tenant boundary |
| `departmentId` | Responsible or emitting department when applicable |
| `principal` | Authenticated human, agent, or service attribution |
| `payload` | Schema-validated, minimally sufficient event-specific facts |
| `schemaVersion` | Payload/envelope compatibility version |
| `aggregateVersion` | Resulting authoritative aggregate version when applicable |
| `classification` | Security, privacy, retention, and redaction class |
| `provenance` | Producing component, source references, and adapter/provider IDs where applicable |

Timestamps do not define processing order. Per-aggregate versions establish transition order; consumers must tolerate delayed, duplicate, and cross-stream out-of-order delivery.

## Publication and consumption

1. The Application use case loads authoritative state, obtains applicable Governance evidence, and asks the Kernel to validate the command.
2. The Application layer coordinates atomic persistence of the accepted state transition, its domain events, and any caused execution intent.
3. After commit, publish integration or Runtime notifications from persisted records using an outbox-equivalent mechanism.
4. Consumers deduplicate by event identity and record their own progress before acknowledging work.
5. Consumer failure never rolls back an already committed source fact; recovery replays the persisted event.

Events are immutable. Corrections use a new compensating or superseding event and, where needed, a legal state transition. Event payloads carry references rather than duplicating large artifacts, secrets, conversations, or knowledge bodies.

## Failure and compatibility

- Unknown event types or unsupported schema versions are quarantined, not guessed.
- Duplicate delivery is normal and must not duplicate a domain transition or external effect.
- A missing required event blocks dependent dispatch and raises reconciliation; it is not reconstructed from logs.
- Consumers declare supported versions. Breaking semantic changes create a new event type or version with an explicit migration strategy.
- Replay must not repeat non-idempotent external effects without idempotency, reconciliation, or compensation.
- Transport acknowledgements prove delivery mechanics only, not business completion.

## OSS evidence

| Reference | Observed pattern | CompanyOS use / rejection |
|---|---|---|
| [Temporal history API](https://github.com/temporalio/temporal/tree/fb8894cfcc0a511b06146d291f673f715df3b51a/api/history/v1) and [workflow mutable state](https://github.com/temporalio/temporal/blob/fb8894cfcc0a511b06146d291f673f715df3b51a/api/persistence/v1/workflow_mutable_state.pb.go) | Durable ordered history drives deterministic recovery while mutable execution state is stored separately | Borrow stable identities, replayable history, and separate current execution state; reject Temporal history as organizational truth |
| [LangGraph.js checkpoint contract](https://github.com/langchain-ai/langgraphjs/blob/a86f813954e010fbf30711c37baa5c53444613d5/libs/checkpoint/src/base.ts) | Checkpoints carry IDs, timestamps, channel values/versions, versions seen, metadata, and pending writes | Borrow versioned recovery envelopes; reject channel updates or messages as legal CompanyOS transitions |
| [Inngest event model](https://github.com/inngest/inngest/blob/1f91829a35cccf2372768fef4aa275f56fbd4843/pkg/event/event.go) and [execution state](https://github.com/inngest/inngest/blob/1f91829a35cccf2372768fef4aa275f56fbd4843/pkg/execution/state/state.go) | Events have identity, timestamp, version, data, user/meta fields; runs separately track workflow/run/event identities and state | Borrow validated envelopes, deduplication, and identity correlation; reject an incoming event as trusted or sufficient authority |
| [JARVIS event taxonomy](https://github.com/vierisid/jarvis/blob/6e144520c747a6e0b8673ba9b75769d5d5f10a9c/src/workflows/runtime/event-types.ts) and [event bus](https://github.com/vierisid/jarvis/blob/6e144520c747a6e0b8673ba9b75769d5d5f10a9c/src/workflows/runtime/event-bus.ts) | Central event names reduce publisher/subscriber drift; payload examples aid workflow composition | Borrow a governed catalog and typed payload contracts; reject in-process bus delivery or free-form observational events as durable organizational records |

## Invariants

- Agent messages, conversations, provider callbacks, logs, and telemetry are not CompanyOS events until validated and recorded through the owning boundary.
- An event records an occurrence; it is never a substitute for loading current authoritative state.
- State and its resulting domain events are persisted atomically before dependent execution or publication.
- Application orchestration may coordinate event persistence and notification but cannot define event meaning or publish an uncommitted domain event.
- Event identity, organization, correlation, causation, workflow, principal, timestamp, payload, and schema version are explicit when applicable.
- Consumers assume at-least-once, delayed, and out-of-order delivery.
- Event schemas are CompanyOS-owned and provider-independent.
- Events cannot grant authority, approve themselves, or bypass Kernel legality.
- Sensitive or large content is referenced with integrity and access metadata rather than copied into payloads.

## OPEN QUESTIONS

- What aggregate boundaries determine atomic state-and-event persistence?
- Which events are retained permanently, compacted, or redacted?
- What event-schema compatibility policy and registry are required for the first slice?
- Which integration events may cross organization boundaries, and under what policy?

## Dependencies

- [Kernel](kernel.md)
- [Runtime](runtime.md)
- [Application layer](application.md)
- [Governance](governance.md)
- [Persistence](persistence.md)
- [Knowledge](knowledge.md)
