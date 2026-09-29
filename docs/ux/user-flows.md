# User Flows

Status: DRAFT (2026-09-29, awaiting project-owner review)

Each flow below documents what the user is trying to accomplish, not just page-to-page navigation.
Flows are grounded in real backend actions per [`screen-inventory.md`](screen-inventory.md) —
where a step has no real backend yet, it's marked `BLOCKED` rather than drawn as if it works.

## Reviewing a research investigation

The primary Research flow today is **observation, not authoring** — a human doesn't submit signals
or write findings from this UI; Research/M&E/Finance are `AUTOMATIC`-autonomy Agent/Service actions
(`docs/departments/research.md`). The human flow is understanding what happened:

```mermaid
flowchart TD
  A[Land on Overview] --> B{Latest discovery card present?}
  B -- yes --> C["Click 'Open research →'"]
  B -- no --> D[Navigate to Research via sidebar]
  C --> E[Research screen: timeline + Discoveries list]
  D --> E
  E --> F{Discovery led to a Decision?}
  F -- yes --> G["Click 'View decision →'"]
  F -- no --> H[Read Discovery/Insight text, evidence count, confidence]
  G --> I[Decisions screen, that Decision's trace expanded]
```

## Making a decision (resolving a proposed Objective)

```mermaid
flowchart TD
  A[Approvals screen] --> B[See pending item: 'Propose new objective from Q3 research']
  B --> C{Need more context?}
  C -- yes --> D["Click 'View technical details'"]
  D --> E[See raw action, requester, requested-at]
  C -- no --> F[Read plain-language title + detail]
  E --> F
  F --> G{Decide}
  G -- Approve --> H[POST /v1/approvals/id/decide — approve]
  G -- Decline --> I[POST /v1/approvals/id/decide — reject]
  H --> J[Objective status becomes APPROVED; item leaves the pending list]
  I --> K[Objective status becomes REJECTED; item leaves the pending list]
```

Backend reality check: `ResolveApproval`'s reject branch is a real state transition for
`ApproveKnowledgeItem` (Knowledge) but a no-op for `ProposeObjective`/`CancelWorkflow` beyond
closing the Approval — a rejected Objective proposal simply never becomes a real Objective row
(`objective_propose.go`). The UI's "declined" messaging should say that plainly, not imply an
Objective was created and then rejected.

## Approving or declining any governed action (generic Approval flow)

Same shape as above, generalized — this is the one real interaction pattern every `REQUIRE_APPROVAL`
action shares (`CancelWorkflow`, `ProposeObjective`, `ApproveKnowledgeItem` today):

```mermaid
sequenceDiagram
  participant U as User
  participant UI as Approvals screen
  participant API as companyd
  U->>UI: Open Approvals
  UI->>API: GET /v1/approvals?status=PENDING (BLOCKED — no endpoint yet)
  API-->>UI: List of pending Approvals
  U->>UI: Click Approve or Decline
  UI->>API: POST /v1/approvals/{id}/decide
  API-->>UI: 200 (resolved) or 409 (already decided)
  UI->>U: Remove from list, or show "already resolved" if 409
```

## Reviewing a Knowledge candidate

```mermaid
flowchart TD
  A[Knowledge review queue] --> B[See DRAFT item, with required purpose shown]
  B --> C{Are you the item's own producer?}
  C -- yes --> D[Approve/Decline disabled — separation of duties]
  C -- no --> E{Decide}
  E -- Approve --> F[POST /v1/knowledge/items/id/request-approval, then decide=approve]
  E -- Decline --> G[... then decide=reject]
  F --> H[Item status APPROVED, appears in Knowledge library]
  G --> I[Item status REJECTED]
```

## Checking capability performance (Monitoring & Evaluation)

```mermaid
flowchart TD
  A[Navigate to Monitoring and Evaluation] --> B[See KPI cards per capability]
  B --> C{Classification}
  C -- PASS --> D[Green — no action implied]
  C -- FAIL --> E[Red — human may want to investigate; no in-UI action wired]
  C -- INCONCLUSIVE --> F[Grey — insufficient samples, not a verdict]
```

## Tracking budget and cost constraints

```mermaid
flowchart TD
  A[Navigate to Budget] --> B[See total allocated/spent/remaining/utilization]
  B --> C{Any department APPROACHING_LIMIT or EXCEEDED?}
  C -- yes --> D[Alert banner names the department]
  C -- no --> E[No alert shown]
  B --> F[Per-department table: bar + status dot+label]
```

## Reviewing the activity/audit log

```mermaid
flowchart TD
  A[Navigate to Activity log] --> B[Entries grouped by day]
  B --> C[Each entry: actor, plain verb, GOVERNED tag if applicable, time]
  C --> D{Need the raw record?}
  D -- yes --> E[Technical-detail toggle — not yet built, endpoint BLOCKED]
  D -- no --> F[Read plain-language line]
```

## Tracing a decision through the Insights graph

```mermaid
flowchart TD
  A[Navigate to Insights graph] --> B[Vertical causal chain renders, nothing selected]
  B --> C[Click a node]
  C --> D[Ancestors + descendants highlighted, connectors lit, rest dimmed]
  D --> E[Detail panel: type, title, status, Derived from, Led to]
  E --> F{Click same node again?}
  F -- yes --> B
  F -- no --> G[Click a different node]
  G --> D
```

## Managing configuration/settings

`BLOCKED` — no persisted configuration exists anywhere in the backend today. The only honest flow
is: navigate to Settings, see every item marked unavailable. No further flow to document until a
concrete config-backed slice exists (see `ui-implementation-plan.md`).

## Flows deliberately not documented here

The original brief's list included "creating research questions," "adding sources/evidence,"
"recording discoveries," "creating findings," "creating actions," "recording achievements" as
human-initiated flows. None of these are human-initiated in the current backend — they're Agent/
Service actions under `AUTOMATIC` autonomy (`research.signal.submit` etc., all governed but not
gated on human approval). Documenting a human "Create Discovery" flow would describe a UI action
that has no endpoint to call. If human-authored research input becomes a real product requirement,
it needs its own backend design pass before a UI flow can be honestly documented here.
