# Fixed: Authority Model Formalization

Status: IMPLEMENTED (2026-08-24) — implemented and verified against the real database. **Not yet committed to git** as of this writing — check `git status` before assuming this is landed on a branch.

This is the implementation record for [`docs/adr/ADR-0010-authority-model-formalization.md`](../adr/ADR-0010-authority-model-formalization.md), which is the authoritative decision record — read it first. This document is the "what actually shipped, what it proves, what's still open" record, in the same shape as [`fixed-execution-lifecycle-hardening.md`](fixed-execution-lifecycle-hardening.md) and [`fixed-runtime-resilience.md`](fixed-runtime-resilience.md).

## What was found (establishing current semantics, before any change)

A full read of `internal/governance`, the domain types for `policy`/`principal`/`approval`/`command`/`organization`, every Application-layer governance helper, `ResolveApproval`'s repository implementation, `governance.md`, `ADR-0008`, and this audit's own `backlog-p2-p4.md`/`findings.md`, confirmed:

- LEGALITY (Kernel), AUTHORITY (Governance), and APPROVAL (the Approval domain) were already cleanly separated architecturally — no exception found in any of `internal/kernel/{workflow,objective,knowledge}`.
- `policy.AutonomyHumanOnly` was genuinely dead: no policy rule ever set it, and `governance.Request` carried no principal-*kind* signal at all, so the branch that would handle it unconditionally denied regardless of who was actually asking.
- `docs/architecture/governance.md`'s "Knowledge approval boundary" section described `knowledge.approve` as a direct, one-step human decision; the implementation (`knowledge_review.go`, `policy.go`) was a two-step `REQUIRE_APPROVAL` queue. Structurally different models, not a wording mismatch.
- The deciding principal's `Kind` on `POST /v1/approvals/{approvalId}/decide` was never checked for any `CommandType` — true only by the accident that `fixtures.ApproverPrincipal()` (Kind HUMAN) is the only decider that has ever existed.
- `evaluateObjectiveProposalGovernance` never set `ExcludedPrincipalID` — self-approval protection for Objective proposals held only by fixture-pairing accident, same root cause.
- `command.PendingCommand.ExpiresAt` existed in the domain type and DB schema but was never set or enforced anywhere — an Approval could sit `PENDING` forever.
- The real Principal/Kind resolution chain (`RequireHumanAuth` → `ResolvePrincipal` → `identity.Resolver` → real `principal.Principal{Kind}` in context) already existed end-to-end for Objective/Knowledge HTTP handlers (`requirePrincipal` already read the real resolved Principal, just discarded `.Kind`) — closing the `RequestingPrincipalKind` gap for these two paths needed no new resolution machinery, only reading a field that was already there.

## What was implemented

