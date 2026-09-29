# UX & Product Design Documentation

Status: DRAFT (2026-09-29, awaiting project-owner review)

This directory owns the **product/UX-process layer** for CompanyOS's UI (`apps/web`, or its
eventual replacement): personas and goals, information architecture, the full screen inventory,
user flows, the reusable design system and component set, visualization/interaction/responsive/
accessibility guidance, standard UX states, the (aspirational) permissions model, and the
design-to-data mapping that ties UI elements back to real backend contracts.

It does **not** own the architectural decision itself — [`docs/architecture/ui-ux.md`](../architecture/ui-ux.md)
remains the canonical source for the screen inventory's backend-status column, the decided visual
direction, and the cross-cutting interaction invariants every screen must follow. Where this
directory's documents restate part of that inventory (e.g. `screen-inventory.md`'s per-screen
detail), `ui-ux.md` is the tie-breaker if the two ever disagree — this directory elaborates,
`ui-ux.md` decides.

All documents here were produced from a design-mockup review the project owner directed and
iterated on directly (recorded in `ui-ux.md`'s 2026-09-29 revision note), not from independent
invention — they systematize decisions already made rather than proposing new ones, except where
explicitly marked `OPEN QUESTION`.

| Document | Purpose | Status | Read when |
|---|---|---|---|
| [`product-overview.md`](product-overview.md) | Product purpose, personas, goals, terminology | DRAFT | Before any UX or product-copy decision. |
| [`information-architecture.md`](information-architecture.md) | App hierarchy, navigation, cross-module relationships | DRAFT | Before adding a screen or changing navigation. |
| [`screen-inventory.md`](screen-inventory.md) | Full per-screen spec: purpose, actions, data, states, permissions | DRAFT | Before building or redesigning any specific screen. |
| [`user-flows.md`](user-flows.md) | End-to-end flows across screens, as Mermaid diagrams | DRAFT | Before designing a multi-step interaction. |
| [`design-system.md`](design-system.md) | Tokens: typography, color, spacing, radius, and core UI primitives | DRAFT | Before styling any component. |
| [`component-architecture.md`](component-architecture.md) | Reusable component specs: variants, states, props, a11y | DRAFT | Before building or reusing a shared component. |
| [`visualization-guidelines.md`](visualization-guidelines.md) | When to use a card vs. table vs. chart vs. graph, and why | DRAFT | Before adding any chart, graph, or dashboard element. |
| [`interaction-animation.md`](interaction-animation.md) | Motion principles, durations, easing, when not to animate | DRAFT | Before adding any transition or animation. |
| [`responsive-design.md`](responsive-design.md) | Desktop/tablet/mobile adaptation per component class | DRAFT | Before assuming desktop-only layout is acceptable. |
| [`accessibility.md`](accessibility.md) | Keyboard, focus, contrast, screen-reader, motion requirements | DRAFT | Before shipping any interactive component. |
| [`ux-states.md`](ux-states.md) | Standard loading/empty/error/success/etc. patterns | DRAFT | Before writing a one-off state instead of the standard one. |
| [`permissions.md`](permissions.md) | Role-based UX — mostly aspirational, blocked on real RBAC | DRAFT | Before assuming a role/visibility rule that isn't built yet. |
| [`data-ui-mapping.md`](data-ui-mapping.md) | UI element → data source → entity → action → state | DRAFT | Before wiring a screen to real endpoints. |
| [`ui-implementation-plan.md`](ui-implementation-plan.md) | Build order, dependencies, phases, definition of done | DRAFT | Before starting any Phase 10 implementation slice. |

## Dependencies

- [`docs/architecture/ui-ux.md`](../architecture/ui-ux.md) — canonical screen inventory, visual
  direction, and cross-cutting invariants; this directory's tie-breaker.
- [`docs/domain/`](../domain/README.md) — canonical entity definitions this directory maps UI onto.
- [`docs/departments/`](../departments/README.md) — department responsibility docs the Research/M&E/
  Finance screens surface.
- [`ROADMAP.md`](../../ROADMAP.md) Phase 10 — the authorized implementation slices this directory's
  `ui-implementation-plan.md` sequences work against.
