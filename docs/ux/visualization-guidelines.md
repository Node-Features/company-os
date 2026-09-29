# Dashboard & Data Visualization Guidelines

Status: DRAFT (2026-09-29, awaiting project-owner review)

Every visualization must have a clear UX purpose and render only real, reconciled backend data —
never mocked, illustrative, or placeholder values in the shipped product
([`docs/architecture/ui-ux.md`](../architecture/ui-ux.md) invariant 7). This document says *when*
to use which visualization; the `dataviz` skill available in this environment should be consulted
for the actual color/accessibility system when Phase 10 builds these for real
(`ui-ux.md`'s "Visual direction").

## When to use what

| Content | Use | Why | Not |
|---|---|---|---|
| A single scalar with context (Active workflows: 12, ↑3 today) | **StatCard** | Fastest to scan; a chart would over-represent one number | A gauge/donut for a lone number |
| A short, bounded list with status (Recent workflows, pending Decisions) | **List rows with StatusBadge/Dot** | The status *is* the data point; a table adds columns nobody reads | A bar chart of statuses |
| A larger, sortable/filterable dataset (Workflows across every department) | **Table** | Rows need to be scanned and compared, not glanced at | Cards — doesn't scale past ~10 items |
| A trend over a fixed recent window (M&E success rate) | **Small bar chart (sparkline-adjacent)** | Enough to show direction without implying precision a small sample doesn't have | A full line chart with axes for 7 data points |
| A proportion of a whole (Budget utilization) | **ProgressBar** | One glance answers "how much is left" | A pie/donut chart — worse at precise comparison |
| A chronological sequence with cause/effect (Research timeline) | **Vertical timeline** | Order and "what led to what" matter more than absolute time spacing | A calendar/Gantt view — implies scheduling precision that doesn't exist |
| A causal chain across many entity types (Insights graph) | **Vertical causal-chain diagram** (this project's specific graph pattern) | The relationships *are* the content; a table would flatten them into unreadable repetition | A generic force-directed graph — decorative, not more legible (see `ui-ux.md`'s revision note) |
| Relationships between organizational entities (Department/Agent directory) | **Node-link graph** | Real graph-shaped data (`DepartmentMembership` edges) | A table — loses the relationship structure that's the point |
| Geographic data | **Not applicable** | No location data exists anywhere in CompanyOS's domain | Do not add a map "because dashboards have maps" |

## Specific dashboard content

- **Research progress**: stat chips (open questions / evidence / discoveries / insights) + the
  timeline, not a progress bar — there's no single "% complete" a research investigation has.
- **Discoveries / Findings**: list of DiscoveryCards, each showing evidence count and (if present)
  the Insight it produced — never a bare count without the ability to open one.
- **Decisions**: list + expandable trace, not a chart — a decision's value is its reasoning chain,
  not a number.
- **KPIs / M&E results**: KpiCard (classification badge + rate + small trend bars) per capability;
  do not aggregate multiple capabilities into one composite score — `monitoring-evaluation.md`
  deliberately keeps evaluation per-subject, and a fabricated composite would misrepresent that.
- **Budget utilization**: StatCards (totals) + ProgressBar + table (per department) — three
  complementary views of the same numbers, each answering a different question (how are we doing
  overall / at a glance / broken down by department).
- **Risks, Milestones, Achievements**: **no backing domain type exists for any of these** (no
  `Risk`, `Milestone`, or `Achievement` entity anywhere in `docs/domain/`). The mockup's "Recent
  achievements" panel used derived, human-written text, not a real entity — flagged honestly here
  rather than implying a Risk/Milestone tracking feature exists. If these become real product
  requirements, they need their own domain design before a visualization can be honestly specified.
- **Activity**: grouped list (by day), never a chart — an audit trail's value is completeness and
  order, not visual summarization.

## Reconciliation rule

Every number or chart in the shipped product must be traceable to one of the real endpoints listed
in [`screen-inventory.md`](screen-inventory.md)'s "Data sources" field. A screen that cannot yet
reach real data for part of its layout shows that part's honest empty/blocked state
([`ux-states.md`](ux-states.md)) rather than a placeholder chart with invented numbers — this
applies even during implementation with mock data (see
[`ui-implementation-plan.md`](ui-implementation-plan.md)'s mock-data guidance: mock the *shape* of
a real response, never invent a metric that has no real source at all).
