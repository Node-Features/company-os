# Fixed: CI Runs the Real Database-Backed Suite

Status: IMPLEMENTED (2026-08-24) — implemented and verified locally against every underlying
command CI now runs; **not yet verified against a real GitHub Actions execution**, which this
session's tools cannot trigger — see "Known limitations" below.

This is the implementation record for [`gap-ci-integration-coverage.md`](gap-ci-integration-coverage.md)
(P1), in the same shape as [`fixed-execution-lifecycle-hardening.md`](fixed-execution-lifecycle-hardening.md)
and [`fixed-runtime-resilience.md`](fixed-runtime-resilience.md): what actually shipped, what it
proves, and what's still open — not just the plan.

## What was implemented

`.github/workflows/ci.yml` rewritten from 2 jobs (`companyd`: `go vet`+`go test`, no
`DATABASE_URL`; `web`: `npm install`+`npm run build`) to 9 jobs across two tiers, plus a new
`.github/workflows/nightly.yml`. Full layering, job list, and category mapping recorded in
[`docs/testing/strategy.md`](../testing/strategy.md)'s "CI gates" section — not repeated here.

Two previously unnoticed, unrelated bugs were found and fixed as part of getting this to actually
work, not left for CI's first real run to discover:

1. **Stale Go version pin**: `ci.yml` pinned `go-version: "1.22"`; `go.mod` requires `go 1.25.0`.
   CI only "worked" via silent automatic-toolchain-download. Fixed with
   `go-version-file: apps/companyd/go.mod` — structurally can't drift again.
2. **`next lint` no longer exists**: Next.js 16 removed the built-in `next lint` CLI entirely
   (confirmed via `node_modules/next/dist/docs/.../03-eslint.md`'s current setup instructions,
   which now say to configure the ESLint CLI directly). `apps/web/package.json`'s `"lint": "next
   lint"` script — and the CI step this work was about to add around it — would have failed
   immediately. Fixed: added `apps/web/eslint.config.mjs` (flat config, `eslint-config-next`'s
   `core-web-vitals`+`typescript` presets), installed `eslint`+`eslint-config-next` as
   devDependencies, changed the script to `"lint": "eslint ."`.

## Evidence

Every command below was run for real, locally, this session — not inferred from reading code.

### Go side

| Check | Command | Result |
|---|---|---|
| Formatting | `gofmt -l .` (from `apps/companyd`) | Zero output (clean) |
| Static analysis | `go vet ./...` | Zero output (clean) |
| Build | `go build ./...` | Zero output (clean) |
| Linter (new) | `golangci-lint run --timeout=5m` (v2.13.1, installed via `go install`, no `.golangci.yml` — v2's curated default set) | **14 pre-existing findings** (7 errcheck, 4 staticcheck, 3 unused) — see "Known limitations" |
| Workflow syntax | `actionlint` (installed via `go install github.com/rhysd/actionlint/cmd/actionlint@latest`) against both `.yml` files | Exit 0, zero findings — real schema/expression/shellcheck-style validation, not just a YAML parse |
| Full real-DB suite | `go test ./internal/application/... ./internal/runtime/... ./internal/adapters/... -count=1 -v` (remote Supabase Postgres, this session's earlier concurrency work) | `ok` across all 5 packages, exit 0 — the exact suite `go-integration`/the specialized lanes now run in CI, already proven to pass |

### Web side

| Check | Command | Result |
|---|---|---|
| Dependency install (real, fresh) | `rm -rf node_modules && npm ci` | Succeeds cleanly from the committed lockfile — confirms `package-lock.json` is CI-consistent, not just locally-resolved |
| Lint | `CI=true npm run lint` (→ `eslint .`) | Exit 0, zero findings |
| Type-check | `npx tsc --noEmit` | Exit 0, zero findings |
| Build | `CI=true NEXT_TELEMETRY_DISABLED=1 npm run build` | Exit 0 — `Compiled successfully in 59s`, all 9 routes built |

### golangci-lint findings detail, and why they don't block

The 14 findings (`errcheck`: unchecked `defer tx.Rollback(ctx)` in 3 of 9 identical call sites and
unchecked `json.NewEncoder(w).Encode(...)` in 3 of 12 call sites — golangci-lint's default
`max-same-issues: 3` caps repeated-pattern output, so the true count across the codebase is higher
than 14; `staticcheck`: 3 minor style suggestions; `unused`: 3 symbols) are real and pre-existing —
this codebase has never run a linter beyond `go vet` before. One is worth a specific look, not just
a style nit: **`internal/kernel/workflow/transitions.go`'s `requiredPriorState`, `nextState`, and
`statePtr` are entirely unreferenced**, despite the file's own doc comment claiming the table is
"shared by proposal.go and decision.go for every command" — a real doc/code mismatch, left
uninvestigated here (deleting Kernel legality code without understanding why it went unused is out
of scope for a CI task) and flagged as a follow-up.

`ci.yml`'s `golangci-lint` step sets `only-new-issues: true` (the action's documented,
GitHub-API-diff-based mechanism, not a config trick) so these pre-existing findings don't block
every future unrelated PR on day one — a new issue introduced by a PR's own diff still fails that
PR. This is the standard way to adopt a new linter into a codebase that's never run one, not a way
of hiding failures: nothing here is `continue-on-error`, no test is skipped, no assertion loosened.

## Known limitations

- **Not verified against a real GitHub Actions execution.** `actionlint` validates syntax/schema
  and every underlying shell command was run locally and passed, but service-container health-check
  timing, job-dependency sequencing, and GitHub-hosted-runner specifics (Docker version, `-race`
  toolchain availability) can only be fully confirmed by an actual PR triggering this workflow —
  this session has no such access. First real run should be watched, not assumed green.
- **~14+ pre-existing golangci-lint findings**, real baseline debt, tracked above and via
  `only-new-issues: true` — not fixed in this pass, not hidden either.
- **`eslint` pinned to `9.39.5`**, not the current `10.x` line: `eslint-config-next@16.3.2`'s
  bundled `eslint-plugin-react@7.37.5` throws (`contextOrFilename.getFilename is not a function`)
  under ESLint 10's flat-config context API — reproduced directly, not assumed. A live upstream
  compatibility gap; revisit when `eslint-config-next` fixes it.
- **No port-conformance suite** (`docs/testing/contract-tests.md`'s full per-port design) — the new
  `contract-tests` CI job runs today's real `internal/adapters/httpapi` tests as the closest actual
  coverage; the fuller suite is separate, not-yet-authorized work (confirmed with the project owner
  before implementing, not assumed).
- **No frontend test framework** — `web-fast` covers lint/typecheck/build only (confirmed scope
  choice); `web` has no Vitest/Playwright/Jest.
- **`verifier_live_test.go` stays skipped in CI**, deliberately — no `SUPABASE_URL` is set, so the
  one test gated on it (a live external JWKS fetch) doesn't become a non-deterministic CI dependency.

## Dependencies

- [`gap-ci-integration-coverage.md`](gap-ci-integration-coverage.md) — the original problem statement this implements
- [`docs/testing/strategy.md`](../testing/strategy.md) — the canonical current CI shape, updated alongside this
- [`docs/testing/concurrency-guarantees.md`](../testing/concurrency-guarantees.md) — the "named-pattern test selection" rationale the `governance-security`/`concurrency`/`failure-recovery` lanes use
- [`findings.md`](findings.md) §1 ("CI" row, now closed)
- `ROADMAP.md` Phase 8 Slices 1 (done) and 2 (partial — job slot only)
