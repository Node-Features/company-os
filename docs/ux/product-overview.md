# Product & UX Overview

Status: DRAFT (2026-09-29, awaiting project-owner review)

## Product purpose

CompanyOS is the operating system a human runs a semi-autonomous AI company through. It is not a
chat interface to an AI, and not a generic analytics dashboard — it is the surface where a human
sees what the company (its Workflows, its Research, its Agents) is doing, understands *why* a
governed decision happened, and exercises the specific human-authority moments the backend already
enforces (approving a `REQUIRE_APPROVAL` action, reviewing a Knowledge candidate, watching whether
a decision actually worked). See [`ARCHITECTURE.md`](../../ARCHITECTURE.md) and
[`governance.md`](../architecture/governance.md) for what the backend guarantees; this document
covers how a human perceives and acts on those guarantees.

## Target users / personas

No real role/permission system exists yet (see [`permissions.md`](permissions.md) — blocked on
`DepartmentMembership`), so today there is effectively one persona with full authority. The UI is
designed for the following intended personas, which the backend does not yet distinguish:

- **Owner/Operator** (human, full authority) — the persona the current design targets first. Runs
  the company day to day: reviews what happened, decides what needs a human, occasionally proposes
  or authors an objective directly. Represented in the mockup as "Jordan Diaz."
- **Department contributor** (human, scoped to one or more departments) — not yet possible; needs
  `DepartmentMembership` (`ROADMAP.md` Phase 11 Slice 1). Would see only their department's
  Research/M&E/Budget/Approvals.
- **Agent** (a `Principal` of Kind `AGENT`, per [`principal.md`](../domain/principal.md)) — not a
  UI user, but appears constantly *in* the UI as the actor behind a Signal, Finding, Recommendation,
  or Workflow — the Activity log and Decisions screens must render Agent actors as clearly as human
  ones, never implying a human did something an Agent did.

## Primary user goals

1. **See what's happening** without querying the database — active Workflows, open Research
   questions, pending Decisions.
2. **Understand why** a governed action was denied, approved, or requires approval — the real
   Governance reason, not a generic error (`ui-ux.md` invariant 2).
3. **Decide what needs a human** — approve/decline the specific things `REQUIRE_APPROVAL` gates,
   not everything.
4. **Trace a conclusion back to its evidence** — a Decision, Discovery, or Insight should never be
   a bare assertion; the UI must make "where did this come from" one click away.
5. **Know whether a decision worked** — close the loop from Decision to Outcome to Evaluation where
   that data actually exists (see the open question on unmodeled links in `ui-ux.md`).

## Core problems being solved

- **Accountability for AI-driven action.** Every governed action already produces a real
  `GovernanceDecision`; today that's invisible outside the database. The Activity log and Decisions
  screens exist to make accountability something a human can actually see.
- **Trust without micromanagement.** `AUTOMATIC`-autonomy actions (most of Research/M&E/Finance)
  happen without a human in the loop by design — the UI's job is to make their *results* legible
  after the fact, not to insert approval gates the backend doesn't have.
- **Company-wide visibility without SQL.** Today, everything about CompanyOS's actual operation
  lives in Postgres tables and Go structs. This UI is the only planned way a non-engineer will ever
  see it.

## Key workflows

Detailed step-by-step flows live in [`user-flows.md`](user-flows.md). At a glance:

- Reviewing what a research investigation found and whether it led anywhere.
- Deciding on a proposed Objective (approve/decline, with full trace).
- Approving or declining any pending `REQUIRE_APPROVAL` action.
- Reviewing a Knowledge candidate before it becomes `APPROVED`.
- Checking a capability's performance (M&E) or a department's budget standing.
- Reviewing the activity log to answer "who did what, when, under what authority."

## UX principles

Inherited directly from [`docs/architecture/ui-ux.md`](../architecture/ui-ux.md)'s cross-cutting
invariants (numbered there 1–9) — this document does not restate them, only names the product-level
intent behind the three most consequential for everyday use:

- **Plain language first, technical detail on demand.** A governed decision is a sentence a
  non-engineer can read, with the raw action string/principal ID/timestamp one toggle away — never
  only one or the other.
- **Traceability over assertion.** Anything presented as a conclusion (a Discovery, an Insight, a
  Decision) must be one click from what it's actually based on.
- **Honesty over polish.** A screen with no backend yet says so plainly (`ui-ux.md` invariant 7 and
  this directory's [`ux-states.md`](ux-states.md)) rather than showing fabricated sample data.

## Product terminology

The UI's plain-language vocabulary maps onto real domain types — this table is the single source
of truth for that mapping; nowhere else should redefine it:

| UI term | Domain type | Canonical definition |
|---|---|---|
| Signal | `research.Signal` | [`research` department](../departments/research.md) |
| Question | `research.ResearchQuestion` | [`research` department](../departments/research.md) |
| Evidence | `research.Evidence` | [`research` department](../departments/research.md) |
| Discovery | `research.Finding` | [`research` department](../departments/research.md) |
| Insight | `research.Recommendation` | [`research` department](../departments/research.md) |
| Decision | `objective.Objective` (proposed via the Finding/Recommendation/Evaluation/ResourceEvaluation gate) | [`objective.md`](../domain/objective.md) |
| Action | The `Workflow`/execution a Decision is intended to lead to — **not a real linked field today**, see `ui-ux.md`'s open question | [`workflow.md`](../domain/workflow.md) |
| Outcome | `result.Result` | [`result.md`](../domain/result.md) |
| Evaluation | `monitoringevaluation.Evaluation` | [`monitoring-evaluation` department](../departments/monitoring-evaluation.md) |
| Knowledge item | `knowledge.KnowledgeItem` | [`knowledge.md`](../architecture/knowledge.md) |
| Approval | `approval.Approval` | [`approval.md`](../domain/approval.md) |
| Governance decision | `policy.GovernanceDecision` | [`governance.md`](../architecture/governance.md) |
| Department | `department.Department` | [`department.md`](../domain/department.md) |
| Budget / Cost constraint / Resource usage | `finance.Budget` / `CostConstraint` / `ResourceUsage` / `ResourceEvaluation` | [`finance` department](../departments/finance.md), [`resource.md`](../domain/resource.md) |
