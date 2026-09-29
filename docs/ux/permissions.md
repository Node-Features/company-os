# Permissions & Role-Based UX

Status: DRAFT (2026-09-29, awaiting project-owner review) — **mostly aspirational**

**This document is deliberately the most speculative one in this directory.** No real
role/permission system exists in CompanyOS today:

- `internal/fixtures.Registry` hardcodes a fixed set of Principals (`TriggerPrincipal`,
  `ApproverPrincipal`) — not real per-user roles.
- `DepartmentMembership` (the domain concept that would scope a human/Agent to a department) is
  unimplemented (`ROADMAP.md` Phase 11 Slice 1).
- Governance's policy rules (`internal/governance/policy.go`) are hardcoded Go, action-scoped, not
  per-user or per-department.

Everything below describes the **intended** shape once that backend exists, so UI work doesn't have
to be redesigned from scratch when it lands — it must not be read as, or implemented as, real
access control today. **Do not build client-side-only permission checks that look like real
security** — per `ui-ux.md` invariant 1, the UI is a client of Application/Governance, never a
bypass or a substitute for it; a hidden button is a UX nicety, not an authorization boundary.

## Intended roles

| Role | Backed by (once real) | Scope |
|---|---|---|
| **Owner** | The Organization's first Human Principal, or an explicit flag — not designed in the domain yet | Everything |
| **Department contributor** | `DepartmentMembership` (`PROPOSED→ACTIVE→SUSPENDED→ENDED`) | Their department(s) only |
| **Agent** | `Principal` of Kind `AGENT` | Not a UI role — acts, doesn't browse; appears as an actor in others' views |

## Intended effect on navigation

- Owner sees every sidebar item (as the mockup shows today).
- A Department contributor would see Research/M&E/Budget/Approvals filtered to their department(s)
  — Overview, Insights graph, and cross-department views (Departments directory) likely stay
  visible but scoped or read-only outside their own department. **Not decided**: whether a
  contributor sees company-wide totals (e.g. Budget's overall utilization) or only their slice —
  `OPEN QUESTION`, matches the same open question already in `screen-inventory.md#budget`.

## Intended effect on visibility

- **Research/M&E access**: a contributor sees their department's data in full detail; cross-
  department visibility (e.g. Engineering seeing Research's raw Evidence) is not designed — default
  assumption is read access to Discoveries/Insights company-wide (they inform company decisions)
  but not raw Evidence, mirroring how Overview previews a Discovery without its full evidence list.
  **Not decided**, flagged as `OPEN QUESTION`.
- **Budget access**: see above.
- **Configuration/Settings access**: Owner-only for anything company-wide (policy, provider
  registration); department-scoped config (if any ever exists) would be that department's
  contributors' concern. Entirely aspirational — no config is persisted at all today.
- **Audit logs**: leans Owner-only by default (an audit trail's value is partly that it isn't
  filtered per-viewer) — **not decided**, `OPEN QUESTION`.

## Intended effect on actions

- **Editing**: not applicable yet — almost nothing in the current screen inventory has an edit
  action at all (Research/M&E/Finance are Agent/Service-authored; Objectives are proposed, not
  edited).
- **Approvals**: `approval.md`'s existing, real, backend-enforced invariant — only a human, never
  the requesting Principal, may decide. The UI should disable Approve/Decline for the viewer if
  they are the requester, once real Principal identity is available client-side (today it isn't —
  the UI has no way to know who "the current user" is as a domain Principal, only as a Supabase
  Auth session). This is the one row in this table closest to being backed by something real.
- **Research access (write)**: no human write path exists (see `product-overview.md`) — not
  applicable until one is designed.
- **Budget access (write)**: `finance.budget.create`/`.constraint.create` exist as governed
  Actions but have no UI form in current scope (`screen-inventory.md#budget`) — when one is built,
  it should be Owner/Finance-contributor gated.

## What the UI must never do

- Never expose an action a Principal cannot actually perform and let a 403 be the user's first
  signal that it was never allowed — but also never *only* hide it client-side as if that were the
  real security boundary (both are true simultaneously: hide for UX, and the backend must still
  enforce it independently, which it already does for every governed Action).
- Never fabricate a role label ("Research Lead") that doesn't correspond to a real
  `DepartmentMembership`/role value once that exists.

## Open questions

- OPEN QUESTION: cross-department read visibility (Research/M&E/Budget) — not decided.
- OPEN QUESTION: Audit log visibility scope — not decided.
- OPEN QUESTION: how "the current user" as a domain `Principal` (not just a Supabase Auth session)
  becomes available to the UI at all — this blocks every row in this document that says "once real
  identity is available client-side," and is itself downstream of `ROADMAP.md` Phase 3 Slice 6's
  Principal-resolution work already having landed server-side but not yet being surfaced to
  `apps/web`.
