# Persistence Architecture

Status: DRAFT

## Responsibility

CompanyOS Persistence provides durable, concurrency-safe records for organizational truth, execution recovery, audit, artifacts, evidence, metrics, knowledge, and communication. It exposes CompanyOS-owned ports and consistency guarantees; it does not make a database, event broker, workflow engine, vector store, or provider session the domain authority.

No database technology, table design, event-store strategy, or deployment topology is selected here.

## Record classes and responsibilities

| Record class | Semantic owner | Persistence requirement | Authority boundary |
|---|---|---|---|
| Authoritative business state | Kernel/domain | Versioned writes with optimistic concurrency and atomic associated domain events | Current organizational truth |
| Workflow execution state | Runtime | Durable execution/attempt status, leases, waits, timers, idempotency, retry and cancellation data | Execution mechanics only |
| Checkpoints | Runtime through persistence port | Versioned, integrity-checked, compatible recovery material linked to workflow/execution/attempt/state versions | Cannot replace business state |
| Events | Owning domain/application/runtime boundary | Immutable append, unique identity, ordered aggregate version where applicable, publish status | Fact of occurrence, not current state |
| Artifacts | Producing capability/department | Immutable or versioned content reference, digest, media type, creator, lineage, classification, lifecycle and acceptance status | Content is not approved merely because stored |
| Evidence | Owning decision/evaluation | Immutable provenance, subject, claim/decision link, source, capture method, integrity, confidence and validity | Supports decisions; does not grant authority |
| Metrics | M&E/observability definition owner | Measurement definition/version, scope/window, source evidence, computation provenance and result | Derived analytical record |
| Knowledge | Knowledge owner | Versioned claim/synthesis, provenance graph, review status, validity and supersession | Only approved knowledge is canonical for reuse |
| Agent communication | Communication/session owner | Retention-limited messages, tool calls, provider/session IDs, attribution and redaction metadata | Non-authoritative; never a recovery prerequisite |

These classes may use different stores, but their semantic identities and cross-record references remain CompanyOS-owned.

## Required persistence ports

- **AuthoritativeStateRepository:** load a versioned aggregate; atomically compare-and-write accepted state, domain events, and execution intent.
- **ExecutionRepository:** claim and renew fenced leases; store attempts, waits, timers, retries, checkpoints, and terminal outcomes.
- **EventRepository / Outbox:** append immutable events, enforce identities and aggregate versions, and track publication without changing event meaning.
- **ArtifactRepository:** store or reference immutable content by digest and maintain versions, lineage, classification, and retention.
- **EvidenceRepository:** append attributable evidence and link it to decisions, evaluations, metrics, artifacts, and events.
- **KnowledgeRepository:** store versioned claims and syntheses, provenance, review decisions, validity, and supersession; retrieval indexes are projections.
- **CommunicationRepository:** retain bounded session records under explicit security and retention policy.

Ports express behavior, not one repository class per database table. Transaction support and failure semantics are capability requirements of adapters.

## Consistency boundaries

The minimum hard boundary is: an accepted authoritative state transition, its domain events, and any execution intent caused by that transition commit atomically. The [Application layer](application.md) coordinates this unit of work through Persistence ports after Governance and Kernel decisions; Persistence owns the atomicity and concurrency behavior, not the orchestration or domain decision. If this commit fails, no external execution or dependent workflow step begins.

External publication occurs after commit from a durable outbox-equivalent record. External effects cannot share a database transaction; they require idempotency, reconciliation, or compensation and are recorded as requested, dispatched, observed, and accepted phases rather than a single boolean.

Runtime notification also occurs after commit. A failed notification leaves committed execution intent discoverable for recovery and cannot cause the Application layer to repeat the accepted domain transition under a new identity.

Execution checkpoints may commit separately from organizational state because they recover mechanics. A checkpoint must reference the authoritative state version it assumed; resumption rejects stale, corrupt, incompatible, or cross-execution checkpoints.

## Concurrency, identity, and integrity

- Authoritative aggregates use immutable identity and monotonic version tokens; conflicting writes fail rather than merge implicitly.
- Event IDs, execution IDs, attempt IDs, artifact versions, evidence IDs, and knowledge versions are globally unambiguous within an organization.
- Idempotency keys bind the logical operation, not an individual retry attempt.
- Leases use fencing tokens so expired workers cannot commit late progress.
- Content-addressed digests protect artifact, evidence, checkpoint, and large-payload integrity where applicable.
- Cross-record references preserve organization scope and are validated before write and read.
- Time records distinguish occurrence, receipt, decision, persistence, and observation times.

