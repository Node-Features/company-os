# Fixed: Kernel Boundary Leak (`internal/fixtures` → `internal/ports`)

Status: IMPLEMENTED (2026-08-25) — verified via `go build`, `go vet`, and the targeted package tests below; a whole-module `go test ./...` was still running against this environment's `.env`-configured `DATABASE_URL`/`SUPABASE_URL` past the point this doc was written and is not itself part of this record's evidence. **Not yet committed to git** as of this writing — check `git status` before assuming this is landed on a branch.

This is a standalone found-and-fixed record, not an implementation of a pre-existing `gap-*.md` — the underlying audit ("inspect the actual [Kernel] dependency graph") was requested directly by the project owner on 2026-08-25, scoped to `internal/kernel` specifically, and is narrower than the full six-track audit `findings.md` (2026-08-24) already ran. It also **corrects** a claim in that approved doc: `findings.md` §1.1's Kernel row says "Holds, no exceptions found." That verdict was reached by reading Kernel's own code (true — Kernel's functions are pure, deterministic, no I/O) without computing its *compiled* transitive dependency graph, which was not clean. See the `findings.md` edit under Dependencies below.

## What was found

**1. Misplaced responsibility.** [`internal/kernel/workflow`](../../apps/companyd/internal/kernel/workflow/proposal.go) took `fixtures.Registry` as a parameter in `ValidateCreateProposal` ([proposal.go:37](../../apps/companyd/internal/kernel/workflow/proposal.go#L37)), `ValidateStartProposal` ([proposal.go:63](../../apps/companyd/internal/kernel/workflow/proposal.go#L63)), and `FinalizeStart` ([decision.go:69](../../apps/companyd/internal/kernel/workflow/decision.go#L69)). `internal/fixtures` ([firstslice.go](../../apps/companyd/internal/fixtures/firstslice.go), pre-fix) bundled two unrelated things in one package: pure hardcoded lookup data (`Registry.Organization()`/`.Objective()`/`.WorkflowDefinition()`/`.Capability()` — the part Kernel actually needs) and `NewRegistryFromDB(ctx context.Context, orgRepo ports.OrganizationRepository) (Registry, error)` — a boot-time database loader called exactly once, from `cmd/companyd/main.go`.

**2. Why it violates the boundary.** [`docs/architecture/kernel.md`](../architecture/kernel.md)'s Non-responsibilities section lists "database drivers, transactions,... event brokers, or workflow-engine SDKs," "model inference,... provider APIs," and "UI, transport protocols, notifications, metrics exporters" as things the Kernel does not own, and states the Kernel "cannot depend on their concrete implementations." Because Go's unit of compilation is the package, Kernel importing `fixtures` for its lookup accessors forced it to also compile `fixtures`'s `context` and `internal/ports` imports — and `internal/ports` is one undivided package ([`ls apps/companyd/internal/ports/*.go`](../../apps/companyd/internal/ports)) declaring `ProviderAdapter` (model calls/provider routing, [provideradapter.go](../../apps/companyd/internal/ports/provideradapter.go)), `Authenticator`, `Notifier`, `MetricsRecorder`, the outbox/publisher ports, and every persistence repository (Workflow, Finance, M&E, Research, Knowledge, Objective, Identity). Measured with `go list -deps ./internal/kernel/workflow` before the fix, Kernel's transitive closure included `internal/ports` and, through it, `domain/finance`, `domain/monitoringevaluation`, `domain/research`, `domain/execution`, and `domain/approval` — none of which any Kernel workflow function reads or returns. No Kernel code actually *called* any of those interfaces; the violation was structural (compiled dependency graph), not behavioral (a rogue function call) — which is exactly why `findings.md`'s code-reading pass didn't catch it.

**3. Where it now lives.** The database read moved to [`cmd/companyd/main.go`](../../apps/companyd/cmd/companyd/main.go#L166-L184) (Daemon boot stage 4c), which is the composition root and already imports every adapter package. `internal/fixtures` gained `Registry.WithOrganization(org organization.Organization) Registry` ([firstslice.go:74-77](../../apps/companyd/internal/fixtures/firstslice.go#L74-L77)) — a pure value-copy method — and lost `NewRegistryFromDB`, its `context` import, and its `internal/ports` import entirely.

**4. Dependency direction now enforced.**
```
internal/kernel/workflow  →  internal/fixtures  →  internal/domain/*
cmd/companyd/main.go (Daemon)  →  internal/adapters/persistence/supabase, internal/fixtures, internal/application, internal/runtime
```
Infra flows into the composition root; it never flows into Kernel. Measured after the fix:
```
$ go list -f '{{join .Deps "\n"}}' ./internal/kernel/workflow | grep company-os/apps/companyd/internal
.../domain/capability   .../domain/command   .../domain/event   .../domain/objective
.../domain/organization .../domain/policy    .../domain/principal .../domain/result
.../domain/workflow     .../fixtures

$ go list -f '{{join .Deps "\n"}}' ./internal/fixtures | grep company-os/apps/companyd/internal
.../domain/capability .../domain/command .../domain/event .../domain/objective
.../domain/organization .../domain/principal .../domain/workflow
```
`internal/ports` no longer appears in either closure.

**5. Tests that prove it.** New file [`internal/kernel/kernel_boundary_test.go`](../../apps/companyd/internal/kernel/kernel_boundary_test.go). `TestKernelPackagesDoNotImportInfrastructure` runs `go list ./internal/kernel/...` to discover every current Kernel package (not a hardcoded list — a new `internal/kernel/<x>` package is covered automatically the day it's added), then `go list -deps` on each and fails on any same-module import outside `internal/domain/*` / `internal/fixtures` / `internal/kernel/*`. Verified this is load-bearing, not decorative: reverted the fix with `git stash push -- internal/fixtures/firstslice.go cmd/companyd/main.go`, re-ran the test, and it failed with `internal/kernel/workflow transitively imports .../internal/ports, which is outside the Kernel's allowed boundary`; `git stash pop` restored the fix and the test passed again.

## What was checked and found already correct (not touched)

- `internal/kernel/objective`, `internal/kernel/knowledge` — no `fixtures` dependency, no infra imports of any kind.
- [`decision.go`](../../apps/companyd/internal/kernel/workflow/decision.go)'s `Finalize*` functions only *verify* a caller-supplied `policy.GovernanceDecision` via `verifyAllow` ([decision.go:218-226](../../apps/companyd/internal/kernel/workflow/decision.go#L218-L226)); they don't compute authorization. `internal/governance` independently imports only `domain/policy` and `domain/principal` — confirmed via `go list -f '{{join .Imports "\n"}}' ./internal/governance/...`.
- [`transitions.go`](../../apps/companyd/internal/kernel/workflow/transitions.go) — a plain state-transition table, zero non-domain imports.
- `grep`'d all of `internal/kernel` for `time.Now()`, `os.Getenv`, `net/http`, `context.Context`: zero hits in production code (only in `_test.go` files constructing `DeclaredTime` input fixtures) — matches `kernel.md`'s "never reads wall-clock time, network state, environment variables... implicitly" invariant, which already held.

Deliberately not done, as cosmetic relative to the stated rule (Kernel doesn't own model calls/provider routing/scheduling/HTTP/Redis/Supabase/auth/retries/infra):
- Renaming `internal/fixtures` (its name reads like a test-only package despite being a real production Kernel dependency) — a legibility improvement, not a boundary fix.
- Splitting the monolithic `internal/ports` package by concern (persistence vs. provider vs. auth vs. notify) — would shrink blast-radius for other consumers, but Kernel no longer depends on `internal/ports` at all after this fix, so it isn't required to satisfy the invariant this audit was scoped to.

## Verified

- `go build ./...` — clean.
- `go vet ./...` — clean.
- `gofmt -l` on every changed file — zero output.
- `go test ./internal/kernel/... ./internal/fixtures/...` — pass (`internal/kernel`, `internal/kernel/knowledge`, `internal/kernel/objective`, `internal/kernel/workflow` all `ok`; `internal/fixtures` has no test files of its own).
- Regression proof: `git stash` the fix → `TestKernelPackagesDoNotImportInfrastructure` fails with the exact expected message → `git stash pop` → passes again (see finding 5 above).
- Whole-module `go test ./...`: not completed as evidence for this doc — see the Status line. Re-run it and fold the result in here (or file a new note) before treating this as fully closed the way `fixed-runtime-resilience.md` §Verified did with its three-consecutive-run standard.

## Dependencies

- [`architecture/kernel.md`](../architecture/kernel.md) — the Non-responsibilities contract this fix restores
- [`findings.md`](findings.md) §1.1 Kernel row — corrected alongside this doc: the "Holds, no exceptions found" verdict covered Kernel's own logic, not its compiled dependency graph
- [`README.md`](README.md) — indexed there
