# Event Architecture

Status: DRAFT

## Responsibility

This architecture defines how CompanyOS events are transactionally recorded, published, routed, consumed, replayed, and evolved without transferring ownership of organizational state to a transport or agent conversation.

The canonical [Event domain contract](../domain/event.md) owns event meaning, identity, envelope fields, and domain invariants. This architecture owns delivery boundaries, ordering expectations, publication guarantees, consumption rules, and integration projections. It does not select a broker, define database tables, or make event delivery itself authoritative.

## Domain contracts

Event architecture consumes the canonical [Event](../domain/event.md), [Workflow](../domain/workflow.md), [Evidence](../domain/evidence.md), [Artifact](../domain/artifact.md), [Metric](../domain/metric.md), and [Knowledge](../domain/knowledge.md) contracts. Those documents own their definitions and invariants. Architecture preserves their distinct identities while routing references among them; it never converts conversation, delivery, checkpoint, artifact, evidence, metric, or knowledge records into events implicitly.

## Event routing

- Domain and Workflow Events conform to the canonical Event contract and enter publication only after their owning transition commits.
- Integration publication derives a stable, minimized representation for another bounded context or external integration without changing the source Event.
- Security and governance audit streams retain attributable Event references and evidence while applying their own access and retention policy.

An agent message may cause an application command. Only a successful Kernel decision coordinated by the [Application layer](application.md) and committed through Persistence may produce a domain event. Adapters translate external callbacks into untrusted inputs before validation; they do not relabel callbacks as CompanyOS facts.

## Envelope use

Publishers and consumers use the canonical [Event minimum contract](../domain/event.md#minimum-contract) without extending its meaning locally. Transport delivery identity, queue position, publication attempt, acknowledgement, and consumer checkpoint remain execution metadata outside the domain envelope. Ordering decisions use the subject version defined by the domain contract rather than timestamps or broker position.

## Publication and consumption

1. The Application use case loads authoritative state and asks the Kernel for non-mutating proposal validation.
2. Governance evaluates that exact proposal; only on current `ALLOW` does the Application layer request the final Kernel decision and coordinate atomic persistence of the accepted state transition, its domain events, and any caused execution intent.
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

## Architectural invariants

- Publication begins only from a persisted Event conforming to the canonical domain contract.
- Application coordinates atomic recording but cannot redefine Event meaning or publish an uncommitted Event.
- Delivery is treated as at-least-once, delayed, and potentially out of order; consumers persist deduplication and progress independently.
- Consumer processing cannot rewrite the source Event or roll back its committed authoritative transition.
- Transport acknowledgements, broker order, retries, and checkpoints remain delivery mechanics rather than organizational facts.
- External callbacks and messages enter through adapters as untrusted input, never directly as publishable CompanyOS Events.

## OPEN QUESTIONS

- What aggregate boundaries determine atomic state-and-event persistence?
- Which events are retained permanently, compacted, or redacted?
- What event-schema compatibility policy and registry are required for the first slice?
- Which integration events may cross organization boundaries, and under what policy?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [Event](../domain/event.md)
- [Workflow](../domain/workflow.md)
- [Evidence](../domain/evidence.md)
- [Artifact](../domain/artifact.md)
- [Metric](../domain/metric.md)
- [Knowledge](../domain/knowledge.md)