## Failure and recovery

- A persistence timeout has an indeterminate outcome until reconciled by operation identity; it is not blindly retried under a new identity.
- Partial adapter failure cannot expose an accepted state without its domain events and execution intent.
- Read replicas, indexes, caches, search, and vector retrieval are projections and may be stale; legality checks load an authoritative version.
- Backup and restore must preserve identities, versions, event order, artifact integrity, provenance links, and tenant separation.
- Migration is versioned and reversible or accompanied by a tested forward-recovery plan; unsupported record versions fail explicitly.
- Retention deletion leaves required audit/supersession evidence unless policy requires complete erasure and permits it.
- Communication-store loss must not prevent workflow recovery; authoritative recovery never depends on conversation memory.

## OSS evidence

| Reference | Observed pattern | CompanyOS use / rejection |
|---|---|---|
| [Temporal persistence API](https://github.com/temporalio/temporal/tree/fb8894cfcc0a511b06146d291f673f715df3b51a/api/persistence/v1), [history tree](https://github.com/temporalio/temporal/blob/fb8894cfcc0a511b06146d291f673f715df3b51a/api/persistence/v1/history_tree.pb.go), and [mutable workflow state](https://github.com/temporalio/temporal/blob/fb8894cfcc0a511b06146d291f673f715df3b51a/api/persistence/v1/workflow_mutable_state.pb.go) | Durable history, mutable execution state, tasks, queues, and versioned persistence records are distinct but correlated | Borrow durable execution identity, history/recovery separation, and disposable workers; reject engine persistence as CompanyOS business authority |
| [LangGraph.js checkpoint base](https://github.com/langchain-ai/langgraphjs/blob/a86f813954e010fbf30711c37baa5c53444613d5/libs/checkpoint/src/base.ts) and [Postgres saver](https://github.com/langchain-ai/langgraphjs/tree/a86f813954e010fbf30711c37baa5c53444613d5/libs/checkpoint-postgres/src) | A stable saver contract supports checkpoints, metadata, pending writes, listing, and several backends | Borrow a provider-neutral checkpoint port and compatibility tests; reject graph checkpoints and stores as authoritative domain repositories |
| [Inngest execution state](https://github.com/inngest/inngest/blob/1f91829a35cccf2372768fef4aa275f56fbd4843/pkg/execution/state/state.go) and [CQRS history](https://github.com/inngest/inngest/tree/1f91829a35cccf2372768fef4aa275f56fbd4843/pkg/cqrs) | Run metadata separates workflow, version, run, event and idempotency identities; persisted state supports leases, pauses, history and retries | Borrow explicit identities, idempotency and error taxonomy; reject licensing-dependent source reuse and function-run state as organizational truth |
| [JARVIS vault schema](https://github.com/vierisid/jarvis/blob/6e144520c747a6e0b8673ba9b75769d5d5f10a9c/src/vault/schema.ts) | SQLite records conversations, observations, facts, entities, approvals, audit and content in one local vault | Borrow explicit record categories and provenance fields as evidence; reject a shared schema, free mutation, or conversation/fact rows as automatically authoritative |

## Invariants

- Persistence succeeds before execution advances.
- Only Kernel-authorized state written through the authoritative consistency boundary becomes organizational truth.
- The Application layer coordinates authoritative writes; Runtime, Daemon, agents, providers, and transport adapters cannot invoke them as shortcuts.
- Conversation, cache, search index, vector index, provider state, checkpoint, and message history are never authoritative business state.
- State, resulting domain events, and caused execution intent commit atomically.
- Every authoritative write is version-checked, attributable, tenant-scoped, auditable, and idempotent where retried.
- Checkpoints recover execution mechanics and always reference compatible workflow, execution, and authoritative-state versions.
- Artifacts, evidence, metrics, knowledge, and communication retain distinct types and lifecycles.
- Storage adapters cannot weaken Kernel legality, Governance decisions, retention policy, or security classification.
- Failed or indeterminate writes block dependent effects until reconciled.

## OPEN QUESTIONS

- What is the first authoritative aggregate and transaction boundary?
- Will the initial implementation use state-plus-outbox, event sourcing, or another adapter behind the same contract?
- Which records require append-only tamper evidence beyond ordinary database controls?
- What recovery point, recovery time, retention, residency, and erasure requirements apply per record class?
- Which artifact bodies live in object storage versus an authoritative metadata store?

## Dependencies

- [Kernel](kernel.md)
- [Runtime](runtime.md)
- [Application layer](application.md)
- [Events](events.md)
- [Knowledge](knowledge.md)
- [Governance](governance.md)
