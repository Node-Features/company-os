# Audit Documentation

Status: APPROVED (2026-08-24)

This directory records diagnostic findings from a full-repository architecture-reality audit (2026-08-24) and tracks remediation as implementation-ready slices. It does not define architecture — [`findings.md`](findings.md) cites the `APPROVED` documents in [`architecture/`](../architecture/README.md), [`domain/`](../domain/README.md), and [`adr/`](../adr/README.md) it was checked against, and each gap doc proposes a code change, never a change to already-approved doc content.

**Approval scope:** `APPROVED` here means the problem statements and remediation plans below are accurate and agreed — it does **not** by itself authorize implementation. Per this repository's established convention (design docs approved, then a specific slice separately authorized), the project owner still selects and authorizes a specific `gap-*.md` before any code changes begin.

**Token economy note:** read [`findings.md`](findings.md) once for full evidence and severity rationale. When implementing a specific gap, read only that gap's doc — each is self-contained, cites `findings.md` by section instead of repeating it, and does not require reading the other gap docs.

| Document | Severity | Purpose | Status | Read when |
|---|---|---|---|---|
| [`findings.md`](findings.md) | — | Architecture-reality report, critical invariants, failure-mode matrix, architecture-center component check, orchestration-framework assessment, recommended order | APPROVED | Once, before implementing any gap below. |
| [`gap-execution-lease-reclaim.md`](gap-execution-lease-reclaim.md) | P0 | ExecutionIntents claimed then abandoned (crash or deploy) are never reclaimed; shutdown does not drain in-flight work | IMPLEMENTED | Touching Runtime/daemon shutdown or lease/reclaim — implemented and verified 2026-08-24, read the doc's Implementation section first. |
| [`gap-approval-principal-attribution.md`](gap-approval-principal-attribution.md) | P1 | Workflow commands and Approval decisions execute as a fixed fixture Principal, not the authenticated caller | APPROVED | Implementing per-user authorization or the approval-inbox UI. |
| [`gap-approval-flow-durability.md`](gap-approval-flow-durability.md) | P1 | Crash windows in the REQUIRE_APPROVAL path can orphan or duplicate PendingCommand/Approval rows; an idempotency-store write error is silently swallowed | APPROVED | Touching `pipeline.go`, `approval_resolve.go`, or `application.go`'s idempotency helpers. |
| [`gap-runtime-resilience.md`](gap-runtime-resilience.md) | P1 | No panic recovery or concurrency bound on Runtime's per-intent dispatch goroutines | IMPLEMENTED | Touching `runtime.go`'s dispatch loop — implemented and verified 2026-08-24, read [`fixed-runtime-resilience.md`](fixed-runtime-resilience.md) first. |
| [`gap-ci-integration-coverage.md`](gap-ci-integration-coverage.md) | P1 | CI never runs the database-backed integration suite — the majority of correctness coverage is manual-only | APPROVED | Planning `ROADMAP.md` Phase 8 Slice 1, or before trusting a green CI run as full verification. |
| [`gap-knowledge-source-uniqueness.md`](gap-knowledge-source-uniqueness.md) | P1 | `knowledge_items` has no DB-level defense against a concurrent duplicate-source capture | APPROVED | Touching Knowledge capture or its migrations. |
| [`backlog-p2-p4.md`](backlog-p2-p4.md) | P2-P4 | Consolidated lower-priority findings, one row each | APPROVED | Backlog grooming; not implementation-ready detail — expand a row into its own doc before starting it. |
| [`fixed-test-suite-nondeterminism.md`](fixed-test-suite-nondeterminism.md) | — | Three root causes of real, empirically-reproduced test flakiness (two test-isolation bugs, one real multi-tenancy data-integrity bug) — found, fixed, and verified, not just planned | APPLIED | Touching `internal/application`'s real-database tests, `fixtures.Registry`, or `performance_profiles`; or investigating any new test flakiness (check here first for the pattern before re-diagnosing from scratch). |
| [`fixed-execution-lifecycle-hardening.md`](fixed-execution-lifecycle-hardening.md) | — | Implementation record for `gap-execution-lease-reclaim.md`: lease reclaim, fencing-token safety, shutdown-safe dispatch context, full 10-hard-invariant walkthrough, state-transition diagram — implemented and verified, not just planned | APPLIED | Touching `runtime.go`, `daemon.go`, or `ExecutionRepository`; or reasoning about execution-lifecycle correctness after a crash/restart. |
| [`fixed-runtime-resilience.md`](fixed-runtime-resilience.md) | — | Implementation record for `gap-runtime-resilience.md`: `dispatchBounded` (panic recovery + semaphore-bounded concurrency) replacing the bare per-claim goroutine — implemented and verified, not just planned | APPLIED | Touching `runtime.go`'s `Sweep`/dispatch mechanics, `MaxConcurrentDispatch`, or reasoning about what happens when one dispatch panics. |
| [`fixed-authority-model-formalization.md`](fixed-authority-model-formalization.md) | — | Implementation record for `docs/adr/ADR-0010-authority-model-formalization.md`: real `HUMAN_ONLY`, structural (not opt-in) self-approval/human-decider/expiry checks in `ResolveApproval`, `knowledge.approve` split into `knowledge.review.request` + a generic decide-time check — implemented and verified, not just planned. Resolves this doc's own "Governance-autonomy drift" and "Separation-of-duties is opt-in" rows below. | APPLIED | Touching `governance/evaluate.go`, `ResolveApproval`, `policy.Decision`, or reasoning about who may approve/decide an Approval. |

