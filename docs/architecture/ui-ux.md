# CompanyOS Product UI

Status: APPROVED (re-approved 2026-09-29, project owner `Node-Features` — supersedes the 2026-08-23
`APPROVED` version's visual direction and screen inventory)

Reopened after a design-mockup pass (an interactive canvas the project owner reviewed and iterated
on directly, not yet transcribed into `apps/web` code) found the previously-`APPROVED` cyberpunk
direction read as a developer console, not a usable interface for a company-management audience —
the project owner's own words: "the design does not seem to be usable to the normal user... it
should be a company management system, i.e. departments, employees, operations." The screen
inventory also grew substantially in the same pass, reframing the Research/Objective/M&E domain
model into a plain-language "observation → decision → action → evaluation" narrative the project
owner requested. Every section below reflects that revised direction; nothing here has shipped —
`apps/web`'s live UI is still whatever Phase 1/3 built (Workflow console, login).

## Responsibility

This document owns the **screen inventory and data-contract mapping** for CompanyOS's product UI
(`apps/web`, or its eventual replacement) — which screens exist or are needed, which already-built
backend endpoint each one is grounded in, and a small number of cross-cutting interaction rules
that every screen must follow regardless of which department it surfaces. It is deliberately
minimal: an information architecture, not a design system.

It now also owns one decided constraint — the visual *direction* (see "Visual direction" below):
a clean, professional dark interface (not cyberpunk — see the 2026-09-29 revision note above),
causal-chain/traceability graph visualization for Insights and for Department/Agent and Workflow
relationships, and a real charting library reconciled against live metrics. It does not own pixel-level design (exact
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

Reframed 2026-09-29 into the plain-language vocabulary the project owner asked for, mapped
explicitly onto the real domain types so the mapping stays honest rather than inventing parallel
concepts: **Discovery = `Finding`**, **Insight = `Recommendation`**, **Decision = `Objective`**
(specifically one proposed through the existing Finding/Recommendation/Evaluation/ResourceEvaluation
gate, Phase 4 Slice 4), **Action = the `Workflow`/execution a Decision leads to**, **Outcome =
`Result`**, **Evaluation = M&E's `Evaluation`**. No new domain type is introduced by this document;
where the UI's narrative needs something the domain doesn't yet track (e.g. an explicit
Decision→Action link), it's called out below rather than assumed.

| Screen | Backed by | Status |
|---|---|---|
| Overview (home) | Composed from `GET /v1/workflows*`, `/v1/approvals*`, `/v1/research/*`, `/v1/me/*`, `/v1/finance/*` — no single new endpoint, a client-side rollup of existing reads | Backend ready, UI missing |
| Workflow console | `POST/GET /v1/workflows*`, realtime `workflow:<id>` channel | Built (Phase 1) |
| Login | Supabase Auth session | Built (Phase 3 Slice 4) |
| **Approval inbox** — every pending `REQUIRE_APPROVAL` across Workflow cancel, Objective proposals, Knowledge approvals in one list | `POST /v1/approvals/{id}/decide`, `GET /v1/approvals` (added 2026-09-29 — defaults to `PENDING`-only; pass `status=` for resolved history) | Backend ready, UI missing |
| Knowledge library (browse `APPROVED` items) | `GET /v1/knowledge/items` (Slice 3) | Backend ready, UI missing |
| Knowledge review queue (see `DRAFT`, approve/reject) | `POST /v1/knowledge/items/capture`, `.../request-approval`, `GET /v1/knowledge/items?statuses=DRAFT&purpose=...` | Backend ready, UI missing |
| **Research** — a chronological narrative (Signal → Question → Evidence → Discovery → Insight) for one `ResearchQuestion`, not a generic timeline endpoint | `GET /v1/research/questions/{id}` (its nested Findings/Recommendations already carry timestamps) | Backend ready, UI missing — the "timeline" is a client-side reordering of one existing response, not new data |
| **Decisions** — every Objective proposed from a Finding/Recommendation/Evaluation/ResourceEvaluation, with its full trace back through the source that proposed it and forward through its Approval | `GET /v1/objectives` (added 2026-09-29 — **approved Decisions only**, see caveat below), `GET /v1/approvals?status=PENDING,REJECTED` (pending/declined Decisions), `GET /v1/objectives/{id}`, `GET /v1/approvals/{id}`, plus a walk back through the cited `SourceType`/`SourceID` | Backend ready — needs **both** endpoints combined client-side, not either alone (see caveat) |
| **Insights graph** — a single investigation's causal chain (Signal→Question→Evidence→Discovery→Insight→Decision→Action→Outcome→Evaluation) as a traceable diagram, not a generic multi-entity knowledge graph | Same reads as Research + Decisions above, walked and assembled client-side | Backend ready for one chain; no dedicated graph/relationship endpoint exists or is proposed — this screen is presentation over existing reads, see Open questions |
| M&E performance view | `GET /v1/me/subjects/{id}/performance-profile` | Backend ready, UI missing |
| Finance / budget dashboard | `GET /v1/finance/constraint-status`, `.../evaluations/{id}` | Backend ready, UI missing |
| **Activity log** — plain-language, human-readable feed of governed actions, with a technical-detail toggle per entry (raw action/decision id) | `governance_decisions` table has every decision | `BLOCKED` — no read endpoint exists yet |
| Objective / mission view | `GET /v1/objectives/{id}` | Backend ready (read-only), UI missing |
| **Objective / mission authoring** (human directly writes one, not only via the Finding/Recommendation/Evaluation proposal gate) | none | `BLOCKED` — no such use case exists; today an Objective can only be *proposed* from a Finding/Recommendation/Evaluation/ResourceEvaluation (Phase 4 Slice 4) |
| Department & Agent directory — each Agent shown with its name and AI-generated avatar | none | `BLOCKED` — `docs/domain/agent.md`/`department.md` (both `APPROVED`) are unimplemented (ROADMAP.md Phase 11) |
| "My departments" management view | none | `BLOCKED` — depends on `DepartmentMembership`, unimplemented |
| **Settings** (organization, roles, research/M&E/budget/workflow configuration, notifications, integrations, AI & automation, security & audit) | none for any of it | `BLOCKED` — none of this is persisted data yet: policy is hardcoded Go, there is no config table of any kind. A grouped IA shell can exist and honestly say so per-item; do not build working forms ahead of the backend that would back them |
| Policy administration (view/edit governed-action rules) | none | `BLOCKED` — policy is hardcoded Go (`governance/policy.go`), not persisted data (also listed under Settings above) |
| AI provider registration | none | `BLOCKED` — providers are env vars read once at boot; needs `ADR-0003`'s Router plus secrets management (`ROADMAP.md` Phase 9 Slice 3) |
| External/social publishing | none | Out of scope for this document — needs its own department doc and threat model first (see `docs/departments/education.md`, currently `NOT YET SPECIFIED`) |

**Decisions caveat, added 2026-09-29**: an `Objective` row only ever exists once a proposal is
approved (`CreateObjective` is called exclusively from `resumeProposeObjective`, the
`REQUIRE_APPROVAL` resume-on-approve path — see `internal/application/objective_propose.go`); a
rejected proposal never creates one at all (its reject branch is the generic no-op every
`CommandType` but `ApproveKnowledgeItem` gets). So `GET /v1/objectives` can only ever surface
**already-approved** Decisions — a pending or declined Decision exists only as an `Approval`
record. The Decisions screen genuinely needs both `GET /v1/objectives` (approved history) and
`GET /v1/approvals?status=PENDING,REJECTED` (everything not yet approved) combined client-side;
neither endpoint alone gives the full picture this screen's spec (`docs/ux/screen-inventory.md`)
describes.

## Visual direction

**Revised 2026-09-29, superseding the 2026-08-23 cyberpunk decision below**, at the project
owner's direction, after reviewing an interactive mockup of both directions. The cyberpunk theme
(neon glow, monospace-everywhere, raw governance vocabulary exposed by default) read as a
developer console rather than a tool the project owner could hand to someone running day-to-day
company operations. The replacement direction was grounded in a second, unrelated project of the
project owner's (`KNAUF`, a Next.js/shadcn-ui/Tailwind app) rather than invented from scratch, so
CompanyOS's UI shares a proven, already-shipped visual language instead of a one-off palette.

- **Theme: clean, professional, dark-first — not cyberpunk.** `Inter` for all interface text;
  `JetBrains Mono` reserved for identifiers, status codes, and technical detail specifically (not
  every label, the way the superseded direction used it). Restrained neutral-gray dark surfaces
  (not near-black with neon glow), one soft accent color used sparingly (not saturated neon), and
  status communicated primarily through small colored dots/badges and typography, not full-card
  color washes or glow effects. Exact palette/tokens are still implementation detail — the
  shadcn-ui/Tailwind token shape (`--background`, `--card`, `--primary`, `--muted-foreground`,
  `--border`, a shared `--radius`) is the recommended starting point since `apps/web` can adopt the
  same component library (`shadcn/ui`) `KNAUF` already uses, not just its color values.
- **Plain language by default; technical detail is opt-in, never hidden.** Every screen that
  surfaces a governed decision (Approvals, Decisions, Activity log) states the decision in a human
  sentence first ("Cancel 'Customer onboarding email draft' — this workflow is already running, so
  cancelling it needs your sign-off") with a **"View technical details"** toggle revealing the raw
  action string, requester, and timestamp. This does not weaken invariant 2 below (the real
  Governance reason must still be shown) — it changes which register it's shown in by default and
  lets a user reach the raw form on demand.
- **Insights graph, Department/Agent directory, Workflow execution — one causal/relationship graph
  system, not decorative node-link art.** Three distinct uses of the same idea:
  - **Insights graph**: a single investigation's chain (Signal→Question→Evidence→Discovery→
    Insight→Decision→Action→Outcome→Evaluation) laid out as a **vertical causal chain** — top to
    bottom, branching only where the real data branches (e.g. multiple Evidence items feeding one
    Discovery) — never a wide horizontal row of cards requiring page-level horizontal scroll.
    Selecting a node highlights its real ancestors and descendants (computed from the actual
    parent/child relationships the screen already has to fetch, not a separate graph store) and
    dims everything unrelated; a persistent side panel (not a modal) shows what the selected node
    came from and led to.
  - **Department & Agent directory**: departments and member Agents as connected nodes, requiring
    real `DepartmentMembership` edges (Phase 11) — unchanged from the superseded version except for
    the restrained-color rule above.
  - **Workflow execution**: `WorkflowExecutionTree.tsx`'s flat list becomes a vertical thread (the
    Intent as a spine, each Attempt branching off it) — unchanged in mechanism from the superseded
    version.

  All three should share one graph-rendering approach in implementation (recommended:
  `@xyflow/react`, unchanged from the superseded version) even though the Insights graph's vertical
  causal-chain layout is visually distinct from the Department/Agent view's free-form node graph.
  Node type is communicated through a small icon + uppercase type label + status badge — restrained
  and uniform per node, not a different bright color per type; only the actively-selected/highlighted
  path uses the accent color. Final library choice remains a Phase 10 implementation decision.
- **Metrics and dashboards — a real charting library, reconciled against live data.** Research/M&E/
  Finance dashboards (screen inventory above) render through one charting library, not ad hoc SVG
  or plain numbers. Recommended candidates: `visx` (fully composable, easiest to skin to the chosen
  token set) or `Nivo` (strong dark-theme defaults out of the box). Final pick is Phase 10
  implementation detail. **Reconciliation is not optional**: every chart renders only real
  DB-backed metrics from the endpoints already listed in the screen inventory (`PerformanceProfile`,
  `ResourceEvaluation`, research question status) — never mocked, illustrative, or placeholder data
  in the shipped product. This extends invariant 1 below to visualization specifically.
- This project has a `dataviz` skill available in the Claude Code environment for exactly this kind
  of work — it should be consulted when Phase 10 actually builds these dashboards, for a
  consistent, accessible color system rather than each chart improvising its own.
- **Motion has a real cost, stated honestly:** hover/highlight/fade-in effects must respect
  `prefers-reduced-motion`, and contrast needs genuine checking (4.5:1 body text, 3:1 at 24px+) —
  restrained is not automatically accessible without verifying it.

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
8. **A governed decision is stated in plain language first, with technical detail (raw action
   string, principal ID, timestamp) available behind an explicit toggle, never shown only one way.**
   Added 2026-09-29 — see "Visual direction."
9. **A screen offering "trace what this came from / led to" computes that trace from the same data
   it already fetched to render the screen — it never implies a relationship the underlying
   domain data doesn't actually establish** (e.g. the Insights graph's Decision→Action edge is only
   as real as `Objective`'s own fields make it; see the open question below).

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
- OPEN QUESTION (added 2026-09-29, important): the Insights graph's chain only has real backing
  data through **Decision** (`Objective.SourceType`/`SourceID` genuinely links a Decision back to
  the Finding/Recommendation/Evaluation that proposed it). **Decision→Action, Action→Outcome, and
  Outcome→Evaluation are not modeled anywhere today** — `Objective` has no field referencing a
  `Workflow`, no `Workflow` references an `Objective`, and no `Result`/`Evaluation` references back
  to the `Objective` it satisfies. Rendering that back half of the chain in Phase 10 without first
  deciding how (or whether) to add those links would violate invariant 9 above by implying a
  traceable relationship that doesn't exist in the domain. This needs its own design decision
  before the Insights graph screen goes beyond Decision — do not invent the linking fields here.
- OPEN QUESTION (added 2026-09-29): is a Settings information-architecture shell (grouped,
  everything explicitly marked unavailable) worth building in Phase 10 ahead of any of its
  backing config actually being persisted, purely to establish the navigation structure — or does
  that risk looking like a broken/incomplete product rather than an honest one? Not decided here.

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
