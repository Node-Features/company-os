# Component Architecture

Status: DRAFT (2026-09-29, awaiting project-owner review)

Each component below was built at least once in the reviewed mockup unless marked "specified, not
built." Reuse these rather than creating a one-off per screen — several screens in
[`screen-inventory.md`](screen-inventory.md) already reuse the same handful of primitives (`Card`,
`Badge`, `Button`) rather than each inventing their own.

## NavItem

- **Purpose**: one sidebar navigation entry.
- **Variants**: default, active, disabled (used for `BLOCKED` screens like Settings' children —
  actually rendered as a normal NavItem linking to an honest empty state, *not* a disabled variant;
  disabled is reserved for genuinely non-navigable items).
- **States**: default, active (accent background + primary text + border + glow), hover.
- **Props**: `icon` (16px stroke SVG), `label`, `badgeCount?` (e.g. Approvals' pending count), `to`.
- **Interaction**: click navigates; no keyboard shortcut defined.
- **Responsive**: not addressed — see `responsive-design.md`'s open gap on sidebar collapse.
- **Accessibility**: real `<button>`/`<a>`, but **no explicit `:focus-visible` style was built** —
  flagged in `accessibility.md` as a required fix before this ships.

## PageHeader

- **Purpose**: title + optional subtitle + right-aligned action slot, top of a screen's content.
- **Variants**: with/without subtitle, with/without right-slot actions.
- **Props**: `title`, `subtitle?`, `actions?`.
- **Responsive**: title truncates before actions wrap, not built — verify at narrow widths.

## StatCard

- **Purpose**: one labeled metric with an optional delta/context line.
- **Variants**: plain (Budget's "Allocated $20,000") vs. emphasized (Overview's glow-num style —
  larger, colored, text-shadow). **Product decision needed**: the mockup used the emphasized style
  inconsistently (Overview yes, Research/Budget's smaller stat chips no) — `OPEN QUESTION` for
  `ui-implementation-plan.md` to settle one rule (e.g. emphasized only on Overview, the company's
  single glanceable summary; plain everywhere else).
- **Props**: `label`, `value`, `context?` (text + color), `emphasized?: boolean`.
- **States**: static only — no loading/error sub-state; the parent screen handles that (see
  `ux-states.md`).

## PanelCard (generic Card)

- **Purpose**: the one general-purpose container — section wrapper, list container, detail card.
- **Variants**: static vs. `card-hover` (lift + glow on hover — use only when the card is itself
  clickable/expandable, never on a purely static container, per the left-edge-accent fix already
  applied once during the mockup review).
- **Props**: `children`, `hoverable?: boolean`.
- **Accessibility**: if hoverable and clickable, must be a real `<button>`/`<a>` wrapper, not a
  `<div onClick>` — the mockup's DC format used real buttons throughout; carry that into React.

## StatusBadge (bordered glow pill)

- **Purpose**: standalone categorical status.
- **Variants**: `running`/`completed`/`pending`/`failed`/`soon` (color-mapped — see
  `design-system.md`'s color table).
- **Props**: `kind`, `label`.
- **Do not use** where a StatusDot (below) is more appropriate — see `design-system.md`'s
  bordered-pill-vs-dot rule.

## StatusDot (dot + mono label)

- **Purpose**: compact status paired with another visual element (a ProgressBar, a graph node).
- **Props**: `color`, `label`.
- **Introduced**: 2026-09-29, replacing StatusBadge in the Budget screen's per-row status after
  project-owner feedback that the bordered pill was visually heavier than the thin bar above it.

## ProgressBar

- **Purpose**: allocation/utilization visualization (Budget's per-department bars).
- **Props**: `pct` (0–100), `color`.
- **States**: static — no animated fill-in built; `interaction-animation.md` recommends one.

## Button

- **Purpose**: the only interactive-action primitive.
- **Variants**: `primary` (filled, glow), `ghost-danger` (outline, destructive text), `toolbar`
  (outline, muted text — Insights graph's Fit/Reset).
- **States**: default, hover (brightness/filter shift), active (scale-down press feedback),
  disabled (not yet styled — `OPEN QUESTION`).
- **Accessibility**: real `<button>` everywhere in the mockup; icon-only buttons (topbar bell,
  toolbar icons) need an explicit `aria-label` — **not added yet**, flagged in `accessibility.md`.

## PlainLanguageDisclosure ("View technical details" toggle)

- **Purpose**: the pattern introduced this round for showing a governed action's technical detail
  on demand rather than always — Approvals, Decisions, Activity log all use it.
- **Props**: `expanded: boolean`, `onToggle`, `summary` (plain-language content, always shown),
  `detail` (technical content, shown when expanded).
- **Interaction**: click toggles; content below fades/rises in (`rise` animation class).
- **Accessibility**: should use `aria-expanded` on the toggle button and associate it with the
  revealed content (`aria-controls`) — not present in the mockup; add in implementation.

## TimelineNode / CausalChainNode

- **Purpose**: one step in a vertical sequence — used by Research's timeline, the Workflow
  execution thread, and the Insights graph (a more elaborate variant, see GraphNode below).
- **Variants**: simple (Research timeline: dot + type label + text) vs. rich (Insights graph: full
  card with icon, type, status badge, title, meta).
- **Props**: `kind`, `color`, `label`, `meta?`.
- **Connector**: a separate small element (vertical line, optionally dashed for a "loop" edge) —
  not part of the node itself, so branching/merging layouts (evidence fan-out) can compose it
  independently.

## GraphNode

- **Purpose**: Insights graph's node — the rich TimelineNode variant, with selection/path state.
- **Props**: `type`, `title`, `meta`, `status?`, `selected: boolean`, `inPath: boolean`,
  `filteredOut: boolean`, `onClick`.
- **States**: default, in-path (full opacity), out-of-path (dimmed, ~0.32 opacity), selected
  (primary border + glow), filtered-out (dimmed further, ~0.22 opacity — filter takes precedence
  over path state).
- **Interaction**: click toggles selection (click again to clear); computes ancestor/descendant
  highlighting from the same relationship data the screen already fetched (`ui-ux.md` invariant 9
  — never a separate, potentially-inconsistent graph store).
- **Accessibility**: needs keyboard activation (Enter/Space) and a visible focus ring — **not
  built**, required per `accessibility.md`.

## DetailPanel (side panel, not a modal)

- **Purpose**: shows the selected GraphNode's (or, in principle, any selected entity's) full
  detail — type, title, status, "Derived from," "Led to."
- **Props**: `selection: Node | null`.
- **States**: empty ("Select a node to see…") vs. populated.
- **Placement rule**: persistent side panel, not an overlay/modal — established by the Insights
  graph and recommended as the default pattern for "view detail without leaving the list/graph"
  interactions elsewhere (e.g. a future Decisions detail view).

## FilterChip

- **Purpose**: toggleable type filter, doubling as a legend (Insights graph's 10 type chips).
- **Props**: `label`, `dot color`, `count`, `active: boolean`, `onToggle`.
- **Pattern note**: combining filter + legend into one control was a deliberate scope decision to
  avoid building two separate UI elements for closely related information — revisit only if a
  future screen needs filtering without wanting a visible legend, or vice versa.

## Breadcrumb

- **Purpose**: scope context above a screen title (Insights graph's "Operations / Customer
  onboarding / Investigation #042").
- **Props**: `segments: string[]`, `current: string`.
- **Data source caveat**: in the mockup this was static/illustrative text — no real "Investigation"
  concept exists in the domain. Before using this pattern for real, confirm what entity it should
  actually reflect (an Organization/Department/ResearchQuestion?) rather than inventing an
  "Investigation" grouping the backend doesn't have — `OPEN QUESTION`.

## Modals / Drawers (specified, not built)

- **Recommendation**: prefer a drawer (like DetailPanel above) over a modal wherever the action is
  "see more about this item while keeping context," per the project owner's own stated preference
  in the Insights-graph redesign brief. Reserve modals for genuinely blocking, short confirmations
  (e.g. a destructive-action confirmation) — none exist in the mockup yet.
- **Not specified**: exact modal chrome, backdrop behavior, focus-trap requirements — needed before
  the first real confirmation dialog ships.

## SettingsGroupHeader / SettingsItemRow

- **Purpose**: the Settings shell's grouped, honestly-unavailable item list.
- **Props**: `groupLabel`; `name`, `description`, `status: 'soon'` (only status implemented; a real
  future state once any config is persisted is undesigned).
