# Current State

- **Last updated:** 2026-08-21
- **Current phase:** Phase 1 first vertical slice — Slices 0-5 and 7 implemented and verified live end-to-end; Slice 6 (`CANCEL_WORKFLOW`) and Slice 8 (realtime activity layer) remain.
- **Architecture approver:** Project owner (`Node-Features`)
- **Next task:** `ROADMAP.md#phase-1--first-vertical-slice` Slice 6 — `CANCEL_WORKFLOW` (`PLANNED`/`READY` → `CANCELLED`, authorized-Principal only) and Slice 8 — realtime activity layer (replace `WorkflowTrigger`'s polling with Supabase Realtime push, per `ADR-0004` item 4's already-deferred notification mechanism); independent of each other. After that, Phase 1 is complete; `ROADMAP.md` was expanded 2026-08-21 to cover the full path to production (Phases 2-9: security/testing foundations, governed execution, adaptive organization, knowledge, engineering workspaces, remaining departments, CI/CD, production deployment) — Phase 2 (security/testing docs) can start in parallel with either, since none of the three depend on each other.

`CREATE_WORKFLOW → START_WORKFLOW → Runtime dispatch → ACCEPT_WORKFLOW_RESULT/REJECT_WORKFLOW_RESULT` is implemented in `apps/companyd/internal/{kernel,governance,application,runtime,daemon,adapters}` and verified live: real HTTP requests through both `companyd` directly and through `web`'s Next.js routes, against the real hosted Postgres, ending in `COMPLETED` with real generated text from a live provider call. Unit tests cover the Kernel transition table, Governance policy evaluation, and Application pipeline (idempotency, conflict, notify-after-commit); integration tests cover the persistence adapter and the Application layer against the real database.

## Approved material

On 2026-08-20 the project owner explicitly approved, document by document following a guided review against the criteria in `docs/architecture/README.md` and `docs/adr/README.md`:

- `ARCHITECTURE.md`, the canonical top-level document;
- all 14 `docs/architecture/*.md` documents: `system-context.md`, `identity.md`, `kernel.md`, `application.md`, `runtime.md`, `daemon.md`, `departments.md`, `intelligence.md`, `coding-agents.md`, `workspaces.md`, `governance.md`, `events.md`, `knowledge.md`, `persistence.md`;
- all 20 `docs/domain/*.md` documents: `organization.md`, `objective.md`, `department.md`, `workflow.md`, `execution.md`, `agent.md`, `capability.md`, `command.md`, `principal.md`, `policy.md`, `approval.md`, `artifact.md`, `evidence.md`, `result.md`, `metric.md`, `evaluation.md`, `resource.md`, `workspace.md`, `event.md`, `knowledge.md`;
- `docs/adr/ADR-0001-kernel-runtime-daemon.md`, `docs/adr/ADR-0002-pluggable-departments.md`, and `docs/adr/ADR-0003-model-independent-intelligence.md` — all three foundational ADRs.

Gaps flagged during review were resolved before approval rather than approved with an asterisk: `docs/domain/workflow.md` now defines `FAILED` and `CANCELLED` terminal states, the `REJECT_WORKFLOW_RESULT` and `CANCEL_WORKFLOW` commands, and explicit `INDETERMINATE`/`PARTIAL` handling (previously only the successful path was specified), rippling into `command.md`, `result.md`, `application.md`, `persistence.md`, and `runtime.md`; `ADR-0003` gained a provider-substitution test plan; `ARCHITECTURE.md` gained an explicit mention of the `execution.md` contract under Runtime and had its stale "remains a draft" framing corrected; `ADR-0001` and `ADR-0002` had their acceptance criteria updated to note their underlying documents are now themselves `APPROVED`, not merely consistent drafts.

On 2026-08-21 the project owner additionally approved `docs/adr/ADR-0004-first-slice-technology-stack.md`, fixing deployment topology (`companyd` as one Go process; Next.js as UI plus HTTP Application adapter), the Supabase/Postgres state-plus-outbox persistence adapter, an internal Go Runtime, Supabase Realtime plus polling-sweep notification recovery, the first `CapabilityDefinition` (minimal `IntelligenceCapability`), and Supabase Auth as the Identity adapter. Its four sub-questions (below) were left open as implementation detail, mirroring how `ADR-0001` was accepted with its own open questions still unresolved.

## Remaining draft or proposed material

- Research, Monitoring & Evaluation, and Finance department documents are drafts awaiting review.
- `AGENTS.md`, `docs/INDEX.md`, `.companyos/agent-memory/`, and directory indexes await approval.
- `docs/references/feature-provenance.md` is proposed pending source analysis and architecture review.

## Blockers

None currently open. Architecture and domain document approval was gated on applicable `CRITICAL`/`MAJOR` audit findings being resolved; the 2026-08-20 audit found none. This blocker re-activates only if a future audit finds one.

## Decided implementation choices

All four are now settled by the approved `docs/adr/ADR-0004-first-slice-technology-stack.md` (2026-08-21):

- Runtime implementation: internal, in Go, inside `companyd`.
- Persistence adapter: Supabase/Postgres, state-plus-outbox (not event sourcing).
- Notification recovery: Supabase Realtime (`LISTEN`/`NOTIFY`) direct path plus a Runtime polling sweep fallback; no message broker.
- First `CapabilityDefinition`: a minimal `IntelligenceCapability` for short bounded text generation, one `ModelProfile`/`ProviderAdapter`.

`companyd` hosting is decided: local development runs `companyd` via `air` (hot-reload) against Supabase; production runs `companyd` on a DigitalOcean VPS supervised by systemd, satisfying `daemon.md`'s external-supervision expectation. `ADR-0004`'s four sub-questions were left open at approval time as implementation detail, not a blocker — the Phase 1 implementation has since answered them in code (the ADR text itself is unchanged, per its immutability rule):

- LLM provider/model: not one fixed choice — `internal/adapters/intelligence/fallback` composes Gemini (`gemini-2.5-flash`), OpenAI (`gpt-5-nano`), and Anthropic (`claude-haiku-4-5`) behind one `ProviderAdapter`, trying them in priority order and skipping one that just rate-limited or went down (60s cooldown) rather than failing the dispatch.
- Transport: plain REST/JSON on `companyd`'s `net/http` mux (Go 1.22+ method+pattern routing); `web`'s Next.js Route Handlers proxy to it via `fetch`.
- Supabase RLS: enabled with zero policies on every table (default-deny to the `anon`/`authenticated` PostgREST roles `web`'s publishable key uses); `companyd`'s pooled `DATABASE_URL` role is the trusted backend and bypasses RLS. Organization isolation is enforced at the Application/Kernel layer, not the DB layer, this slice.
- Retry policy: 3 attempts (1 + 2 retries), exponential backoff base 2s capped at 30s, full jitter — fields on the `bounded-text-generation` `CapabilityDefinition` fixture, not hardcoded in Runtime.

## Open questions

- OPEN QUESTION: What license terms govern reuse from `vierisid/jarvis`?
- OPEN QUESTION: Should existing workflow, department, security, and testing directories remain as justified extensions to the core layout?
- OPEN QUESTION: When should obsolete predecessor files be deleted after migration review?
