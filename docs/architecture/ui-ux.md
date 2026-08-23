# CompanyOS Product UI

Status: APPROVED (2026-08-23, project owner `Node-Features`)

## Responsibility

This document owns the **screen inventory and data-contract mapping** for CompanyOS's product UI
(`apps/web`, or its eventual replacement) — which screens exist or are needed, which already-built
backend endpoint each one is grounded in, and a small number of cross-cutting interaction rules
that every screen must follow regardless of which department it surfaces. It is deliberately
minimal: an information architecture, not a design system.

It now also owns one decided constraint — the visual *direction* (see "Visual direction" below):
cyberpunk, graph-based visualization for Department/Agent and Workflow relationships, and a real
charting library reconciled against live metrics. It does not own pixel-level design (exact
colors, typography, spacing, branding), a specific component library or framework choice beyond
what's named below, copywriting, or accessibility standards — those stay implementation detail,
constrained by the invariants. It does not invent new backend behavior; every screen either points
at an endpoint that already exists or is explicitly marked `BLOCKED` on backend work that doesn't
exist yet, per the same "don't invent mechanism ahead of what's built" discipline every other
document in this repo follows.

## Domain contracts consumed

This document coordinates references to already-approved contracts without changing them:
[Governance](governance.md) (`Outcome`/`Reasons` vocabulary every screen must render faithfully),
[Approval](../domain/approval.md), [Knowledge](knowledge.md) (retrieval contract, draft/approved
distinction), [Departments](departments.md), [Agent](../domain/agent.md),
[Objective](../domain/objective.md), [Resource](../domain/resource.md),
[Finance department](../departments/finance.md), [Research department](../departments/research.md),
and [Monitoring & Evaluation department](../departments/monitoring-evaluation.md).

## Screen inventory

| Screen | Backed by | Status |
|---|---|---|
| Workflow console | `POST/GET /v1/workflows*`, realtime `workflow:<id>` channel | Built (Phase 1) |
| Login | Supabase Auth session | Built (Phase 3 Slice 4) |
| Approval inbox — every pending `REQUIRE_APPROVAL` across Workflow cancel, Objective proposals, Knowledge approvals in one list | `GET /v1/approvals?status=PENDING`, `POST /v1/approvals/{id}/decide` | Built (Phase 10 Slice 1, 2026-08-23) |
| Knowledge library (browse `APPROVED` items) | `GET /v1/knowledge/items` (Slice 3) | Backend ready, UI missing |
| Knowledge review queue (see `DRAFT`, approve/reject) | `POST /v1/knowledge/items/capture`, `.../request-approval`, `GET /v1/knowledge/items?statuses=DRAFT&purpose=...` | Backend ready, UI missing |
| Research report view (Signal → Question → Evidence → Finding → Recommendation) | `GET /v1/research/questions/{id}` | Backend ready, UI missing |
| M&E performance view | `GET /v1/me/subjects/{id}/performance-profile` | Backend ready, UI missing |
| Finance / budget dashboard | `GET /v1/finance/constraint-status`, `.../evaluations/{id}` | Backend ready, UI missing |
| Governance decision/audit trail | `governance_decisions` table has every decision | `BLOCKED` — no read endpoint exists yet |
| Objective / mission view | `GET /v1/objectives/{id}` | Backend ready (read-only), UI missing |
| **Objective / mission authoring** (human directly writes one, not only via the Finding/Recommendation/Evaluation proposal gate) | none | `BLOCKED` — no such use case exists; today an Objective can only be *proposed* from a Finding/Recommendation/Evaluation/ResourceEvaluation (Phase 4 Slice 4) |
| Department & Agent directory — each Agent shown with its name and AI-generated avatar | none | `BLOCKED` — `docs/domain/agent.md`/`department.md` (both `APPROVED`) are unimplemented (ROADMAP.md Phase 11) |
| "My departments" management view | none | `BLOCKED` — depends on `DepartmentMembership`, unimplemented |
| Policy administration (view/edit governed-action rules) | none | `BLOCKED` — policy is hardcoded Go (`governance/policy.go`), not persisted data |
| AI provider registration | none | `BLOCKED` — providers are env vars read once at boot; needs `ADR-0003`'s Router plus secrets management (`ROADMAP.md` Phase 9 Slice 3) |
| External/social publishing | none | Out of scope for this document — needs its own department doc and threat model first (see `docs/departments/education.md`, currently `NOT YET SPECIFIED`) |

## Visual direction

Decided 2026-08-23, at the project owner's direction. Extends, doesn't replace, an existing
precedent — the current Workflow console already established a cyberpunk look (neon cyan glow
text, dark background with a grid backdrop, monospace/technical type). This section makes that the
explicit, intended direction for every future screen, not an accident of the first slice.

- **Theme: cyberpunk.** Dark-first, neon accent glow, monospace/technical type for identifiers and
  status vocabulary (`READY`, `DENIED`, `APPROVED`) — matching what already ships. Exact palette,
  type scale, and component styling are still implementation detail; the *direction* is not.
