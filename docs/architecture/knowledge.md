# Organizational Knowledge Architecture

Status: APPROVED

## Responsibility

CompanyOS Knowledge makes provenance-bearing organizational claims and syntheses discoverable without confusing retrieval output, model memory, conversation, evidence, or drafts with approved organizational truth.

The canonical [Knowledge domain contract](../domain/knowledge.md) owns KnowledgeItem identity, fields, lifecycle, provenance semantics, and domain invariants. This architecture owns ingestion and curation flow, approval orchestration, retrieval ports and projections, source-impact processing, and persistence boundaries. It does not own authoritative workflow state, raw artifact production, metric definitions, policy decisions, model inference, or storage technology.

## Domain contracts

Knowledge architecture consumes the canonical [Knowledge](../domain/knowledge.md), [Artifact](../domain/artifact.md), [Evidence](../domain/evidence.md), [Event](../domain/event.md), [Metric](../domain/metric.md), and [Evaluation](../domain/evaluation.md) contracts. Their definitions, lifecycle states, fields, provenance, confidence, validity, and invariants are not restated here. Architecture coordinates their references and ports without changing their types or authority.

## Ingestion and curation

1. Capture a candidate from an artifact, evidence, event, approved document, external source, human contribution, or model-assisted synthesis.
2. Preserve source identity and content integrity; classify access, retention, and tenant scope.
3. Normalize into a new immutable `KnowledgeItem` version without overwriting its sources.
4. Detect potential duplicates and contradictions as review signals, not automatic merges.
5. Freeze the candidate version and submit `knowledge.review` or `knowledge.approve` through an Application use case with current authenticated reviewer evidence.
6. Governance verifies the human reviewer's Authority, policy, organization and knowledge scope, separation-of-duties requirements, and exact item-version/content digest.
7. Only a current Governance `ALLOW` for `knowledge.approve` permits the Kernel transition to `APPROVED`; Application coordinates atomic persistence of the KnowledgeReview, Governance-decision reference, and item transition.
8. Publish to approved retrieval projections only after that commit.
9. Re-evaluate affected knowledge when sources expire, are retracted, or are superseded.

Models may extract, summarize, relate, or propose knowledge. Agents, services, providers, models, deterministic rules, ingestion pipelines, and retrieval systems cannot approve knowledge. Retrieval-augmented generation returns source and status metadata and does not convert generated answers into stored knowledge automatically.

## Approval boundary

Knowledge approval is the governed Action `knowledge.approve` against a Resource identifying one organization, knowledge scope, KnowledgeItem ID, immutable version, and content/source digest. The requesting reviewer must be an authenticated active `HumanPrincipal` with current Authority for that knowledge class and scope. Policy may require independence from authorship, multiple reviewers, specialist credentials, or additional Approval evidence.

Governance `DENY` leaves the candidate unchanged or permits a separate rejected-review transition according to the review request. `REQUIRE_APPROVAL` pauses the approval action; it does not approve the KnowledgeItem. A stale item version, changed content or sources, expired reviewer evidence, changed Authority/policy, or mismatched organization requires a new Governance evaluation.

The Application layer coordinates Governance, Kernel legality, and atomic persistence. Knowledge storage, review UI, events, indexes, agents, and departments cannot set `APPROVED` directly. Approval of one version does not approve earlier, later, derived, translated, summarized, or conflicting versions.

Deterministic automatic approval is disabled. It may be considered only after a dedicated ADR is accepted that defines narrowly bounded knowledge classes, derivation reproducibility, source eligibility, validation, failure behavior, human accountability, rollback, audit, and governing policy. Until then, no `automatic` autonomy classification, deterministic pipeline, model evaluation, confidence threshold, or provider claim may replace human review.

## Retrieval contract

A query declares organization, principal, scope, permitted statuses, time context, classification clearance, purpose, and result limit. Results include item/version identity, exact status, bounded content, provenance and source links, validity, confidence semantics, and retrieval score/method.

