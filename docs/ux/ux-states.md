# UX States

Status: DRAFT (2026-09-29, awaiting project-owner review)

Standard patterns, consistent across the whole application. A screen should use these rather than
invent its own loading/empty/error treatment — [`screen-inventory.md`](screen-inventory.md)
references these by name per screen.

## Loading

- **Default**: skeleton placeholders matching the real content's shape (skeleton StatCards,
  skeleton list rows) — not a full-page spinner, except where noted below.
- **Exception — Insights graph**: a full-screen "Building evidence graph…" state, because the
  chain's layout depends on every source request resolving together; a partial/progressive render
  would show a visually broken diagram mid-load, unlike Overview's independent panels.
- **Per-panel independence** (Overview specifically): each composed panel loads and fails
  independently — one slow endpoint must not block the whole screen (`screen-inventory.md#overview`).

## Empty

- Every empty state names *why* it's empty and, where a real action would resolve it, *what to do*
  — never a bare "No data." Examples already written: "No workflows yet — create one to get
  started," "Nothing needs your approval right now," "No connected insights yet. Signals, evidence,
  decisions, and outcomes will appear here as CompanyOS learns from operations."
- Distinguish **empty-because-nothing-exists-yet** from **empty-because-filtered** (e.g. Insights
  graph with all type filters off) — the latter should say so and offer to reset filters, not
  imply there's genuinely nothing there.

## Error

- Inline retry, scoped to the failed panel/request — never a full-page error unless the entire
  screen depends on one request (Insights graph again, and single-entity screens like Workflow
  console or Objective view).
- Show what failed in plain language; a technical error code/message can sit behind the same
  "technical details" disclosure pattern used for governance detail, for consistency.

## Success

- No form-submission success pattern exists yet (nothing mutates except Approve/Decline, which
  simply removes the resolved item from its list — treated as sufficient feedback, no separate
  toast). Define a toast/inline-confirmation pattern when the first real form ships (see
  `interaction-animation.md`).

## Partial data

- Applies wherever a screen composes multiple sources (Overview; the Insights graph's per-node
  fetch-and-assemble). A partially-loaded/partially-failed composition should show what it has and
  flag what's missing, per-panel — this is the same mechanism as the Loading/Error states above,
  not a fourth separate state.

## No permissions

- Not yet buildable — no real permission system exists (`permissions.md`). Specify: when it does,
  a no-permission screen/section states plainly what's restricted and why, and never silently
  hides a nav item a user might reasonably expect to exist (per `ui-ux.md`'s honesty principle) —
  hidden vs. visible-but-restricted is itself a product decision to make when RBAC lands, not
  assumed here.

## Offline / unavailable data

- Not addressed in the mockup. Minimum bar: a failed network request should be distinguishable from
  a genuine empty state (don't show "No workflows yet" when the real problem is the API is
  unreachable) — reuses the Error state above; a dedicated offline banner is an `OPEN QUESTION`.

## Processing

- Applies to a mutating action in flight (e.g. Approve/Decline while the request is pending). Not
  built in the mockup (static canvas). Minimum bar: disable the triggering button and show inline
  progress on it (not a full-screen block) so the rest of the screen stays usable.

## Pending approval

- This is a first-class *content* state, not just a loading variant — an item whose
  `Outcome`/`Status` is `PENDING`/`APPROVAL_REQUIRED` renders with the `warning` status color and,
  where relevant, an explicit "awaiting approval" label — already the pattern used throughout
  Decisions/Approvals; keep it consistent everywhere an Approval-gated entity appears (a Workflow
  card, a Decision card, an Objective view).
