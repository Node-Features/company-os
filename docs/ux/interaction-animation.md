# Interaction & Animation Guidelines

Status: DRAFT (2026-09-29, awaiting project-owner review)

Animation communicates hierarchy, relationship, progress, or change — never exists purely for
decoration (project owner's own instruction, carried directly into this document). Every rule below
was extracted from what the reviewed mockup actually did, not invented ahead of it.

## Principles

1. **Motion explains a state change the user caused or should notice** — a card entering view, a
   node's relationship being revealed, a value updating. It never runs on a loop or "for delight."
2. **Restrained by default.** The mockup's one animated-for-its-own-sake element (a pulsing "LIVE"
   dot) is a deliberate, narrow exception tied to a real invariant (`ui-ux.md`'s "never trust a
   realtime push as authoritative" — the pulse signals "this is live," not decoration).
3. **Respect `prefers-reduced-motion`** on every animation without exception — not audited in the
   mockup itself (a static DC canvas), required in the real implementation.

## Durations and easing

| Motion | Duration | Easing | Used for |
|---|---|---|---|
| Card/section entrance | 450ms | `cubic-bezier(.16,1,.3,1)` (ease-out, slight overshoot-free deceleration) | Panels fading/rising in on screen load, staggered by ~50ms per group |
| Hover lift | 180ms | `cubic-bezier(.16,1,.3,1)` | Card hover (transform + shadow) |
| Button feedback | 100–150ms | default ease | Press (scale down), hover (brightness) |
| Progress bar fill | not built — recommend 400–600ms ease-out | — | Budget/utilization bars on data load |
| Badge/status appearance | 200ms | default ease | A status badge appearing (scale 0.9→1, opacity 0.6→1) — pattern exists in `KNAUF`'s `StatusBadge`, not yet ported |
| Node selection (graph) | instant style change, no transition currently defined | — | **Gap**: selecting a GraphNode should transition its border/opacity/glow, not snap — recommend 150–200ms |
| Pulse (LIVE indicator only) | 2s loop | ease-in-out | The one intentional looping animation in the system |

**Nothing exceeds ~450ms.** This is an operational tool, not a marketing site — motion should never
make the user wait to perceive a state.

## Specific interactions

- **Page/section transitions**: content fades+rises in on nav change (no route-level page
  transition framework specified — a client-side single-page nav switch in the mockup, real router
  transitions are an implementation detail).
- **Navigation**: active nav item's background/border/glow transitions on change (180ms); no
  animated "sliding pill" was built (unlike `KNAUF`'s `layoutId` spring-pill sidebar) — **`OPEN
  QUESTION`**: adopt that pattern (needs `framer-motion` or equivalent) or keep the simpler
  color-transition-only approach used so far.
- **Graph nodes**: selecting highlights ancestors/descendants and dims the rest — should animate
  the opacity/border change (gap noted above), not snap.
- **Timeline events**: no entrance stagger built beyond the shared card-entrance pattern; each
  timeline node could stagger in individually for a stronger "building the chain" feel — not
  decided, low priority.
- **Status changes**: a StatusBadge changing value should use the scale-in pattern above, not just
  swap text instantly.
- **Progress updates**: bar fill should animate on first render and on value change (gap noted
  above) — never jump instantly, since the point is communicating change over time.
- **Expand/collapse**: the "View technical details" / "View trace" disclosures fade+rise in
  (reusing the entrance animation) rather than a height-collapse animation — acceptable and simpler;
  a true height transition is not required.
- **Hover states**: cards that are clickable/expandable lift 2px and gain a soft glow shadow;
  static cards do not hover-react (a card that doesn't respond to hover shouldn't look like it
  might).
- **Loading states**: not built in the mockup (a static canvas has no real async state) — see
  [`ux-states.md`](ux-states.md) for the required skeleton/spinner pattern; skeletons should not
  themselves pulse/shimmer aggressively — a subtle, slow opacity pulse is sufficient.
- **Success/error feedback**: not built — no form submissions exist yet to give feedback on. When
  the first mutating form ships (none in current scope beyond Approve/Decline, which just removes
  the row), define a toast/inline-confirmation pattern then, informed by these duration/easing
  values rather than ad hoc.

## When to avoid animation

- Financial figures and counts never "count up" from zero — this reads as a gimmick and implies a
  precision/liveness the data doesn't have (`ui-ux.md` invariant 6: budget figures are as-of, not
  live).
- Tables never animate row-by-row on load — with real data volume this becomes slow and
  distracting; render immediately.
- Do not add motion to communicate something text/color already communicates unambiguously (e.g. a
  bouncing icon on a `FAILED` badge — the color and label already say it).