- **Department & Agent directory — a node-link graph, not a table.** Departments and their member
  Agents rendered as connected nodes ("threads" between a Department and each of its Agents), with
  a hover/motion state revealing detail (name, avatar, purpose, current work). Requires real
  `DepartmentMembership` edges (Phase 11) — this is a visualization of real relationship data, not
  a decorative layout. Recommended library: `@xyflow/react` (React Flow) — handles node/edge
  graphs with hover and animated-edge states out of the box, and the same library can serve the
  Workflow thread visualization below, avoiding a second graphing dependency for a closely related
  problem. Final library choice remains a Phase 10 implementation decision, not fixed here.
- **Workflow execution — a thread, not a flat list.** Replaces `WorkflowExecutionTree.tsx`'s
  current plain indented list with a branching visual: the Intent as a spine, each Attempt as a
  node along it, retries visibly branching rather than just listed in sequence. Same
  graph-rendering approach as the Department/Agent view, for one consistent visual system rather
  than two unrelated ones.
- **Metrics and dashboards — a real charting library, reconciled against live data.** Research/M&E/
  Finance dashboards (screen inventory above) render through one charting library, not ad hoc SVG
  or plain numbers. Recommended candidates: `visx` (fully composable, easiest to skin for a custom
  neon palette) or `Nivo` (strong dark-theme defaults out of the box). Final pick is Phase 10
  implementation detail. **Reconciliation is not optional**: every chart renders only real
  DB-backed metrics from the endpoints already listed in the screen inventory (`PerformanceProfile`,
  `ResourceEvaluation`, research question status) — never mocked, illustrative, or placeholder data
  in the shipped product. This extends invariant 1 below to visualization specifically.
- This project has a `dataviz` skill available in the Claude Code environment for exactly this kind
  of work — it should be consulted when Phase 10 actually builds these dashboards, for a
  consistent, accessible color system rather than each chart improvising its own.
- **Motion has a real cost, stated honestly:** hover/animated-edge effects must respect
  `prefers-reduced-motion`, and a neon-on-dark palette needs genuine contrast checking, not just
  "looks cool" — "cyberpunk" is a direction, not an exemption from accessible design.

## Cross-cutting interaction invariants

These apply to every screen above, regardless of department:

1. **No UI writes directly to persistence.** Every mutating action goes through the same governed
   HTTP endpoint the API already exposes — the UI is a client of Application/Governance, never a
   bypass of it (mirrors `governance.md`'s "no governed action reaches an executor without a
   current persisted `ALLOW` decision").
2. **A `DENIED` outcome shows the real Governance reason, not a generic error.** Silently
   swallowing why an action was denied defeats the point of having a real authorization layer.
3. **Draft and approved material are always visually distinguishable when both can appear
   together** (`knowledge.md`: draft-inclusive results must be labeled — satisfied today by the
   real `Status` field on every `KnowledgeItem`; the UI must not hide it).
4. **Any screen issuing a draft-inclusive Knowledge query collects and displays a `purpose`** —
   the UI cannot silently default one in to work around the backend's own requirement.
5. **The UI never trusts a realtime push payload as authoritative** — it's a hint to refetch, the
   same pattern `WorkflowTrigger.tsx` already established for the one realtime channel that exists.
6. **Cost/budget figures show an as-of timestamp.** `ResourceEvaluation` is computed on demand
   from summed `ResourceUsage`, not streamed live — presenting it without a timestamp implies a
   freshness guarantee the backend doesn't make.
7. **Every chart or graph renders real, reconciled backend data — never mock, illustrative, or
   placeholder values in the shipped product.** A node-link graph with no `DepartmentMembership`
   data yet shows an honest empty state, not fabricated sample nodes (see "Visual direction").

## Open questions

- OPEN QUESTION: Pixel-level design system, exact palette, and branding remain unresolved —
  genuinely out of this document's scope; only the cyberpunk *direction* is decided.
- OPEN QUESTION: Final graph library (`@xyflow/react` recommended, not fixed) and final chart
  library (`visx` or `Nivo` recommended, not fixed) — both are Phase 10 implementation decisions.
- OPEN QUESTION: Per-Principal visibility/authorization for department-scoped views (the "my
  departments" screen) cannot be correctly designed until `DepartmentMembership` exists — this
  document does not invent an interim authorization rule.
- OPEN QUESTION: Should realtime coverage extend beyond Workflow to Approval/Knowledge events
  (so the Approval inbox pushes live instead of polling)? Not decided — `event_outbox` today has
  exactly one writer (`workflow_repo.go`).
- OPEN QUESTION: Is `apps/web`'s current Next.js/React stack the long-term choice for this much
  larger surface, or does a UI this size warrant revisiting that? Not addressed here.

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [Governance](governance.md)
- [Knowledge](knowledge.md)
- [Departments](departments.md)
- [Agent](../domain/agent.md)
- [Objective](../domain/objective.md)
- [Approval](../domain/approval.md)
- [Resource](../domain/resource.md)
- [Finance department](../departments/finance.md)
- [Research department](../departments/research.md)
- [Monitoring & Evaluation department](../departments/monitoring-evaluation.md)
- [ROADMAP.md](../../ROADMAP.md) Phase 10
