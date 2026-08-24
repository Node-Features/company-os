# ADR-0010: Authority Model Formalization, Real HUMAN_ONLY, and `knowledge.approve` Resolution

Status: APPROVED

## Context

`docs/audit/backlog-p2-p4.md`'s "Governance-autonomy drift" finding (P2, open since 2026-08-23): `docs/architecture/governance.md:74` describes `knowledge.approve` as a governed Action classified `human_only`, where Governance returns `DENY` outright for any non-human requester. `governance/policy.go:91` (before this ADR) instead classified it `AutonomyApprovalRequired` — a two-step queue (any principal may request review; a separate `/decide` call resolves it later). The same finding named the root cause: `policy.AutonomyHumanOnly` is dead code — no rule ever sets it, `governance.Request` carries no principal-*kind* signal at all, and the branch that would handle it unconditionally denies regardless of who is actually asking.

A second, related finding in the same audit ("Separation-of-duties is opt-in, not structural"): `evaluateObjectiveProposalGovernance` never sets `ExcludedPrincipalID`; self-approval protection for Objective proposals held only because the decider (`fixtures.ApproverPrincipal()`) and every requester fixture happen to always be different Principals — never because anything checked it.

A third, undocumented gap found while resolving the above: the deciding principal's `Kind` on `POST /v1/approvals/{approvalId}/decide` was never verified at all for any `CommandType` — it happens to always be `fixtures.ApproverPrincipal()` (Kind HUMAN), an accident of current fixture wiring, not an enforced invariant. `docs/domain/approval.md`'s "only a human, not the requester, may approve" invariant and `governance.md`'s "agents, services, providers, and models cannot serve as the reviewer" invariant were both true only by construction.

A fourth, smaller gap: `command.PendingCommand.ExpiresAt` exists in the domain type and DB schema but was never set or enforced — an Approval could sit `PENDING` forever.

This request (a full CompanyOS governance formalization, prompted ahead of growing the autonomous workforce) asked explicitly: distinguish LEGALITY/AUTHORITY/APPROVAL; adopt a four-value decision model (`AUTOMATIC`/`REQUIRE_APPROVAL`/`HUMAN_ONLY`/`DENIED`); formalize the Authority tuple (`Principal + Organization + Department + Role + Capability + Action + Resource + Scope + Autonomy Class + Constraints + Evidence`); and resolve the `knowledge.approve` ambiguity explicitly rather than silently.

## Decision

### Decision vocabulary: rename going forward, never rewrite history

