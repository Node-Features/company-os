# Fixed: Test Suite Nondeterminism

Status: APPLIED (2026-08-24) — root-caused, fixed, and empirically verified. **Not yet committed to git** as of this writing (`git status` still shows the files below as modified/untracked in the working tree) — check current repo state before assuming these fixes are landed on a branch.

Three independent root causes were found and fixed by direct empirical investigation (running the suite repeatedly, under deliberate concurrency, with debug instrumentation where needed) rather than by inspection alone. Two were pure test-isolation bugs; the third turned out to be a real, previously-undetected multi-tenancy data-integrity bug in production code, only exposed once the test-isolation bugs were fixed. See [`findings.md`](findings.md) §6 for how this updates the "known flaky tests" record, and [`backlog-p2-p4.md`](backlog-p2-p4.md)'s "multi-tenant isolation is structural-only-by-absence" row for the related "no second tenant exists yet" finding this directly confirms as reachable, not theoretical.

## Root cause A — Load/Commit race gives two legitimate loser outcomes

**Test:** `TestStartWorkflow_ConcurrentSameVersion_OneAcceptedOneConflict` ([pipeline_test.go](../../apps/companyd/internal/application/pipeline_test.go))

**Mechanism:** `StartWorkflow`'s pipeline ([workflow_start.go](../../apps/companyd/internal/application/workflow_start.go)) is Load → Kernel-validate → Governance → `CommitTransition`, with no atomicity between Load and Commit. Two concurrent callers with the same `ExpectedVersion` can interleave two individually-correct ways: (1) both Load before either Commits → the storage layer's compare-and-swap lets one through, the other gets `ports.ErrConflict` → `Outcome=Conflict`; (2) one caller's entire pipeline completes before the other even calls Load → the second caller's Load observes the already-bumped version, so Kernel-level staleness validation (`ValidateStartProposal`, [proposal.go:68-73](../../apps/companyd/internal/kernel/workflow/proposal.go)) rejects it *before* `CommitTransition` is ever reached → `Outcome=Rejected, [ILLEGAL_STATE VERSION_MISMATCH]`. The test asserted only interleaving 1 could happen.

**Empirical measurement:** 51/500 runs failed (10.2%) with `go test -run ... -count=500`, exact symptom `REJECTED (reasons: [ILLEGAL_STATE VERSION_MISMATCH])`.

**Fix:** [fake_repo_test.go](../../apps/companyd/internal/application/fake_repo_test.go) gained `rendezvous`, a one-shot 2-party barrier; `fakeRepo` gained an opt-in `loadGate` field; `LoadWorkflow` now completes its locked read, releases the lock, *then* blocks on the gate before returning to its caller. [pipeline_test.go](../../apps/companyd/internal/application/pipeline_test.go) arms the gate before spawning its two goroutines. This forces every run into interleaving 1 — the specific mechanism the test names — deterministically rather than by scheduler luck.

One implementation pitfall worth recording: an earlier version of this fix placed the gate *before* acquiring the mutex (synchronizing "starting the load" rather than "completing the load"), which did not close the race — one goroutine could still lock, read, unlock, and run all the way through Commit before the other was ever scheduled to acquire the lock. Debug tracing (`println` instrumentation, removed after) was needed to see this precisely; reasoning about the fix in the abstract was not enough.

**Verification:** 0/3000 and 0/1000 failures across two separate confirmation runs (previously 51/500).

## Root cause B — shared fixture organization ID collided with organization-wide `ClaimDueIntents`

**Mechanism:** every real-database test in `internal/application` built its `Application` via `fixtures.NewRegistry()`, which returns the same hardcoded `OrganizationID` for every test, every process, every run ([firstslice.go](../../apps/companyd/internal/fixtures/firstslice.go)). `ClaimDueIntents` claims due `ExecutionIntents` organization-wide, not scoped to a specific workflow or test ([execution_repo.go](../../apps/companyd/internal/adapters/persistence/supabase/execution_repo.go)). The persistence-layer package (`internal/adapters/persistence/supabase`) had already fixed this exact class of bug for itself in a prior session (`testOrgID()`, [integration_test.go:35-46](../../apps/companyd/internal/adapters/persistence/supabase/integration_test.go), whose own doc comment names the identical mechanism) — but `internal/application`'s tests, and the shared `startedReadyWorkflow`/`submitFakeResultWithProvider` helpers every department's test file depends on, never got the equivalent fix.

