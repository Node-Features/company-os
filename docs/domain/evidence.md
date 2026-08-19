# Evidence Domain

Status: DRAFT

## Definition

`Evidence` is an immutable, attributable observation used to support or challenge a claim, decision, evaluation, metric, or transition. Evidence informs reasoning but does not grant authority, establish causality by itself, or become approved Knowledge automatically.

## Minimum contract

Evidence contains:

- stable evidence identity, organization, type, and schema version;
- subject and supported or challenged claim references;
- source type/identity, producer Principal or provider, and collection method;
- observed-at, captured-at, validity interval, and freshness information;
- content or Artifact reference with digest and media type;
- provenance including tool, model, query, code, or adapter versions when applicable;
- integrity, classification, access, retention, and licensing metadata;
- quality flags covering completeness, independence, estimation, conflict, and known limitations;
- correlation, workflow, objective, and event references when applicable.

## Invariants

- Evidence is append-only; correction or retraction creates a linked record without erasing history.
- Provider self-report, human assertion, derived observation, and independently reproduced evidence remain distinguishable.
- Missing, stale, conflicting, or insufficient evidence is explicit and cannot default to success.
- A confidence value is valid only with a named method, scope, and provenance.
- Evidence crossing organization or classification boundaries is rejected unless an explicit governed contract permits it.

## OPEN QUESTIONS

- Which evidence types and quality flags are mandatory for the first slice?
- What integrity mechanism is required beyond a content digest?

## Dependencies

- [Event](event.md)
- [Artifact](artifact.md)
- [Principal](principal.md)
- [Organization](organization.md)
