# Screen Inventory

Status: DRAFT (2026-09-29, awaiting project-owner review)

Elaborates [`docs/architecture/ui-ux.md`](../architecture/ui-ux.md)'s screen inventory table with
full per-screen detail. **That document's `Status` column (built / backend-ready / `BLOCKED`) is
authoritative** — this document does not repeat backend-readiness verdicts, only the product-level
spec assuming each screen is built. Components referenced by name are defined in
[`component-architecture.md`](component-architecture.md).

## Overview

- **Purpose**: single-glance summary of the whole company — what's active, what needs a decision,
  what was learned, what's the current standing.
- **User**: Owner/Operator (today, the only persona).
- **Entry point**: default post-login landing screen.
- **Primary actions**: Approve/Decline a previewed pending decision; navigate to any module via its
  preview card's "→" link.
- **Secondary actions**: none (Overview is read-oriented by design — it previews, other screens act).
- **Data displayed**: active-workflow count, pending-decision count, M&E average success rate,
  budget utilization %, latest Discovery, recent achievements (derived — see `data-ui-mapping.md`),
  recent Workflows, Department summaries, a "needs your decision" preview.
- **Data sources**: composed client-side from `GET /v1/workflows*`, `/v1/approvals` (added
  2026-09-29), `/v1/research/questions/{id}`, `/v1/me/subjects/{id}/performance-profile`,
  `/v1/finance/constraint-status`.
- **Related screens**: Research, Decisions (Approvals until Decisions ships), Workflows, Departments.
- **Required components**: StatCard, PanelCard, StatusBadge, AchievementRow, ApprovalPreviewRow.
- **Empty state**: a fresh organization with no Workflows/Research yet — each panel shows its own
  empty message ("No workflows yet — create one to get started"), not a single full-page empty
  state, since some panels may have data while others don't.
- **Loading state**: skeleton StatCards + skeleton panel rows; do not block the whole page on the
  slowest of the composed requests — render each panel independently as its data arrives.
- **Error state**: a failed panel shows an inline retry, not a full-page error — one failed
  composed request must not blank the rest of Overview.
- **Permission requirements**: none today (single persona); future: every previewed item should
  already be scoped to what the viewer's role can see (no filtering client-side after the fact).

## Workflow console

- **Purpose**: create, start, monitor, and cancel one Workflow; watch its execution thread live.
- **User**: Owner/Operator; future Engineering department contributors.
- **Entry point**: sidebar "Workflows" (list) → a specific Workflow; or Overview's recent-workflows
  list; or a direct `/workflows/{id}` link.