1. **`policy.Decision`**: four values (`AUTOMATIC`/`REQUIRE_APPROVAL`/`HUMAN_ONLY`/`DENIED`), replacing `ALLOW`/`DENY`/`REQUIRE_APPROVAL`. New `Decision.Allows() bool` helper centralizes the "may this proceed" gate (`runtime.Runtime.execute`, `kernel/workflow.verifyAllow` both use it instead of a literal equality check, so `HUMAN_ONLY` is never mistaken for unauthorized).
2. **`governance.Request.RequestingPrincipalKind principal.Kind`** (new field): `Evaluate`'s `HUMAN_ONLY` branch now checks it for real — a human requester gets `DecisionHumanOnly`; anyone/anything else (including an unset zero-value Kind, which fails closed) gets `DecisionDenied`. Threaded from `fixtures.TriggerPrincipal().Kind` for Workflow commands, and from the real resolved Principal's `Kind` (via `httpapi.PrincipalFromContext`, `p.Kind`) for Objective and Knowledge requests.
3. **`knowledge.approve` → `knowledge.review.request`**: Action-string rename only (`command.ActionFor`, `governance/policy.go`'s rule) — `CommandType` Go identifier and the HTTP route are unchanged, so no persisted-payload deserialization risk. `RequestingPrincipalKind` was added to both `ObjectiveProposalCommandEnvelope` and `KnowledgeApprovalCommandEnvelope` (persisted, so a resumed replay evaluates governance against the same requester-kind fact the original request did).
4. **`ResolveApproval` restructured** (`ports.PendingCommandRepository`, `pending_repo.go`): signature changed from a bare `decidedByPrincipalID uuid.UUID` to a full `decidingPrincipal principal.Principal`. Internally restructured from a single `UPDATE ... RETURNING` to `SELECT ... FOR UPDATE` (locking both the Approval and its linked PendingCommand) before deciding anything — same one transaction, no new TOCTOU window — then unconditionally, for every `CommandType`:
   - not found / not `PENDING` → `ports.ErrConflict` (unchanged from before)
   - `PendingCommand.ExpiresAt` passed → both rows transition to `EXPIRED`, `ports.ErrApprovalExpired`
   - decider equals the original requester → `ports.ErrSelfApproval`, no state change
   - decider's `Kind != principal.KindHuman` → `ports.ErrNonHumanDecider`, no state change
   - otherwise: proceed with the original write, unchanged.
5. **`approvalTTL` (24h)**: new constant in `pipeline.go`; `ExpiresAt` now set on every `PendingCommand` at creation (`evaluateGovernance`, `evaluateObjectiveProposalGovernance`, `evaluateKnowledgeApprovalGovernance` — all three, previously none of them set it).
6. **Migration**: `supabase/migrations/20260824010000_governance_decision_outcomes.sql` widens `governance_decisions.outcome`'s `CHECK` constraint additively — applied to the real dev database via a one-off tool (same precedent as every prior slice), deleted after use. No historical row's `outcome` value was touched.

## Hard invariants — verified, not asserted

| # | Invariant | How it holds |
|---|---|---|
| 1 | LEGALITY/AUTHORITY/APPROVAL not conflated | Confirmed by full-package reads before any change (see above); no code change was needed to fix a conflation, since none existed — only to make the AUTHORITY and APPROVAL layers' own internal logic (HUMAN_ONLY, self-approval) actually correct |
| 2 | Deny by default | Unchanged: `matchRule`'s default-deny still gates every Action; `HUMAN_ONLY`'s new branch fails closed on an unset/non-human Kind, same discipline as every other check in `Evaluate` |
| 3 | Explicit authority / explicit scope | `RequestingPrincipalKind` is explicit input, never inferred from silence — an unset Kind is `DENIED`, not `AUTOMATIC` |
| 4 | No client-controlled authorization decision | `decidingPrincipal` in `ResolveApproval` is never client-supplied — `ResolveApprovalRequest` carries only `ApprovalID`/`Approve`/`Reason`; the decider is always `fixtures.ApproverPrincipal()`, resolved server-side, same as before |
| 5 | Approval cannot be bypassed by retry | An illegitimate resolution attempt (self-approval, non-human decider) leaves the Approval `PENDING`, not consumed — proven by `TestIntegration_ResolveApproval_SelfApprovalDenied`/`NonHumanDeciderDenied` explicitly re-resolving successfully afterward with a legitimate decider |
| 6 | Approval resolution is idempotent | Unchanged CAS guarantee (`ports.ErrConflict` on a second resolution), now inside a `SELECT ... FOR UPDATE` transaction instead of a single `UPDATE`; proven unchanged by the pre-existing `TestIntegration_ResolveApproval_UnknownApprovalConflict`/`ApprovedResumesAndCancels`'s double-resolve assertion, both passing unmodified |
| 7 | Stale approvals rejected | New: `approvalTTL` + `ResolveApproval`'s expiry check, transitioning to `EXPIRED` rather than leaving `PENDING` forever — `TestIntegration_ResolveApproval_ExpiredPendingCommandRejected` |
| 8 | Separation of duties enforced | Self-approval check is now unconditional for every `CommandType`, not opt-in per use case — `TestIntegration_ResolveApproval_SelfApprovalDenied` (fabricated collision, since no real request path can produce one today) |
| 9 | No privilege escalation through provider responses / no LLM-generated authority | Unaffected by this work — no provider response or model output was ever a source of authority in this codebase, confirmed during the establishing-semantics read; nothing in this change introduces one |
| 10 | Concurrent approval resolution safe | `TestIntegration_ResolveApproval_ConcurrentResolutionOneWins` (5 racing goroutines against one real REQUIRE_APPROVAL flow) — exactly 1 accepted, 4 conflicted |

## Tests added

`internal/domain/policy/types_test.go` (new file): `TestDecision_Allows`.

`internal/governance/evaluate_test.go`: `TestEvaluate_HumanOnly_HumanRequesterAllowed`, `TestEvaluate_HumanOnly_NonHumanRequesterDenied`, `TestEvaluate_HumanOnly_UnsetKindDenied`. "Wrong department" (`TestAuthorize_DeniedAction_Fails`) and "wrong resource scope" (`TestEvaluate_MismatchedApprovalEvidence_StaysRequireApproval`) were already covered by pre-existing tests — confirmed, not duplicated.

`internal/application/approval_invariants_integration_test.go` (new file, real database): `TestIntegration_ResolveApproval_SelfApprovalDenied`, `TestIntegration_ResolveApproval_NonHumanDeciderDenied`, `TestIntegration_ResolveApproval_ExpiredPendingCommandRejected`, `TestIntegration_ResolveApproval_ConcurrentResolutionOneWins`, plus the shared `fabricatePendingApproval` helper (fabricates a GovernanceDecision/PendingCommand/Approval row combination directly through the repositories — the same "bypass the external boundary to prove the mechanism" pattern this package's other integration tests already use, since no real Application code path can produce a self-approval or non-human-decider collision today).

**"Stale policy" — deliberately not given a dedicated new test.** `PolicyVersion` is a hardcoded constant with no second real value to test against without inventing fake policy-versioning machinery this slice doesn't build (matching this codebase's "don't invent state" discipline). The mechanism that would protect against a genuinely stale policy — fresh re-evaluation at resume time, not a replayed/cached decision — is the same mechanism `resumeCancelWorkflow`'s existing state/version-mismatch tests already exercise (a fresh `governance.Evaluate` call happens on every resume, unconditionally); adding a test that changes `PolicyVersion` mid-flow would require building policy versioning infrastructure specifically to test around, which is worse than not testing it.

**Verified**: all new tests pass individually and together against the real database, run at least once each; the full pre-existing `internal/application` Knowledge/Objective/ResolveApproval suites (14 tests) were re-run in full and pass unmodified, proving the `ResolveApproval` SQL restructuring and the `knowledge.approve`→`knowledge.review.request` rename introduced no regression. `go build ./...`/`go vet ./...`/`gofmt -l` clean across the whole module.

## Files changed

`domain/policy/types.go` (`Decision` rename + `Allows()`) · `domain/command/types.go` (`ActionFor` rename, `RequestingPrincipalKind` on two envelopes) · `governance/evaluate.go` (`RequestingPrincipalKind` field, real `HUMAN_ONLY` branch, restructured doc comments mapping onto `authority-model.md`'s pipeline stages) · `governance/policy.go` (rule rename) · `ports/persistence.go` (three new sentinel errors, `ResolveApproval` signature) · `adapters/persistence/supabase/pending_repo.go` (`ResolveApproval` restructured) · `application/{pipeline,automatic_governance,objective_governance,objective_propose,objective_request,knowledge_governance,knowledge_review,approval_resolve}.go` · `adapters/httpapi/{objectives,knowledge}.go` (source `PrincipalKind` from the already-resolved context Principal) · `runtime/runtime.go`, `kernel/workflow/decision.go` (`.Allows()` gate) · new: `domain/policy/types_test.go`, `application/approval_invariants_integration_test.go` · extended: `governance/evaluate_test.go`, existing `*_test.go` files mechanically renamed to the new `Decision` vocabulary · new migration `supabase/migrations/20260824010000_governance_decision_outcomes.sql` (applied to the real dev database) · new docs: `architecture/authority-model.md`, `adr/ADR-0010-authority-model-formalization.md` · updated docs: `architecture/governance.md`, `architecture/knowledge.md`, `adr/ADR-0008-authority-capability-model.md` (flipped to `APPROVED`), `adr/README.md`, `audit/backlog-p2-p4.md`.

## Dependencies

- [`docs/adr/ADR-0010-authority-model-formalization.md`](../adr/ADR-0010-authority-model-formalization.md) — the decision record this implements
- [`docs/architecture/authority-model.md`](../architecture/authority-model.md) — the formalized model this produces
- [`findings.md`](findings.md), [`backlog-p2-p4.md`](backlog-p2-p4.md) (both resolved rows updated)
- [`gap-approval-principal-attribution.md`](gap-approval-principal-attribution.md) — remains open; this work makes the decider-Kind/self-approval checks real and structural, but the *requester* identity for Workflow commands is still `fixtures.Registry`, not a real authenticated caller
