# Design-to-Data Mapping

Status: DRAFT (2026-09-29, awaiting project-owner review)

For each major screen: **UI element → data source → entity → action → resulting state.** This is
the concrete wiring spec implementation should follow — it does not restate
[`screen-inventory.md`](screen-inventory.md)'s narrative fields, only the data path.

## Research

| UI element | Data source | Entity | Action (if any) | Resulting state |
|---|---|---|---|---|
| Stat chips (open questions, evidence, discoveries, insights) | `GET /v1/research/questions/{id}` | `ResearchQuestion` (nested `Evidence`/`Finding`/`Recommendation`) | read-only | display only |
| Timeline rows | Same response, reordered client-side by timestamp | `Signal`, `ResearchQuestion`, `Evidence`, `Finding`, `Recommendation` | read-only | display only |
| DiscoveryCard | `Finding` (nested in the same response) | `Finding` | read-only | display only |
| "→ Insight" sub-row | `Recommendation` linked to the `Finding` | `Recommendation` | read-only | display only |
| "View decision →" | Requires resolving which `Objective` (if any) cites this `Finding`/`Recommendation` as `SourceID` — **no endpoint returns this reverse lookup today**; either add one or the UI must know the Objective ID from context | `Objective` | navigate | opens Decisions, that Decision expanded |

## Decisions

| UI element | Data source | Entity | Action | Resulting state |
|---|---|---|---|---|
| Decision list (approved) | `GET /v1/objectives` (added 2026-09-29 — only ever returns already-approved Decisions, see below) | `Objective` | read-only | display only |
| Decision list (pending/declined) | `GET /v1/approvals?status=PENDING,REJECTED` (added 2026-09-29), filtered client-side to `action=objective.propose` | `Approval` | read-only | display only |
| Status badge | `Approval.Status` (`PENDING`/`CONSUMED`≈approved/`REJECTED` — not `Objective.Status`, which stays `PROPOSED` for every real row; see the caveat below) | `Approval` | read-only | `PENDING`/`APPROVED`/`REJECTED` |
| "View trace" expand | Walks `Objective.SourceType`/`SourceID` (approved case) or `Approval.ResourceType`/`ResourceID` (pending/declined case) back to the `Finding`/`Recommendation`/`Evaluation`/`ResourceEvaluation` | multiple, per `SourceType` | read-only | shows trace steps |
| (Decisions screen has no Approve/Decline of its own — see below) | — | — | — | — |

**Caveat, confirmed against `internal/application/objective_propose.go` 2026-09-29**: `CreateObjective`
is only ever called from the approve-resume path — a pending or rejected proposal never creates an
`Objective` row at all, so `GET /v1/objectives` can never itself show a pending/declined Decision.
The two data sources above are both required; neither alone gives the Decisions screen its full
scope. This is also documented in `docs/architecture/ui-ux.md`'s screen-inventory table.

## Approvals

| UI element | Data source | Entity | Action | Resulting state |
|---|---|---|---|---|
| Pending list | `GET /v1/approvals` (added 2026-09-29 — defaults to `PENDING`-only when no `status` is given) | `Approval` | read-only | display only |
| Plain-language title/detail | Derived client-side from `PendingCommand.CommandType` + the governed Action string | `PendingCommand` | — | display only |
| "View technical details" | Same payload, raw fields shown | `PendingCommand`, `GovernanceDecision` | — | reveals detail block |
| Approve button | `POST /v1/approvals/{id}/decide` (approve) | `Approval` | **mutating, governed** | `Approval.Status=APPROVED`; underlying resource (Workflow cancelled / Objective created / KnowledgeItem approved) transitions per its own `CommandType` |
| Decline button | `POST /v1/approvals/{id}/decide` (reject) | `Approval` | **mutating, governed** | `Approval.Status=REJECTED`; underlying resource's reject behavior is `CommandType`-specific (see `user-flows.md`'s note on `ProposeObjective`'s no-op reject) |

## Monitoring & Evaluation

| UI element | Data source | Entity | Action | Resulting state |
|---|---|---|---|---|
| KpiCard (classification, rate, samples) | `GET /v1/me/subjects/{id}/performance-profile` | `PerformanceProfile` | read-only | display only |
| Trend bars | Same response — **no historical time-series exists in `PerformanceProfile`** (it's a single latest-Evaluation-per-subject upsert per `docs/departments/monitoring-evaluation.md`'s scope note); the mockup's trend bars were illustrative, not real | — | — | **`OPEN QUESTION`**: either add real historical tracking or remove trend bars from the shipped screen |
| As-of timestamp | `PerformanceProfile.UpdatedAt` (or equivalent) | `PerformanceProfile` | read-only | display only |

## Budget

| UI element | Data source | Entity | Action | Resulting state |
|---|---|---|---|---|
| Total stat cards | `GET /v1/finance/constraint-status` (aggregated client-side across departments) | `Budget`, `CostConstraint` | read-only | display only |
| Constraint alert banner | Same response, filtered to `ConstraintOutcome != WITHIN_LIMIT` | `CostConstraint` | read-only | display only |
| Per-department table row | Same response, one row per department subject | `Budget`, `ResourceUsage`, `ResourceEvaluation` | read-only | display only |
| ProgressBar + StatusDot | `ResourceUsage` summed vs. `Budget.Amount`; `ConstraintOutcome` | `ResourceEvaluation` | read-only | display only |

## Activity log

| UI element | Data source | Entity | Action | Resulting state |
|---|---|---|---|---|
| Every row | **`BLOCKED`** — no read endpoint over `governance_decisions` exists | `GovernanceDecision` | read-only | N/A until endpoint exists |
| `GOVERNED` tag | `GovernanceDecision.Outcome` present for that action | `GovernanceDecision` | — | display only |
| Technical-detail toggle | Same record's raw fields (action string, decision ID) | `GovernanceDecision` | — | reveals detail block |

## Knowledge

| UI element | Data source | Entity | Action | Resulting state |
|---|---|---|---|---|
| Library list | `GET /v1/knowledge/items` (default `APPROVED`-only) | `KnowledgeItem` | read-only | display only |
| Review queue list | `GET /v1/knowledge/items?statuses=DRAFT&purpose=...` | `KnowledgeItem` | read-only | display only |
| Approve/Reject | `POST /v1/knowledge/items/{id}/request-approval`, then `POST /v1/approvals/{id}/decide` | `KnowledgeItem`, `Approval` | **mutating, governed** | `KnowledgeItem.Status` → `APPROVED`/`REJECTED` |

## Cross-cutting note

Every "mutating, governed" row above already goes through the real Governance pipeline
server-side — the UI's job in every case is send-the-command-and-reflect-the-real-outcome, never
optimistic-update-and-hope (`ui-ux.md` invariant 1). No row in this table describes a client-side
state mutation that isn't backed by a real API call.