Retrieval score measures relevance, not truth. Default organizational queries return only current `APPROVED` items. Draft-inclusive queries require an explicit purpose and label every draft. Search, embedding, graph, and model-generated summaries are projections that may be rebuilt; authoritative knowledge status is loaded from its repository.

Contradictory approved items are not silently ranked away. They are returned or quarantined with conflict evidence according to policy and trigger review.

## OSS evidence

| Reference | Observed pattern | CompanyOS use / rejection |
|---|---|---|
| [LangGraph.js checkpoint stores](https://github.com/langchain-ai/langgraphjs/tree/a86f813954e010fbf30711c37baa5c53444613d5/libs/checkpoint-postgres/src/store) and [checkpoint contract](https://github.com/langchain-ai/langgraphjs/blob/a86f813954e010fbf30711c37baa5c53444613d5/libs/checkpoint/src/base.ts) | Long-term stores and execution checkpoints are separate abstractions with namespaces, metadata, and backend adapters | Borrow separation of resumable execution state from retrievable memory; reject stored agent memory as approved knowledge |
| [JARVIS facts](https://github.com/vierisid/jarvis/blob/6e144520c747a6e0b8673ba9b75769d5d5f10a9c/src/vault/facts.ts), [observations](https://github.com/vierisid/jarvis/blob/6e144520c747a6e0b8673ba9b75769d5d5f10a9c/src/vault/observations.ts), and [vault schema](https://github.com/vierisid/jarvis/blob/6e144520c747a6e0b8673ba9b75769d5d5f10a9c/src/vault/schema.ts) | Raw observations are stored separately from facts; facts include source, confidence and verification time; entities and relationships aid retrieval | Borrow observation/claim separation and provenance fields; reject default confidence, mutable facts, extraction, or mere storage as approval |
| [Temporal workflow history](https://github.com/temporalio/temporal/tree/fb8894cfcc0a511b06146d291f673f715df3b51a/api/history/v1) | Execution history is optimized for deterministic recovery and audit of workflow mechanics | Reference history as provenance when relevant; reject replay history as a knowledge base |
| [Inngest event model](https://github.com/inngest/inngest/blob/1f91829a35cccf2372768fef4aa275f56fbd4843/pkg/event/event.go) | Events preserve triggering data, identity, timestamp and version for function execution | Use persisted events as possible sources; reject event payloads as curated knowledge without review |

## Architectural invariants

- Ingestion, retrieval, indexing, and storage adapters cannot change KnowledgeItem lifecycle status.
- Approval orchestration always passes through Application, Governance, Kernel, and atomic Persistence boundaries.
- Approved retrieval projections update only after the authoritative transition commits and remain rebuildable from canonical records.
- Retrieval applies organization, authority, classification, purpose, status, and validity filters before relevance ranking.
- Projection failure or staleness cannot promote draft material, erase provenance, or make a projection authoritative.
- Source expiry, retraction, supersession, or contradiction creates persisted impact-review work rather than silent projection edits.
- Departments and providers access Knowledge only through governed contracts and cannot bypass knowledge-scope authority.

## OPEN QUESTIONS

- Which Governance-authorized human roles, independence rules, and review counts apply to each knowledge class?
- OPEN QUESTION: Should narrowly defined deterministic approval ever be allowed? It remains disabled until a dedicated ADR is accepted.
- What contradiction, retraction, freshness, and periodic-review policies apply by knowledge class?
- What taxonomy and scope hierarchy are required for the first vertical slice?
- Which sources may be quoted, summarized, or retained under license, privacy, and confidentiality rules?

## Dependencies

- [Events](events.md)
- [Persistence](persistence.md)
- [Governance](governance.md)
- [Metrics](../domain/metric.md)
- [Evaluations](../domain/evaluation.md)
- [Artifact](../domain/artifact.md)
- [Evidence](../domain/evidence.md)
- [Knowledge contract](../domain/knowledge.md)