- **Primary actions**: Create Workflow, Start Workflow, Cancel Workflow.
- **Secondary actions**: none currently exposed (no retry/pause — the backend doesn't expose one).
- **Data displayed**: Workflow ID/state/created/definition, execution thread (Intent → Attempts,
  status per attempt, provider/model, duration).
- **Data sources**: `POST/GET /v1/workflows*`, realtime `workflow:<id>` channel (hint-to-refetch
  only, per `ui-ux.md` invariant 5).
- **Related screens**: Overview, Decisions (if this Workflow is the Action a Decision led to — not
  a real link yet, see `ui-ux.md`'s open question).
- **Required components**: Toolbar buttons, IdentityCard, ExecutionThread (spine + branching
  Attempt nodes), StatusBadge.
- **Empty state**: no Workflow selected — prompt to create one.
- **Loading state**: thread renders progressively as Intent/Attempts arrive; do not wait for the
  full chain before showing the Workflow's own state.
- **Error state**: a `DENIED` outcome renders the real Governance reason (`ui-ux.md` invariant 2),
  never a generic "action failed."
- **Permission requirements**: Cancel is already backend-gated to the initiating Principal
  (`CANCEL_WORKFLOW`'s ownership check) — the UI should hide/disable Cancel for a non-owner once
  real Principal identity is UI-visible, rather than let the backend's 403 be the only signal.

## Login

- **Purpose**: authenticate via Supabase Auth.
- **User**: any human Principal (no self-serve signup — accounts are provisioned out-of-band).
- **Entry point**: unauthenticated access to any route.
- **Primary actions**: Sign in.
- **Secondary actions**: none (no password reset/signup UI in scope).
- **Data displayed**: email/password form only.
- **Data sources**: Supabase Auth session.
- **Related screens**: redirects to Overview on success.
- **Required components**: form Input, Button.
- **Empty/loading/error states**: standard form validation + auth-failure message; no special case.
- **Permission requirements**: none (this screen exists to establish identity).

## Research

- **Purpose**: show one investigation's full chronological narrative — how a Signal became a
  Discovery and, if it did, an Insight.
- **User**: Owner/Operator; future Research department contributors.
- **Entry point**: sidebar "Research"; Overview's "Open research →".
- **Primary actions**: none mutating on this screen (research capture/publish are Agent/Service
  actions today — Slice 1's `AUTOMATIC` autonomy; a human doesn't submit signals from this UI yet).
- **Secondary actions**: "View decision →" on a Discovery/Insight that led to a proposed Objective.
- **Data displayed**: open-question/evidence/discovery/insight counts, a chronological timeline
  (Signal → Question → Evidence → Discovery → Insight → Decision), a Discoveries→Insights list.
- **Data sources**: `GET /v1/research/questions/{id}` (nested Findings/Recommendations carry the
  timestamps the timeline reorders client-side — no new endpoint).
- **Related screens**: Decisions, Insights graph (same underlying chain, different presentation).
- **Required components**: TimelineNode, PanelCard, plain-language DiscoveryCard.
- **Empty state**: "No research yet" if the organization has no `ResearchQuestion`s.
- **Loading state**: timeline renders once the question's full nested payload arrives (single
  request — no progressive loading needed for one question's worth of data).
- **Error state**: standard retry panel.
- **Permission requirements**: future — scoped to a Research department member's own department.

## Decisions

- **Purpose**: list every Objective proposed from a Finding/Recommendation/Evaluation/
  ResourceEvaluation, with full trace and resolution status.
- **User**: Owner/Operator (today the only Approval decider — `ApproverPrincipal` fixture).
- **Entry point**: sidebar "Decisions"; Research's "View decision →"; Overview's decision preview.
- **Primary actions**: none on the list itself; a *pending* Decision's Approve/Decline lives on the
  Approvals screen (`ui-ux.md`: Decisions is a list+trace view, Approvals is where the actual
  `POST /v1/approvals/{id}/decide` happens) — **product decision**: do not duplicate the
  approve/decline action here; link to Approvals instead, to avoid two places that can resolve the
  same Approval.
- **Secondary actions**: "View trace" expand/collapse per Decision.
- **Data displayed**: title, status (`PENDING`/`APPROVED`/`REJECTED`), source type, decider, time,
  and — expanded — the full causal trace back to the originating source and the resulting Approval.
- **Data sources**: `GET /v1/objectives` (approved Decisions, added 2026-09-29) combined with
  `GET /v1/approvals?status=PENDING,REJECTED` (pending/declined — an `Objective` row never exists
  for either, see `ui-ux.md`'s Decisions caveat), plus `GET /v1/objectives/{id}`/
  `/v1/approvals/{id}` for single-item detail.
- **Related screens**: Research (source), Approvals (resolution), Insights graph (same trace,
  visualized as a chain instead of a list).
- **Required components**: DecisionCard, TraceStep, StatusBadge.
- **Empty state**: "No decisions yet."
- **Loading/error states**: standard.
- **Permission requirements**: future — who may *decide* (not just view) should reuse
  `approval.md`'s "only a human, not the requester, may approve" invariant; today enforced
  backend-side by the `ApproverPrincipal` fixture, not yet a real per-user check.

## Insights graph

- **Purpose**: trace one investigation's full causal chain visually — Signal through Evaluation —
  answering "where did this come from, what did it cause."
- **User**: Owner/Operator.
- **Entry point**: sidebar "Insights graph."
- **Primary actions**: select a node (highlights ancestors/descendants, dims the rest, opens the
  detail panel).
- **Secondary actions**: toggle a type filter chip; Reset (clear selection); Fit (present, not
  wired — no pan/zoom exists in a fixed-layout vertical chain).
- **Data displayed**: the same chain Research/Decisions show, laid out as a vertical causal-chain
  diagram with edge labels (`raised`, `investigated_by`, `derived_from`, …).
- **Data sources**: same as Research + Decisions, assembled client-side — no dedicated graph
  endpoint exists or is proposed.
- **Related screens**: Research, Decisions.
- **Required components**: GraphNode, GraphConnector, DetailPanel (side panel, not a modal),
  FilterChip.
- **Empty state**: "No connected insights yet — signals, evidence, decisions, and outcomes will
  appear here as CompanyOS learns from operations" (per the project owner's own brief wording).
- **Loading state**: "Building evidence graph…" (per brief) — the chain assembly needs every source
  request to resolve before the layout is stable, so this is a real full-screen loading state,
  unlike Overview's per-panel approach.
- **Error state**: explain the failed fetch, offer retry; never render a blank/broken graph.
- **Permission requirements**: same as Research.
- **Known limitation** (see `ui-ux.md`'s open question): only backed by real data through Decision —
  Decision→Action→Outcome→Evaluation is not modeled in the domain yet.

## Monitoring & Evaluation

- **Purpose**: show how well each capability is performing.
- **User**: Owner/Operator; future M&E department contributors.
- **Entry point**: sidebar "Monitoring & Evaluation."
- **Primary actions**: none (evaluations run automatically — `me.evaluation.run`, `AUTOMATIC`
  autonomy; no human trigger in this UI).
- **Secondary actions**: none yet (drill into one subject's evaluation history — not built).
- **Data displayed**: per-capability KPI cards — classification (`PASS`/`FAIL`/`INCONCLUSIVE`),
  success rate, sample count, a small trend chart, an as-of timestamp (`ui-ux.md` invariant 6).
- **Data sources**: `GET /v1/me/subjects/{id}/performance-profile`.
- **Related screens**: Decisions (an Evaluation can be a Decision's source), Budget (both read
  `ResourceEvaluation`/`PerformanceProfile` together in Finance's cross-department read).
- **Required components**: KpiCard, TrendBars, StatusBadge.
- **Empty state**: "No evaluations yet for this capability."
- **Loading/error states**: standard.
- **Permission requirements**: future — department-scoped.

## Budget

- **Purpose**: show what's allocated, spent, and remaining, and flag departments approaching their
  cost constraint.
- **User**: Owner/Operator; future Finance department contributors.
- **Entry point**: sidebar "Budget."
- **Primary actions**: none (Budget/CostConstraint creation is not exposed in this UI pass —
  `finance.budget.create`/`finance.constraint.create` exist as governed Actions but no form is
  scoped here; see `ui-implementation-plan.md`).
- **Secondary actions**: none yet.
- **Data displayed**: total allocated/spent/remaining/utilization, a constraint-approaching alert,
  a per-department table (allocated, spent, a progress bar, a status dot+label using the real
  `ConstraintOutcome` vocabulary — `WITHIN_LIMIT`/`APPROACHING_LIMIT`/`EXCEEDED`).
- **Data sources**: `GET /v1/finance/constraint-status`, `.../evaluations/{id}`.
- **Related screens**: Monitoring & Evaluation, Decisions (a ResourceEvaluation can source one).
- **Required components**: StatCard, ProgressBar, StatusDot (plain dot+label, not a bordered badge —
  see `design-system.md`'s note on this specific fix).
- **Empty state**: "No budget set for this organization yet."
- **Loading/error states**: standard.
- **Permission requirements**: future — department-scoped; company-wide totals may need to stay
  Owner-only even once department scoping exists (`OPEN QUESTION`, not decided here).

## Approvals

- **Purpose**: act on every pending `REQUIRE_APPROVAL` action across the company in one list.
- **User**: Owner/Operator (today the only human decider).
- **Entry point**: sidebar "Approvals"; Overview's decision preview.
- **Primary actions**: Approve, Decline.
- **Secondary actions**: "View/Hide technical details" toggle per item.
- **Data displayed**: plain-language title+detail per pending item, a technical-detail block
  (action string, requester, requested-at) behind the toggle.
- **Data sources**: `GET /v1/approvals` (added 2026-09-29, defaults to `PENDING`-only), `POST
  /v1/approvals/{id}/decide` (act).
- **Related screens**: Decisions (Objective-sourced items), Activity log (once resolved).
- **Required components**: ApprovalCard, PlainLanguageDisclosure, Button (primary/ghost-danger).
- **Empty state**: "Nothing needs your approval right now."
- **Loading/error states**: standard.
- **Permission requirements**: only a human, never the requesting Principal — already a real
  backend invariant (`approval.md`), the UI should reflect it (disable Approve if the viewer is
  somehow the requester, once real identity is available client-side).

## Knowledge library

- **Purpose**: browse `APPROVED` Knowledge items.
- **User**: any user researching what the company has established as fact.
- **Entry point**: sidebar "Knowledge."
- **Primary actions**: none (read-only browse).
- **Secondary actions**: filter by source type (future — not designed here).
- **Data displayed**: item claim text, source, version, approved-at.
- **Data sources**: `GET /v1/knowledge/items` (defaults to `APPROVED`-only per `knowledge.md`'s
  retrieval contract).
- **Related screens**: Research (a Finding's auto-captured Knowledge candidate), Knowledge review
  queue.
- **Required components**: PanelCard, list row.
- **Empty state**: "No approved knowledge yet."
- **Loading/error states**: standard.
- **Permission requirements**: none documented as restrictive in `knowledge.md`.

## Knowledge review queue

- **Purpose**: review `DRAFT` Knowledge candidates and approve/reject them.
- **User**: Owner/Operator (separation-of-duties: not the item's own producer — `governance.md`'s
  Knowledge approval boundary).
- **Entry point**: sidebar "Knowledge" (a sub-tab or filtered view of the library, not designed as
  a fully separate screen here — `OPEN QUESTION` for `ui-implementation-plan.md`).
- **Primary actions**: Approve, Reject.
- **Secondary actions**: none.
- **Data displayed**: same as library, filtered to `DRAFT`, plus the required `purpose` field
  (`ui-ux.md` invariant 4 — the UI must collect and display it, never silently default one in).
- **Data sources**: `GET /v1/knowledge/items?statuses=DRAFT&purpose=...`, `POST
  /v1/knowledge/items/{id}/request-approval`, then the shared `POST /v1/approvals/{id}/decide`.
- **Related screens**: Knowledge library, Approvals.
- **Required components**: same as Approvals' ApprovalCard, specialized for Knowledge.
- **Empty state**: "Nothing awaiting review."
- **Loading/error states**: standard.
- **Permission requirements**: self-review is a real, backend-enforced `DENIED` (Phase 5 Slice 2) —
  the UI should disable review for the producer, not just let the 403 surface.

## Activity log

- **Purpose**: plain-language, grouped feed of governed actions — who did what, when, under what
  authority.
- **User**: Owner/Operator.
- **Entry point**: sidebar "Activity log."
- **Primary actions**: none (read-only).
- **Secondary actions**: technical-detail toggle per entry (raw action/decision ID).
- **Data displayed**: actor, plain-language verb+object, a `GOVERNED` tag when the action passed
  through Governance, timestamp — grouped by day.
- **Data sources**: `governance_decisions` table — **no read endpoint exists**, `BLOCKED`.
- **Related screens**: Approvals (once resolved), any screen whose action produced a
  `GovernanceDecision`.
- **Required components**: ActivityRow, DayGroupHeader, GovernedTag.
- **Empty state**: "No activity yet."
- **Loading/error states**: standard.
- **Permission requirements**: future — likely Owner-only or department-scoped; not decided.

## Objective / mission view

- **Purpose**: single Objective's read-only detail (superseded in day-to-day use by Decisions, but
  still a real, independently linkable screen per `ui-ux.md`'s inventory).
- **User**: Owner/Operator.
- **Entry point**: direct `/objectives/{id}` link, or from Decisions.
- **Primary/secondary actions**: none (read-only; resolution happens via Approvals).
- **Data displayed**: intent, source, governance decision, approval, status.
- **Data sources**: `GET /v1/objectives/{id}`.
- **Related screens**: Decisions, Research.
- **Required components**: IdentityCard, StatusBadge.
- **Empty/loading/error states**: standard.
- **Permission requirements**: none documented.

## Departments

- **Purpose**: show how the company's work is organized.
- **User**: Owner/Operator.
- **Entry point**: sidebar "Departments."
- **Primary/secondary actions**: none yet (no department-management actions exposed).
- **Data displayed**: department name, mission, active status, member avatars, member count.
- **Data sources**: today, hardcoded/mocked — no `DepartmentRegistry`/`DepartmentMembership`
  backend exists (`BLOCKED`, `ROADMAP.md` Phase 11).
- **Related screens**: People, Budget (per-department rows).
- **Required components**: DepartmentCard, AvatarStack.
- **Empty state**: "No departments configured" (once real).
- **Loading/error states**: standard.
- **Permission requirements**: future.

## People

- **Purpose**: directory of human and Agent Principals and their department membership.
- **User**: Owner/Operator.
- **Entry point**: sidebar "People."
- **Status**: `BLOCKED` — no backend exists (`ROADMAP.md` Phase 11); render the shell's honest
  "on the way" placeholder, per `ux-states.md`.

## Settings

- **Purpose**: organization, access, and per-department/system configuration.
- **User**: Owner/Operator.
- **Entry point**: sidebar "Settings."
- **Status**: `BLOCKED` — none of this is persisted data yet (policy is hardcoded Go). A grouped
  IA shell (Organization / Research & Operations / System) can exist, with every item honestly
  marked unavailable — see `ui-ux.md`'s open question on whether this is worth building ahead of
  its backend, not resolved here.
- **Required components** (for the shell only): SettingsGroupHeader, SettingsItemRow (name, short
  description, a disabled "SOON" badge).
