# Responsive Design Strategy

Status: DRAFT (2026-09-29, awaiting project-owner review)

**Honest starting point**: the reviewed mockup is desktop-only (a fixed 1440×900 canvas) — nothing
about responsive behavior was designed or validated this round. This document states intent, not
built behavior. Treat every rule below as a requirement for implementation, not a description of
what exists.

## Priorities

CompanyOS is an internal operations tool; desktop is the primary and best-supported surface,
consistent with the existing Workflow console. Tablet should be usable, not necessarily optimized.
Mobile should be functional for the highest-value read/decide actions (Approvals, Overview) even if
data-dense screens (Insights graph, Budget table) are simplified rather than fully ported.

## Desktop (primary target, ≥1280px)

Full sidebar + content layout as built. No changes required beyond what's already in
[`design-system.md`](design-system.md).

## Tablet (≈768–1279px)

- **Navigation**: sidebar collapses to icon-only rail (labels hidden, tooltip on hover) — not
  built; matches the width-reduction pattern already used by `KNAUF`'s own sidebar
  (`w-16`/`md:w-56`), which this project's design system is otherwise derived from, so reusing that
  exact breakpoint behavior is the recommended default rather than inventing a new one.
- **Tables**: reduce column count (drop the least-critical column, e.g. Workflows' "Updated") before
  resorting to horizontal scroll; horizontal scroll on a table is acceptable as a last resort,
  unlike on the Insights graph (see below), since a table's own row structure stays legible under
  scroll in a way a diagram doesn't.
- **Dashboards**: 4-column stat grids become 2-column; 2-column panel splits (e.g. Overview's
  workflows+departments row) stack vertically.
- **Charts/graphs**: Insights graph keeps its vertical layout (already tablet-friendly by
  construction — the redesign's entire point was eliminating horizontal scroll) but the side
  DetailPanel should become a bottom sheet or full-width panel below the chain rather than a fixed
  300px column competing for width.
- **Forms**: not yet applicable (no forms built).
- **Side panels**: DetailPanel as above.
- **Modals**: not yet applicable.

## Mobile (<768px)

- **Navigation**: sidebar becomes a slide-over drawer (hamburger-triggered), not a persistent rail
  — persistent icon-only doesn't leave enough width for content.
- **Tables**: convert to a stacked-card list (each row becomes a small card: primary field bold,
  secondary fields as labeled lines) — never force a data table to scroll horizontally on mobile,
  per the project owner's own instruction ("do NOT attempt to squeeze the entire graph horizontally
  into the screen") applied consistently to tables, not only the graph.
- **Dashboards**: single column throughout; StatCards stack full-width.
- **Charts**: mini trend bars stay (they're already compact); ProgressBars stay (already full-width
  by design). Avoid introducing any new wide chart type without a mobile plan.
- **Graphs (Insights graph specifically)**: use a **simplified vertical trace view** — the full
  chain with branch points collapsed to a single "N related items" summary rather than showing
  parallel evidence cards side-by-side (which the desktop layout does at fixed pixel widths that
  won't fit mobile). This is the project owner's own explicit instruction from the graph-redesign
  brief, generalized to this document.
- **Forms**: not yet applicable.
- **Side panels**: DetailPanel becomes a full-screen view (not a sheet) given mobile's limited
  height as well as width.
- **Modals**: full-screen on mobile, per common practice — not yet applicable.

## Non-negotiable rule

**The application shell must never acquire a page-level horizontal scrollbar at any breakpoint** —
stated explicitly by the project owner for the Insights graph and treated here as a project-wide
rule, not a screen-specific fix. Any layout that would require one (a wide table, a wide chart)
must degrade per the rules above instead.

## Open questions

- OPEN QUESTION: no breakpoint values are fixed yet — the ranges above are descriptive, not
  literal `min-width` tokens; pick real values during implementation and record them here.
- OPEN QUESTION: whether `apps/web`'s current stack has a component library with built-in
  responsive primitives (relevant if `shadcn/ui` is adopted per `design-system.md`) — affects how
  much of this needs custom implementation vs. comes free.
