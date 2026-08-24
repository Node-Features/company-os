# Gap: Approval and Workflow-Command Principal Attribution

Status: APPROVED (2026-08-24) — problem statement and remediation plan approved. Implementation still requires the project owner to explicitly select and authorize this slice before any code changes, per this repository's doc-gate convention.

Severity: P1 — threatens runtime reliability and the correctness of the authorization model as soon as more than one human operator exists. See [`findings.md`](findings.md) §1 ("Identity/Authorization"), §2 (invariant 2 in the "does not hold" list).

## Problem

Two related, independently-confirmed gaps in who governance actually evaluates:

**1. Approval decisions are not attributed to the real caller.** `POST /v1/approvals/{approvalId}/decide` runs behind the same `RequireHumanAuth`→`ResolvePrincipal` middleware as every other route, but [`httpapi/approvals.go`](../../apps/companyd/internal/adapters/httpapi/approvals.go) never reads the resolved Principal from context. [`application/approval_resolve.go:36`](../../apps/companyd/internal/application/approval_resolve.go) always attributes the decision to `a.Fixtures.ApproverPrincipal()`, a fixed fixture. **Any authenticated human can decide any pending approval, including one they themselves requested** — separation-of-duties as implemented today (`ExcludedPrincipalID`/`ResourceOwnerPrincipalID` in Governance, [`domain/knowledge.md`](../domain/knowledge.md) step 6) is structurally sound only because the fixture happens to be a fixed `Kind: HUMAN` principal distinct from the fixed `TriggerPrincipal` — it does not verify anything about the real caller.

**2. Workflow commands act as a fixture identity, not the authenticated caller.** `workflow_start.go:46,111,130` and `workflow_cancel.go:73` all use `reg.TriggerPrincipal().PrincipalID` where `reg := a.Fixtures`, never the context-resolved Principal. Research/M&E/Finance/Objective/Knowledge do the opposite — they consume `PrincipalFromContext` correctly ([`httpapi/research.go:59-66`](../../apps/companyd/internal/adapters/httpapi/research.go): "Research and M&E actions use the real signed-in Principal, not a fixtures.Registry stand-in"). This is a real two-tier system in the codebase today, not a documentation lag — Workflow was Phase 1/2 work, predating Phase 3 Slice 6's Principal-resolution machinery, and Phase 3 Slice 6's own notes explicitly deferred "rewiring which Principal Governance/Kernel/Application use as the acting identity for [Workflow] commands" as unnumbered future work.

Today this is masked by there being effectively one operator. It stops being masked the moment `ROADMAP.md` Phase 10's approval-inbox UI or any second human account exists — at that point, the audit trail (`PrincipalID` on `GovernanceDecision`, `ExecutionIntent`, the "who initiated this Workflow" ownership check Phase 3 Slice 1 built) becomes meaningless for the most-exercised command family in the system, and the approval-decision gap becomes directly exploitable by any signed-in user.

## Invariant

Restores: governance evaluated against the real, authenticated Principal for every governed action (already true for Research/M&E/Finance/Objective/Knowledge); a human Approval decision durably and correctly attributed to the human who actually made it.

## Proposed approach (plan-level only)

1. **Approval decide**: read `PrincipalFromContext` in `httpapi/approvals.go`, thread it into `ResolveApproval`'s request instead of hardcoding `ApproverPrincipal()`. Decide (project-owner call, not fixed here) whether *any* authenticated human may decide *any* approval, or whether a role/authority check is needed first — [`domain/approval.md`](../domain/approval.md)'s "only a human, not the requester, may approve" needs the requester-exclusion check ([`governance.Request.ExcludedPrincipalID`](../../apps/companyd/internal/governance/evaluate.go), already built for Knowledge) applied generically here too, not just structurally avoided by fixture choice.
2. **Workflow commands**: rewire `workflow_start.go`/`workflow_cancel.go` to use the context-resolved Principal the same way `research.go`/`objectives.go`/`knowledge.go` already do. This directly re-enables the ownership check Phase 3 Slice 1 built (`CANCEL_WORKFLOW`'s "only the initiating Principal" rule, [`kernel/workflow/proposal.go`](../../apps/companyd/internal/kernel/workflow/proposal.go)) to mean something for real distinct humans, not just the one fixture.
3. Both changes are additive to already-correct plumbing (`ResolvePrincipal` middleware, `PrincipalFromContext`) — no new identity infrastructure is needed, only consuming what Phase 3 Slice 6 already built.

## Files likely to change

- [`apps/companyd/internal/adapters/httpapi/approvals.go`](../../apps/companyd/internal/adapters/httpapi/approvals.go)
- [`apps/companyd/internal/application/approval_resolve.go`](../../apps/companyd/internal/application/approval_resolve.go)
- [`apps/companyd/internal/application/workflow_start.go`](../../apps/companyd/internal/application/workflow_start.go)
- [`apps/companyd/internal/application/workflow_cancel.go`](../../apps/companyd/internal/application/workflow_cancel.go)
- [`apps/companyd/internal/adapters/httpapi/workflows.go`](../../apps/companyd/internal/adapters/httpapi/workflows.go)

## Tests required

**Before (regression baseline, expected to demonstrate today's gap):** call `POST /v1/approvals/{id}/decide` as two different authenticated Principals (one the requester, one not) and confirm both currently succeed identically, with the decision attributed to the fixture regardless of caller.

**After:**
- Approval decided by a non-requester Principal succeeds and is attributed to that real Principal (`Approval.DecidedByPrincipalID` matches the caller, not the fixture).
- Approval decided by the requesting Principal is denied (mirrors the existing Knowledge self-review test, `TestIntegration_Knowledge_RequestApproval_SelfReviewDenied`).
- `CancelWorkflow` by a non-initiating, authenticated Principal is denied using the real caller's identity, not the fixture's (extend `TestIntegration_CancelWorkflow_NonInitiatorDenied` to use two distinct real Principals instead of a fabricated `InitiatingPrincipalID`).

## Dependencies

- [`findings.md`](findings.md) §1, §2.
- [`domain/approval.md`](../domain/approval.md), [`domain/principal.md`](../domain/principal.md), [`architecture/identity.md`](../architecture/identity.md), [`architecture/governance.md`](../architecture/governance.md)
- Phase 3 Slice 6 notes in [`.companyos/agent-memory/current-state.md`](../../.companyos/agent-memory/current-state.md) — explicitly deferred this exact rewiring.
- Related: [`ROADMAP.md`](../../ROADMAP.md) Phase 10 (approval-inbox UI) — should not ship ahead of this gap being closed.

## Open questions

- Should any authenticated human be able to decide any approval, or does this require a role/authority concept that doesn't exist yet (`agent-authority.md`'s Role vocabulary is aspirational beyond Governance's existing `policy.Role` matching)? Project-owner decision, not resolved here.