## Method

Six parallel, code-only investigations (kernel/domain, application, persistence/events/outbox/Redis, governance/identity/auth, runtime/daemon/providers, HTTP/frontend/CI/docs) against the repository as of commit `f11fe21`, each cross-checked against every `APPROVED` document in `architecture/`, `domain/`, `adr/`, `security/`, `testing/`, and `ROADMAP.md`. No code was changed to produce this audit. Every claim carries a `file:line` citation from that pass; nothing here is inferred from documentation alone.

## Reconciliation (2026-08-24)

Approved after a second, targeted pass against the full original audit brief — not a rubber-stamp. Gaps found and closed before approval, per this repository's "resolve before approving" convention:

- The Company → Mission → Governance → Objectives → Departments → Workflows → Agents/Teams → Capabilities → Results → Evaluation chain had never been checked component-by-component. Doing so surfaced two real findings folded into `findings.md` §1.0: **Mission has zero footprint anywhere in the repository**, and **Agents/Teams has no governed implementation** — `docs/domain/agent.md` exists but no `internal/domain/agent` package does; a separate, previously-unaudited `internal/agent` package was found (self-disclaimed as not the governed Agent, unwired from `main.go`, confirmed via `grep` in this pass).
- Added an explicit **authoritative-vs-disposable state** inventory (`findings.md` §5) — items 8-9 of the original brief's per-subsystem checklist, previously only addressed narratively.
- Added an explicit **known-flaky-tests** table (`findings.md` §6), cross-referencing `current-state.md`'s own session notes against the actual test files — the brief names flaky tests as a specific attention area and this had not been written up as a standalone finding.
- Added an explicit **orchestration-framework (LangGraph) verdict** (`findings.md` §7) — the brief explicitly raises this question and it had gone unanswered by omission rather than by stated conclusion.
- Added an explicit **recommended implementation order** (`findings.md` §8) — the brief's section E existed only implicitly via severity ranking before this pass.
- Verified every `gap-*.md`'s file:line citations still resolve to real files/lines and every internal link target exists.

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [`ROADMAP.md`](../../ROADMAP.md)
- [Testing strategy](../testing/strategy.md) — already names the CI/DB gap [`gap-ci-integration-coverage.md`](gap-ci-integration-coverage.md) tracks, as Phase 8 scope.
