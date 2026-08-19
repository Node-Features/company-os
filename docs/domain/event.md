# Event Domain

Status: DRAFT

## Definition

A `DomainEvent` is an immutable, versioned fact produced by an accepted Kernel decision. A `WorkflowEvent` is a DomainEvent whose subject is a Workflow or its organizational progress. An event records what occurred; it is not current state, a command, an agent message, or delivery metadata.

## Minimum contract

Every event contains:

- stable event identity, event type, and schema version;
- organization, subject type/identity, and resulting subject version;
- occurred-at and recorded-at timestamps;
- correlation and causation identities;
- workflow, objective, department, and Principal references when applicable;
- bounded provider-independent payload or an integrity-protected Artifact reference;
- producer boundary and security classification.

Delivery attempt, queue position, publication state, and consumer checkpoint are execution records outside the event payload.

## Invariants

- Events are appended atomically with the authoritative transition that produced them.
- Event identity and meaning never change; correction creates a compensating or superseding event.
- Consumers tolerate duplicate, delayed, and out-of-order delivery and deduplicate by event identity.
- An event cannot grant authority or prove current state without loading the referenced authoritative version.
- Agent messages, provider callbacks, logs, and telemetry become DomainEvents only through their owning Application/Kernel operation.

## OPEN QUESTIONS

- Which first-slice event types and payload schemas are required?
- Which subjects require aggregate sequence numbers in addition to versions?

## Dependencies

- [Principal](principal.md)
- Future Organization domain contract
