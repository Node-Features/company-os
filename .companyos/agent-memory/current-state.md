# Current State

- **Last updated:** 2026-08-20
- **Current phase:** All architecture and domain documents approved; first-slice technology stack drafted as `ADR-0004`, pending owner approval
- **Architecture approver:** Project owner (`Node-Features`)
- **Next task:** `ROADMAP.md#phase-1--first-vertical-slice` now has an ordered, buildable checklist (Slices 0-7). Slice 0 is approving `ADR-0004` (Go `companyd` for Kernel/Application/Governance/Identity/Runtime/Daemon; Next.js for UI and the Application adapter; Supabase/Postgres for persistence, state-plus-outbox, and notification) — the scaffold under `apps/` and `supabase/` already matches its proposal and builds cleanly, with `web` → Supabase → `companyd` → Postgres connectivity verified end-to-end, but contains no domain logic yet.

## Approved material

On 2026-08-20 the project owner explicitly approved, document by document following a guided review against the criteria in `docs/architecture/README.md` and `docs/adr/README.md`:

- `ARCHITECTURE.md`, the canonical top-level document;
- all 14 `docs/architecture/*.md` documents: `system-context.md`, `identity.md`, `kernel.md`, `application.md`, `runtime.md`, `daemon.md`, `departments.md`, `intelligence.md`, `coding-agents.md`, `workspaces.md`, `governance.md`, `events.md`, `knowledge.md`, `persistence.md`;
- all 20 `docs/domain/*.md` documents: `organization.md`, `objective.md`, `department.md`, `workflow.md`, `execution.md`, `agent.md`, `capability.md`, `command.md`, `principal.md`, `policy.md`, `approval.md`, `artifact.md`, `evidence.md`, `result.md`, `metric.md`, `evaluation.md`, `resource.md`, `workspace.md`, `event.md`, `knowledge.md`;
- `docs/adr/ADR-0001-kernel-runtime-daemon.md`, `docs/adr/ADR-0002-pluggable-departments.md`, and `docs/adr/ADR-0003-model-independent-intelligence.md` — all three foundational ADRs.

Gaps flagged during review were resolved before approval rather than approved with an asterisk: `docs/domain/workflow.md` now defines `FAILED` and `CANCELLED` terminal states, the `REJECT_WORKFLOW_RESULT` and `CANCEL_WORKFLOW` commands, and explicit `INDETERMINATE`/`PARTIAL` handling (previously only the successful path was specified), rippling into `command.md`, `result.md`, `application.md`, `persistence.md`, and `runtime.md`; `ADR-0003` gained a provider-substitution test plan; `ARCHITECTURE.md` gained an explicit mention of the `execution.md` contract under Runtime and had its stale "remains a draft" framing corrected; `ADR-0001` and `ADR-0002` had their acceptance criteria updated to note their underlying documents are now themselves `APPROVED`, not merely consistent drafts.

## Remaining draft or proposed material

- Research, Monitoring & Evaluation, and Finance department documents are drafts awaiting review.
- `AGENTS.md`, `docs/INDEX.md`, `.companyos/agent-memory/`, and directory indexes await approval.
- `docs/references/feature-provenance.md` is proposed pending source analysis and architecture review.

## Blockers

None currently open. Architecture and domain document approval was gated on applicable `CRITICAL`/`MAJOR` audit findings being resolved; the 2026-08-20 audit found none. This blocker re-activates only if a future audit finds one.

## Unresolved implementation choices

All four are now proposed in `docs/adr/ADR-0004-first-slice-technology-stack.md`, pending project owner approval:

- Runtime implementation: internal, in Go, inside `companyd`.
- Persistence adapter: Supabase/Postgres, state-plus-outbox (not event sourcing).
- Notification recovery: Supabase Realtime (`LISTEN`/`NOTIFY`) direct path plus a Runtime polling sweep fallback; no message broker.
- First `CapabilityDefinition`: a minimal `IntelligenceCapability` for short bounded text generation, one `ModelProfile`/`ProviderAdapter`.

`companyd` hosting is now decided: local development runs `companyd` via `air` (hot-reload) against Supabase; production runs `companyd` on a DigitalOcean VPS supervised by systemd, satisfying `daemon.md`'s external-supervision expectation. Remaining sub-questions (LLM provider, Next.js↔`companyd` transport, Supabase RLS design, first Retry policy defaults) stay tracked as `ADR-0004`'s open questions.

## Open questions

- OPEN QUESTION: What license terms govern reuse from `vierisid/jarvis`?
- OPEN QUESTION: Should existing workflow, department, security, and testing directories remain as justified extensions to the core layout?
- OPEN QUESTION: When should obsolete predecessor files be deleted after migration review?
