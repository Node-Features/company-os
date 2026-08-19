# Knowledge Domain

Status: DRAFT

## Definition

A `KnowledgeItem` is a versioned, organization-scoped claim or bounded synthesis curated for reuse with explicit provenance, scope, validity, and review status. Evidence supports a KnowledgeItem; an Artifact carries content; neither is Knowledge automatically.

## Minimum contract

A KnowledgeItem contains:

- stable item identity, immutable version, organization, type, and knowledge scope;
- bounded claim/content and canonical content/source digest;
- SourceReferences to Evidence, Events, Artifacts, approved documents, external sources, or human attestations;
- producer Principal and method/model/tool/version provenance;
- classification, purpose, audience, validity conditions, review deadline, and retention;
- lifecycle status: `DRAFT`, `IN_REVIEW`, `APPROVED`, `REJECTED`, `SUPERSEDED`, or `EXPIRED`;
- Governance decision, authenticated reviewer Principal, Authority/policy versions, and review evidence when approved;
- supersession, contradiction, retraction, and derivation links.

## Invariants

- Only a Governance-authorized human review followed by an Application/Kernel transition may set `APPROVED`.
- Deterministic automatic approval remains disabled until a dedicated ADR is accepted.
- Approval binds one immutable item version and digest; derived or edited content requires separate review.
- Draft, rejected, superseded, and expired material is structurally distinguishable from approved Knowledge.
- Source provenance is retained; retraction or contradiction triggers review rather than silent deletion.
- Retrieval relevance, model confidence, repetition, or provider assertion cannot change Knowledge status.
- Search, embeddings, graphs, summaries, and caches are disposable projections.

## OPEN QUESTIONS

- What knowledge scope taxonomy is required for the first slice?
- What freshness and contradiction policies apply to its first KnowledgeItem type?

## Dependencies

- [Artifact](artifact.md)
- [Evidence](evidence.md)
- [Event](event.md)
- [Principal](principal.md)
- Future Organization domain contract
