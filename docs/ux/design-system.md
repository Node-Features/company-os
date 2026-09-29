# Design System Specification

Status: DRAFT (2026-09-29, awaiting project-owner review)

Token values below are what the reviewed mockup actually used, derived from the project owner's
`KNAUF` project (a separate Next.js/`shadcn-ui`/Tailwind app) rather than invented — see
[`docs/architecture/ui-ux.md`](../architecture/ui-ux.md)'s "Visual direction." Adopting `shadcn/ui`
itself (not just its color values) in `apps/web` is the recommended implementation path so these
tokens map directly onto real component primitives; final library choice is a Phase 10
implementation decision, not fixed here.

## Typography

- **Interface font**: `Inter` (400/500/600/700/800) — all UI text by default.
- **Technical font**: `JetBrains Mono` (400/500) — reserved for identifiers, status codes, and
  content behind a "technical details" toggle. Not used for body copy or headings (this is the
  fix for the superseded cyberpunk direction's monospace-everywhere approach).
- **Scale** (as used): 22px/700 page title, 16px/600 section title, 14–15px/500–600 card title,
  13px/400–500 body, 12px/500 secondary, 11px/600 label (uppercase, 0.03–0.06em letter-spacing),
  10.5px mono status text.

## Spacing

8px base unit. Observed usage: 6/8/10/12/14/16/18/20/22/28/32px. Card internal padding 16–20px;
page padding 28px vertical, 32px horizontal; inter-card gaps 14–16px.

## Layout / grid

- Fixed left sidebar (248–250px) + fluid main content, no responsive collapse designed yet (see
  [`responsive-design.md`](responsive-design.md) for the gap).
- Content grids: 4-column stat rows, 2-column panel rows (1.3–1.6fr : 1fr splits observed), full-
  width tables.
- Max content width: unconstrained today (fills the viewport minus the sidebar) — worth revisiting
  for very wide desktop displays; not decided here.

## Colors

Expressed in `oklch()`, dark-first (no light theme built or requested):

| Token | Value | Use |
|---|---|---|
| `--background` | `oklch(0.145 0 0)` | Page background |
| `--card` | `oklch(0.19 0 0)` | Card/panel/sidebar surface |
| `--foreground` | `oklch(0.97 0 0)` | Primary text |
| `--muted` | `oklch(0.24 0 0)` | Muted surface (progress-bar track, technical-detail block bg) |
| `--muted-foreground` | `oklch(0.66 0 0)` | Secondary text |
| `--accent` | `oklch(0.27 0 0)` | Active nav item background |
| `--border` | `oklch(1 0 0 / 10%)` | All borders |
| `--primary` | `oklch(0.75 0.15 205)` (cyan) | The one accent color — active state, links, primary buttons |
| `--primary-foreground` | `oklch(0.12 0 0)` | Text on `--primary` |
| `--success` | `oklch(0.7 0.16 145)` | Positive status (completed, within-limit, pass) |
| `--warning` | `oklch(0.78 0.14 85)` | Caution status (pending, approaching-limit, active) |
| `--destructive` | `oklch(0.65 0.2 25)` | Negative status (failed, denied) |
| `--violet` | `oklch(0.72 0.17 300)` | Reserved for "Insight"-type content only — not a general accent |

**Restraint rule** (carried over from `ui-ux.md`'s revised direction): `--primary` is the only
color used for interactive/selected state. Status colors (`success`/`warning`/`destructive`)
communicate state, never selection. Do not introduce a new color per feature area — if something
needs a new hue, it needs a documented reason here first.

## Surface / background hierarchy

Two levels only: `--background` (page) and `--card` (everything raised above it — panels, sidebar,
stat cards, table rows use no distinct third level). A third "elevated" level (for a modal/drawer
above a card) is not yet defined — needed once Modals/Drawers are actually built; see
[`component-architecture.md`](component-architecture.md#modals--drawers).

## Borders, shadows, radius

- **Radius**: `10px` shared token for cards/buttons/inputs; `999px` for pills/badges/avatars;
  small connector/progress elements use `2–3px`.
- **Border**: `1px solid var(--border)` on every card/input/table row — no card is borderless.
- **Shadows**: minimal by design (`ui-ux.md`'s "minimal shadows" instruction from the project
  owner's brief). The one shadow in active use is a restrained glow (`0 0 16–24px` at the token's
  own color, low opacity) on the active nav item and primary buttons — not a generic drop shadow.
  Avoid adding drop shadows to plain cards; the border is the only separation most surfaces need.

## Core components (see `component-architecture.md` for full specs)

- **Button**: primary (filled `--primary`), ghost-danger (outline `--destructive`), ghost/toolbar
  (outline `--border`, muted text). No secondary/filled-neutral variant defined yet.
- **Input**: `--muted` background, `--border` border, 7–8px vertical padding, 8px radius. Only one
  instance built so far (Insights graph search, unwired) — full input states (focus, error,
  disabled) are not yet specified; `OPEN QUESTION` for the first screen that actually needs a form.
- **Select**: not yet built or specified anywhere in the mockup — `OPEN QUESTION`.
- **Table**: header row (11px/600 muted, letter-spaced), body rows (13px, 1px border-bottom,
  13–14px vertical padding), no zebra striping, no row hover state defined yet.
- **Card**: see Surface hierarchy above; optional `card-hover` lift+glow on hover, used for
  clickable/expandable cards, not static stat tiles.
- **Tabs**: not yet built — Settings' grouped sections and Research's single-screen-with-sections
  approach were chosen specifically to avoid needing Tabs this round; `OPEN QUESTION` for when a
  screen genuinely needs them (e.g. Knowledge library vs. review queue, per `screen-inventory.md`).
- **Badges / status indicators**: two patterns in active use, **do not conflate them**:
  1. **Bordered glow pill** (`badge()` pattern: tinted background, colored text, matching border,
     soft glow) — for short categorical status on its own (`RUNNING`, `COMPLETED`, `PENDING`).
  2. **Dot + mono label** (introduced 2026-09-29, replacing a bordered pill that was visually too
     heavy paired with a progress bar) — for status shown *alongside* another visual (a progress
     bar, a timeline node). Use dot+label whenever status accompanies another indicator in the same
     compact space; use the bordered pill when status is the only thing in that space.
- **Alerts**: one pattern (Budget's cost-constraint banner) — icon + message, `--card` background,
  no colored border-left accent (removed 2026-09-29 per project-owner feedback — see git history
  on the mockup). Keep alerts flat; let the icon and message carry the severity, not a colored edge.
- **Tooltips**: not built or specified — `OPEN QUESTION`.
- **Modals**: not built. `component-architecture.md#modals--drawers` specifies the *intended*
  pattern (drawer preferred over modal, per the Insights graph's detail panel precedent) for when
  one is actually needed.
- **Drawers**: the Insights graph's right-side detail panel is the one built precedent — persistent
  (not dismissible/overlay), 300px fixed width, sticky to viewport top.
- **Navigation**: grouped sidebar, see `information-architecture.md`.
- **Breadcrumbs**: one instance built (Insights graph's "Operations / Customer onboarding /
  Investigation #042") — mono font, muted, `/` separator at 50% opacity. Not yet used elsewhere;
  worth applying anywhere a screen represents one scoped investigation/entity rather than a list.
- **Pagination**: not built or needed yet — every list in the mockup is small/illustrative. Real
  volume (e.g. a mature Knowledge library) will need it; not designed here.
- **Charts**: see [`visualization-guidelines.md`](visualization-guidelines.md).
- **Timelines**: vertical spine + branch-tick pattern, built for Research's timeline and the
  Workflow execution thread — see `component-architecture.md#timeline--causal-chain-node`.
- **Graph nodes**: see `component-architecture.md#graph-node`.
- **Progress indicators**: thin (5–6px) rounded bar, `--muted` track, status-colored fill — paired
  with the dot+label status pattern above, never the bordered pill.

## Open questions

- OPEN QUESTION: no light theme exists or has been requested — confirm dark-only is acceptable
  long-term, or scope a light theme before it becomes expensive to retrofit.
- OPEN QUESTION: Select, Tabs, Tooltip, and full Input states are unspecified because nothing built
  so far needed them — the first screen that does should trigger filling these in, not invent them
  ad hoc in that screen's own code.
