# Organizational Knowledge Architecture

Status: DRAFT

## Responsibility

CompanyOS Knowledge makes provenance-bearing organizational claims and syntheses discoverable without confusing retrieval output, model memory, conversation, evidence, or drafts with approved organizational truth.

Knowledge owns claim/synthesis lifecycle, provenance and lineage, review status, validity and supersession, retrieval contracts, access policy inputs, and links to source artifacts and evidence. It does not own authoritative workflow state, raw artifact production, metric definitions, policy decisions, model inference, or storage technology.

## Knowledge model

| Concept | Definition |
|---|---|
| `KnowledgeItem` | A versioned atomic claim or bounded synthesis with stable identity, content, scope, provenance, status, validity and supersession links |
| `SourceReference` | Immutable reference to an artifact, event, evidence record, approved document, external source, or human attestation, with digest/version when possible |
| `Provenance` | Who or what produced the item, from which sources, by what method/model/tool/version, when, and under which organization and workflow context |
| `KnowledgeReview` | Attributable human review of a specific immutable item version, linked to the Governance decision authorizing that reviewer and action |
| `KnowledgeProjection` | Search, graph, embedding, summary, cache, or index derived from stored items; reproducible and disposable |

An artifact is an output object. Evidence is an observation used to support a claim or decision. Knowledge is a curated, reusable claim or synthesis derived from sources. A metric is a versioned measurement derived under a definition. None becomes another merely by being indexed together.

## Lifecycle and status

Knowledge uses explicit lifecycle states:

- `DRAFT`: captured or generated but not reviewed; returned only when the caller permits draft material.
- `IN_REVIEW`: frozen candidate version awaiting accountable review.
- `APPROVED`: accepted for its declared scope and validity period by a Governance-authorized human reviewer acting on the exact immutable item version.
- `REJECTED`: reviewed and not accepted; retained as evidence subject to policy.
- `SUPERSEDED`: replaced by a linked later approved version.
- `EXPIRED`: no longer valid because its validity condition or review period ended.

These are knowledge-record lifecycle states, not the documentation status vocabulary. Only `APPROVED` knowledge may be presented as canonical organizational knowledge, and even then only within its declared scope and validity. Drafts remain visually and structurally distinguishable in every retrieval result.

## Required item fields

Each version records item and version identity; organization and knowledge-scope identity; claim type; normalized content or artifact reference; status; authoring principal; creating workflow/department; source references; provenance method and tool/model versions; evidence links; confidence with interpretation; effective/expiry times; security classification; KnowledgeReview identity, reviewer Principal and authentication-evidence references, Governance decision and Authority versions, rationale reference and timestamp; supersedes/superseded-by links; and created/recorded timestamps.

Confidence never substitutes for approval. A high-confidence generated statement remains `DRAFT`; an approved item may still express uncertainty.

## Ingestion and curation

1. Capture a candidate from an artifact, evidence, event, approved document, external source, human contribution, or model-assisted synthesis.
2. Preserve source identity and content integrity; classify access, retention, and tenant scope.
3. Normalize into a new immutable `KnowledgeItem` version without overwriting its sources.
4. Detect potential duplicates and contradictions as review signals, not automatic merges.
5. Freeze the candidate version and submit `knowledge.review` or `knowledge.approve` through an Application use case with current authenticated reviewer evidence.
6. Governance verifies the human reviewer's Authority, policy, organization and knowledge scope, separation-of-duties requirements, and exact item-version/content digest.
7. Only a current Governance `ALLOW` for `knowledge.approve` permits the Kernel transition to `APPROVED`; persist the KnowledgeReview, Governance-decision reference, and item transition atomically.
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

## Invariants

- Organizational knowledge has attributable provenance and source references.
- Approved knowledge is structurally distinguishable from drafts, rejected, superseded, and expired versions.
- Conversation, model output, agent memory, retrieved text, external content, and raw observations are not approved knowledge.
- Every approved item identifies the Governance-authorized human reviewer, authenticated Principal evidence, Authority/policy versions, Governance decision, and exact reviewed version/content digest.
- Knowledge approval cannot legalize a workflow transition, grant authority, or replace Governance.
- Only an Application/Kernel transition backed by a current Governance `ALLOW` for `knowledge.approve` can set `APPROVED`.
- Deterministic automatic knowledge approval is prohibited until a dedicated accepted ADR and governing policy explicitly enable narrowly defined cases.
- Source retraction, expiry, or supersession triggers impact review; prior provenance is never erased by editing a projection.
- Retrieval respects organization, security classification, purpose, status, and validity before relevance ranking.
- Vector/search indexes and generated summaries are disposable projections, not canonical records.
- Metrics and evidence retain their original identity when referenced by knowledge.
- A department may own a knowledge scope but cannot read or publish outside its authority.

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
