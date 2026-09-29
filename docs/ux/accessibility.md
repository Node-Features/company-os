# Accessibility Guidelines

Status: DRAFT (2026-09-29, awaiting project-owner review)

**Honest starting point**: the reviewed mockup was built in a static design-canvas format without
an accessibility audit. Several real gaps were identified while writing this document (listed
below as required fixes, not optional polish) rather than glossed over. Nothing here should be read
as "already done."

## Keyboard navigation

- Every interactive element in the mockup is a real `<button>`/`<a>`/`<input>` (never a `<div
  onClick>`), which is the right foundation — semantic elements are keyboard-focusable and
  activatable by default.
- **Gap, required fix**: GraphNodes, NavItems, and Cards using the `card-hover` pattern have no
  visible `:focus-visible` style defined anywhere in the mockup's CSS. Add one (a visible outline
  or the same border/glow treatment used for `:hover`) before implementation ships.
- **Gap**: no explicit tab-order review was performed — the DOM order in the mockup should produce
  a reasonable default order (sidebar → topbar → content top-to-bottom), but this needs verification
  once built, especially for the Insights graph's multi-node layout.
- GraphNode selection must work via Enter/Space, not click-only (see `component-architecture.md`).

## Focus states

- Required: every `Button`, `NavItem`, `GraphNode`, `FilterChip`, and the `PlainLanguageDisclosure`
  toggle needs a visible focus ring distinct from its hover state (hover ≠ focus — a mouse user
  hovering doesn't prove a keyboard user can see where focus is).
- Recommended: reuse the existing `--primary` accent as the focus-ring color (`2px solid var(
  --primary)` with a small offset) for visual consistency with the "selected/active" language
  already established.

## Contrast

- `design-system.md`'s color tokens need verification against real contrast ratios once implemented
  — the oklch values were chosen for visual restraint, not measured for WCAG compliance during the
  mockup pass. Minimum targets (per `ui-ux.md`'s existing invariant, carried forward): 4.5:1 for
  body text, 3:1 for text ≥24px/bold ≥19px.
- Specific risk areas to check first: `--muted-foreground` text on `--card`/`--muted` backgrounds
  (used extensively for secondary text — the token most likely to fail at small sizes), and status
  colors (`--warning`, `--success`) used as text color on `--card`.
- Status must never be conveyed by color alone — every status use in the mockup already pairs color
  with text (a label, a badge word) or an icon; preserve this pattern in every new screen, don't
  regress to a color-only dot.

## Screen readers

- **Gap, required fix**: icon-only buttons (topbar notification bell, Insights graph's Fit/Reset
  when icon-only, any future icon-only affordance) need an explicit `aria-label` — none were added
  in the mockup since DC markup doesn't enforce this.
- **Gap, required fix**: the `PlainLanguageDisclosure` toggle needs `aria-expanded` and
  `aria-controls` linking it to the revealed technical-detail block.
- Decorative icons (the small dots, the icon inside a StatCard) should be `aria-hidden="true"` —
  the adjacent text already carries the meaning.

## Semantic structure

- Use real heading levels (`h1` page title, `h2` section titles) rather than styled `div`s — the
  mockup used styled `div`s throughout (a constraint of the design-canvas format, not a
  recommendation) — implementation should correct this.
- Tables should use real `<table>`/`<th>`/`<td>` markup, not styled flex rows (used in the mockup
  for layout convenience) — a real table gives screen-reader users row/column navigation the flex
  version doesn't.

## Form errors

Not yet applicable — no forms exist beyond the unwired search input. When the first real form ships
(Settings' eventual config forms, most likely), define inline error messaging associated via
`aria-describedby`, not color-only validation state.

## Status communication

Already covered under Contrast above — reiterated here because it's an accessibility requirement,
not just a visual one: every status badge/dot pairs color with text; do not introduce a
color-only status indicator anywhere.

## Motion reduction

- **Gap, required fix**: none of `interaction-animation.md`'s transitions were wrapped in a
  `prefers-reduced-motion` check in the mockup (a static canvas can't meaningfully test this) — this
  is a hard requirement for implementation, not optional, per `ui-ux.md`'s own existing invariant.
- The one looping animation (the "LIVE" pulse) should stop entirely under reduced motion, not just
  slow down — it's decorative-adjacent even though it signals something real.

## Accessible charts and graphs

- Every StatCard/ProgressBar/mini-bar-chart value must also exist as real text nearby (already true
  throughout the mockup — numbers are always printed, not chart-only) — preserve this; never let a
  bar's height be the only representation of a value.
- The Insights graph is the hardest case: a screen-reader user cannot perceive the vertical
  spatial-relationship layout. **Required**: an accessible alternative — at minimum, the
  DetailPanel's "Derived from"/"Led to" text lists (already built) should be independently
  reachable via keyboard/screen-reader without requiring the visual graph to make sense first; a
  more complete solution (e.g. an optional list/table view of the same chain) is an `OPEN QUESTION`
  for implementation to resolve, not decided here.