**Empirical reproduction:** running two `internal/application` process invocations concurrently against the real database (`go test ./internal/application/... & go test ./internal/application/... & wait`) reliably produced `TestIntegration_ProposeObjective_RejectLeavesNoObjective: claimed 2 intents after polling, want exactly 1`. This is the same mechanism [`findings.md`](findings.md) §6 previously attributed (from `current-state.md`'s 2026-08-22 notes) to a different specific test, `TestIntegration_CreateStartReject_FullPipelineToFailed` — same root cause, different test name hit depending on which process happened to lose the race.

**Fix:** [firstslice.go](../../apps/companyd/internal/fixtures/firstslice.go) gained `NewRegistryWithOrganization(orgID)` — the same fixture set as `NewRegistry()`, with all four `OrganizationID` fields overridden (org, objective, trigger principal, approver principal — kept internally consistent rather than partially overridden). `requireRealApp` ([integration_test.go](../../apps/companyd/internal/application/integration_test.go)) now calls it with a fresh `uuid.New()` per test. The 5 call sites that referenced the shared `fixtures.OrganizationID` constant directly were changed to `app.Fixtures.Organization().OrganizationID`. No table referenced by these tests has a foreign key to a real `organizations` row (confirmed via migration grep — only `principals`/`principal_organization_bindings` do), so any `orgID` value is valid.

**Verification:** two independent concurrent-invocation runs after the fix both completed `ok` (554s / 555s) with zero claim-count failures.

## Root cause C — `performance_profiles` primary key was not organization-scoped (a real production bug)

Fixing root cause B immediately and deterministically exposed this — not a concurrency artifact, a genuine latent multi-tenancy bug that had simply never been reachable before, because every test in this project's history had shared one organization.

**Mechanism:** `performance_profiles` was created with `subject_id` alone as `PRIMARY KEY` ([20260823013950_monitoring_evaluation.sql](../../supabase/migrations/20260823013950_monitoring_evaluation.sql)). `UpsertPerformanceProfile` used `ON CONFLICT (subject_id) DO UPDATE SET ...` without `organization_id` in the `SET` list ([monitoringevaluation_repo.go](../../apps/companyd/internal/adapters/persistence/supabase/monitoringevaluation_repo.go)) — so once *any* organization's evaluation wrote a row for a subject, every later organization's evaluation for that same subject silently updated the data fields while `organization_id` stayed frozen at whichever org inserted first. `GetPerformanceProfile` correctly filters by `(organization_id, subject_id)`, so a different organization's read then finds nothing.

**Empirical confirmation this is deterministic, not flaky:** after fixing root cause B, `TestIntegration_MonitoringEvaluation_ResultToPerformanceProfile` failed with `GetPerformanceProfile: ports: not found` in **both** processes of the 2-way concurrent stress test, and again on a third, entirely non-concurrent standalone run — proving the bug is reachable by any two organizations ever using the same subject, not just concurrent ones.

**Fix (production schema + code, not test-only):** new migration [20260824000000_performance_profiles_org_scoped_pk.sql](../../supabase/migrations/20260824000000_performance_profiles_org_scoped_pk.sql) drops the `subject_id`-only primary key and adds `PRIMARY KEY (organization_id, subject_id)`. `UpsertPerformanceProfile`'s conflict target changed to `ON CONFLICT (organization_id, subject_id)`. Applied to the live dev database via a one-off tool (`cmd/applyperformanceprofilepk`, deleted after use — same precedent as every prior slice's one-off migration-apply tools). A test-only workaround (e.g., randomizing the subject too) was deliberately rejected — it would have masked a real data-integrity bug rather than fixing it, and `subjectID` (`fixtures.CapabilityID`) threads into more of the system's invariants (`WorkflowDefinition.RequiredCapabilityID`, Governance dispatch rules) than `OrganizationID` does, making it a riskier thing to randomize than to leave alone.