`policy.Decision` becomes `AUTOMATIC` / `REQUIRE_APPROVAL` / `HUMAN_ONLY` / `DENIED`, replacing `ALLOW`/`DENY`/`REQUIRE_APPROVAL`. This is additive at the database level (`supabase/migrations/20260824010000_governance_decision_outcomes.sql` widens `governance_decisions.outcome`'s `CHECK` constraint to accept both vocabularies) and forward-only: no historical row's `outcome` value is rewritten. `governance.md`'s own invariant — "audit records are append-only organizational evidence" — means old evidence keeps its original spelling; only new decisions use the new vocabulary. `AutonomyLevel` (`AUTOMATIC`/`APPROVAL_REQUIRED`/`HUMAN_ONLY`) is unchanged; it remains the separate "what class of gate applies" classification, while `Decision` answers "what happened on this specific evaluation."

Alternative considered and rejected: a full rename with backfill (rewriting historical rows to the new spelling). Rejected because it edits historical audit evidence, directly conflicting with the append-only invariant above.

### `HUMAN_ONLY` becomes real

`governance.Request` gains `RequestingPrincipalKind principal.Kind`. When effective autonomy is `HUMAN_ONLY`, `Evaluate` now checks it: a human requester who is otherwise eligible gets `DecisionHumanOnly`; anyone else (including an unset/zero-value Kind, which fails closed rather than being treated as human) gets `DecisionDenied`. Populated today from whatever `principal.Principal` the caller already trusts as the requester — `fixtures.TriggerPrincipal().Kind` for Workflow commands, the real resolved Principal's `Kind` (via `httpapi.PrincipalFromContext`) for Objective and Knowledge requests, which already flowed a real, authenticated Principal's ID (just not Kind) into these two paths before this change.

### `knowledge.approve` split

Renamed to `knowledge.review.request` (Action string only — the `CommandType` Go identifier `ApproveKnowledgeItem` and the HTTP route are unchanged). This action stays `AutonomyApprovalRequired`, matching what was already built and tested: any principal may request review. The actual decide act is protected by the new general Approval-domain invariant below, not a per-action policy rule — this is a superset of what a per-action `HUMAN_ONLY` classification on `knowledge.approve` alone would have given, since it now applies to every `CommandType`'s decide act, not just Knowledge's.

Alternative considered and rejected: keep the single action/queue as built and rewrite `governance.md`'s prose to describe "approval-required-plus-structural-human-decider" as the accepted shape instead. Rejected because the decider-Kind check that would make that shape actually true (rather than true "by construction" only) did not exist before this ADR — the accurate documentation is only accurate once the mechanism below exists.

### Approval-domain invariants become structural, not opt-in

`ports.PendingCommandRepository.ResolveApproval`'s signature changes from a bare `decidedByPrincipalID uuid.UUID` to a full `decidingPrincipal principal.Principal`, restructured internally to `SELECT ... FOR UPDATE` before deciding anything (one atomic transaction, no TOCTOU window). It now unconditionally enforces, for every `CommandType`:

1. Self-approval prohibition (`decidingPrincipal.PrincipalID != Approval.RequestingPrincipalID`, else `ports.ErrSelfApproval`) — closes the Objective "opt-in separation of duties" finding for free, without touching `objective_governance.go` at all.
2. Human decider (`decidingPrincipal.Kind == principal.KindHuman`, else `ports.ErrNonHumanDecider`).
3. Non-expiry (`PendingCommand.ExpiresAt` — new `approvalTTL` constant, 24h, set at creation in `pipeline.go`'s `evaluateGovernance` and its Objective/Knowledge equivalents — else `ports.ErrApprovalExpired`, which also transitions both rows to `EXPIRED`).

An illegitimate attempt (self-approval, non-human decider) leaves the Approval untouched (`PENDING`, resolvable by a legitimate decider later) — it is not burned by the attempt, satisfying "approval cannot be bypassed by retry" without also satisfying an attacker's retry.

### "Capability" collision: no new type

The Authority tuple's Capability component maps onto the existing Role × Action × Resource triple, exactly as `ADR-0008` already concluded when the same collision was raised for the same reason: `docs/domain/capability.md`'s `CapabilityDefinition` already owns that name for a different, already-implemented concept (a provider-independent dispatch contract), and nothing here needs a fifth type — `Action` (a string) and `Resource` already fully cover what "Capability" would have meant.

Department and Scope, the tuple's other two components, are documented as not-yet-real rather than invented: Department exists only via Action-string namespace prefixes; Scope exists only as Organization. See `docs/architecture/authority-model.md` for the full mapping.

## Consequences

### Positive

- `HUMAN_ONLY` is a real, tested, reachable decision outcome instead of dead code that always denied regardless of the actual question being asked.
- Self-approval and human-decider protection is now structural for every governed action with an Approval step, not per-feature opt-in — directly satisfies "do not scatter authorization logic across individual features."
- `knowledge.approve`'s documentation and implementation now describe the same model.
- No historical audit evidence was rewritten.
- Zero new abstraction layers, generic RBAC libraries, or parallel authority frameworks — every change extends existing, already-approved machinery (`governance.Evaluate`, `ResolveApproval`, `ADR-0008`'s Role×Action model).

### Costs and risks

- `RequestingPrincipalKind` is, for Workflow commands, still sourced from a fixture, not a real authenticated caller — `docs/audit/gap-approval-principal-attribution.md` remains open and is not closed by this ADR. The decider-Kind/self-approval checks are real, enforced code, but are not exploitable-worth-attacking in production today (a single fixture pairing) — they become load-bearing once that gap closes, not before.
- `ResolveApproval`'s SQL is now two `SELECT ... FOR UPDATE` queries plus writes instead of one `UPDATE ... RETURNING` — one additional round trip inside the same transaction, not a new transaction class.

## Alternatives rejected by this proposal

- A generic `Subject`/`Role`/`Capability`/`Action`/`Resource` authorization library — rejected for the same reasons `ADR-0008` already gave, reaffirmed here: it would duplicate the existing `governance.Evaluate` model and collide with `docs/domain/capability.md`'s existing, different meaning for "Capability."
- Full Decision-vocabulary rename with historical backfill — rejected, edits append-only audit evidence.
- Per-action `HUMAN_ONLY` classification on `knowledge.approve` alone instead of a general decider-Kind invariant — rejected as strictly weaker (protects one action's decide act instead of all of them) for the same implementation cost.

## Acceptance criteria

- [x] `go build ./...`, `go vet ./...`, `gofmt -l` clean across the whole module.
- [x] Full `internal/governance`, `internal/domain/policy`, and `internal/application` suites pass against the real database, including the pre-existing `TestIntegration_ResolveApproval_*`/Knowledge/Objective tests (proving no regression from the `ResolveApproval` restructuring) and new tests for `HUMAN_ONLY` (human/non-human/unset requester), self-approval, non-human decider, expiry, and concurrent resolution.
- [x] Migration applied to the real database; historical `governance_decisions` rows confirmed unchanged.
- [x] the project owner reviews and explicitly records acceptance by changing `Status: PROPOSED` to `Status: APPROVED` — recorded together with `ADR-0008`'s own outstanding `PROPOSED`→`APPROVED` transition, since this ADR extends it and both describe the same live Role×Action policy-matching model.

## Dependencies

- `docs/architecture/authority-model.md` (new, this ADR's formalization target)
- `docs/architecture/governance.md`, `docs/architecture/knowledge.md`
- `docs/domain/approval.md`, `docs/domain/principal.md`, `docs/domain/command.md`
- `docs/adr/ADR-0008-authority-capability-model.md`
- `docs/audit/backlog-p2-p4.md` ("Governance-autonomy drift", "Separation-of-duties is opt-in" — both resolved by this ADR), `docs/audit/gap-approval-principal-attribution.md` (remains open, not closed by this ADR), `docs/audit/fixed-authority-model-formalization.md`
