# Gap: Knowledge Item Source-Uniqueness Race

Status: APPROVED (2026-08-24) — problem statement and remediation plan approved. Implementation still requires the project owner to explicitly select and authorize this slice before any code changes, per this repository's doc-gate convention.

Severity: P1 — data-integrity gap, small and isolated. See [`findings.md`](findings.md) §3 row 6.

## Problem

`knowledge_items` has a `PRIMARY KEY (knowledge_item_id, version)` and a plain, **non-unique** index `knowledge_items_source_idx` on `(organization_id, source_type, source_id)` ([`20260823040000_knowledge.sql`](../../supabase/migrations/20260823040000_knowledge.sql)). `KnowledgeRepository.CaptureItem` ([knowledge_repo.go:28-44](../../apps/companyd/internal/adapters/persistence/supabase/knowledge_repo.go)) is a bare INSERT protected only by the version-PK.

Compare to `objectives`, which has a real `UNIQUE (organization_id, source_type, source_id)` constraint as DB-level defense-in-depth behind its application-layer dedup check ([`20260823030000_objectives.sql`](../../supabase/migrations/20260823030000_objectives.sql), explicitly commented as such). Knowledge's application-layer dedup (`GetLatestBySource`, checked before capture) has no equivalent DB backstop — if two captures for the same source race past that check concurrently (plausible today: `PublishFinding`'s auto-capture, [`research_finding.go`](../../apps/companyd/internal/application/research_finding.go), fires on every Finding publish with no serialization against a second concurrent publish for a logically-related source), both can insert a competing v1 for the same `KnowledgeItemID`-less source, producing two independent Knowledge lineages for what should be one versioned item.

## Invariant

Restores: [`domain/knowledge.md`](../domain/knowledge.md)'s "immutable versioning" contract — one lineage per `(organization_id, source_type, source_id)` — with the same DB-level defense-in-depth pattern `objectives` already established.

## Proposed approach (plan-level only)

Add a unique constraint or partial unique index on `(organization_id, source_type, source_id)` scoped to the first/lowest version per lineage — or, more simply, a separate small lookup table keyed `(organization_id, source_type, source_id) → knowledge_item_id` that `CaptureItem` inserts into exactly once (unique-violation on the second racer), then uses to assign subsequent versions. The exact mechanism is an implementation choice; the constraint needs to prevent two different `KnowledgeItemID`s from ever claiming the same source, which a straightforward `UNIQUE (organization_id, source_type, source_id)` on `knowledge_items` cannot do directly since multiple versions of the same item legitimately share that tuple — this needs slightly more care than `objectives`' single-version case.

## Files likely to change

- A new migration under [`supabase/migrations`](../../supabase/migrations).
- [`apps/companyd/internal/adapters/persistence/supabase/knowledge_repo.go`](../../apps/companyd/internal/adapters/persistence/supabase/knowledge_repo.go) — `CaptureItem`, mapping the new unique-violation case to `ports.ErrConflict` (the existing `isUniqueViolation` helper already used elsewhere in this file).

## Tests required

**Before (regression baseline):** a concurrency integration test firing two simultaneous `CaptureKnowledgeCandidate` calls for the same fabricated source, asserting today's actual outcome — two independent `KnowledgeItemID`s both created.

**After:** the same test, asserting exactly one lineage is created and the second racer either joins the same lineage as v2 or receives a clean conflict — not a second independent item.

## Dependencies

- [`findings.md`](findings.md) §3 row 6.
- [`domain/knowledge.md`](../domain/knowledge.md)
- Phase 5 Slice 1/4 notes in [`.companyos/agent-memory/current-state.md`](../../.companyos/agent-memory/current-state.md) — established the `objectives`-style DB-constraint precedent this gap asks Knowledge to match.