**Verification:** 5/5 sequential re-runs passed (previously failing deterministically); held clean across two further full concurrent stress runs and one full clean `go test ./...` pass.

## Files changed

| File | Change |
|---|---|
| [`apps/companyd/internal/application/fake_repo_test.go`](../../apps/companyd/internal/application/fake_repo_test.go) | `rendezvous` barrier + `fakeRepo.loadGate` (root cause A) |
| [`apps/companyd/internal/application/pipeline_test.go`](../../apps/companyd/internal/application/pipeline_test.go) | Arms the gate in the affected test (root cause A) |
| [`apps/companyd/internal/fixtures/firstslice.go`](../../apps/companyd/internal/fixtures/firstslice.go) | New `NewRegistryWithOrganization` (root cause B) |
| [`apps/companyd/internal/application/integration_test.go`](../../apps/companyd/internal/application/integration_test.go) | `requireRealApp` + 5 call sites use per-test org (root cause B) |
| [`apps/companyd/internal/adapters/persistence/supabase/monitoringevaluation_repo.go`](../../apps/companyd/internal/adapters/persistence/supabase/monitoringevaluation_repo.go) | `UpsertPerformanceProfile` conflict target (root cause C) |
| [`supabase/migrations/20260824000000_performance_profiles_org_scoped_pk.sql`](../../supabase/migrations/20260824000000_performance_profiles_org_scoped_pk.sql) | New migration, applied live (root cause C) |

No Kernel decision function, Governance rule, or domain type was modified for any of the three fixes.

## Verification summary

| Scenario | Before | After |
|---|---|---|
| `TestStartWorkflow_ConcurrentSameVersion_OneAcceptedOneConflict` × 500 | 51 failures (10.2%) | 0/3000, 0/1000 (two runs) |
| Two concurrent `internal/application` invocations vs. real DB | Reliably reproduced both root cause B and (after B was fixed) root cause C | Two independent runs: both `ok`, ~554s each |
| `TestIntegration_MonitoringEvaluation_ResultToPerformanceProfile` × 5 sequential | Deterministic failure once any prior row existed | 5/5 pass |
| Full `go test ./...` | — | Every package `ok`, exit 0 |

## Race detector

`go test -race` cannot run in this environment: `cgo.exe: exit status 2`, reproduced even with `CC` pointed explicitly at a located `gcc.exe` (`C:\msys64\ucrt64\bin\gcc.exe`) — not a PATH issue. This matches a pre-existing, already-documented limitation from this project's own history (ADR-0007's `SafeState` tests hit the identical gap in this environment). High-repetition concurrent stress runs (500–3000×) substituted for it; sufficient here since all three root causes were logically explainable and fixed at the logic/schema level, not silenced by adding synchronization to satisfy an instrumentation tool.

## Remaining known nondeterminism

- The real Postgres dev database is never reset between runs (`cmd/migrate` is non-idempotent, no applied-migrations tracking — see [`backlog-p2-p4.md`](backlog-p2-p4.md#migration-hygiene)). Not a live bug now that the tests touched here are properly isolated, but any *future* test that reintroduces a shared, unscoped fixture value could reproduce the same class of bug.
- `-race` cannot run in this environment (toolchain gap, pre-existing) — worth running for real in CI once [`gap-ci-integration-coverage.md`](gap-ci-integration-coverage.md) lands, likely on Linux where this gap won't exist.
- Sporadic `wsarecv`/connection-timeout errors against the pooled real Postgres connection are documented repeatedly in `current-state.md` across prior sessions; not reproduced in ~15 real-database runs during this investigation. Consistent with genuine network blips, not a logic bug, but not disprovable either — flagged, not fixed.

## Dependencies

- [`findings.md`](findings.md) §6 (updated to reflect these fixes)
- [`testing/strategy.md`](../testing/strategy.md), [`testing/failure-injection.md`](../testing/failure-injection.md)
- [`gap-ci-integration-coverage.md`](gap-ci-integration-coverage.md) — these fixes have no CI regression protection until that gap closes
- [`backlog-p2-p4.md`](backlog-p2-p4.md) — multi-tenant isolation and migration hygiene rows, both directly touched by root cause C
