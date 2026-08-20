# CompanyOS Roadmap

Status: DRAFT

This expands the phase summary in [README.md — Roadmap](README.md#roadmap). It tracks intended sequencing, not committed dates. Phase 0 is complete; Phase 1 has a connected scaffold but no domain logic yet (see `README.md#project-status`). Live blockers and next tasks are tracked in [.companyos/agent-memory/current-state.md](.companyos/agent-memory/current-state.md), which is authoritative for current status; this file is not.

## Phase 0 — Foundation (complete)

- [x] Define project direction and core principles (`README.md`)
- [x] Establish documentation hierarchy (`AGENTS.md`, `docs/INDEX.md`)
- [x] Establish open-source provenance process (`docs/references/`)
- [x] Complete architecture reconciliation across `docs/architecture/` and `docs/domain/`
- [x] Resolve remaining ownership boundaries and open blockers (see `current-state.md`)
- [x] Accept foundational ADRs — `ADR-0001` (Kernel/Runtime/Daemon separation), `ADR-0002` (pluggable departments), `ADR-0003` (model-independent intelligence)
- [x] Complete a fresh read-only architecture audit
- [x] Project owner (`Node-Features`) explicitly approves architecture and domain documents (2026-08-20; see `current-state.md`)

## Phase 1 — First vertical slice

Grounded in the [Application layer's first vertical slice](docs/architecture/application.md#first-vertical-slice) and the [Workflow first-slice lifecycle](docs/domain/workflow.md#first-slice-lifecycle). `ADR-0004` proposes answers to this phase's three long-standing open questions (Runtime, persistence adapter, first CapabilityDefinition) but is not yet owner-approved — approving it is slice 0 below.

A connected `companyd` (Go) + `web` (Next.js) + Supabase scaffold already exists and builds cleanly; none of it contains domain logic yet (see `README.md#project-status`). The slices below are ordered so each one is independently buildable and testable — no slice depends on a later one.

- [ ] **Slice 0 — Formalize the stack decision.** Flip `ADR-0004` `Status: PROPOSED` → `APPROVED` (the scaffold already matches it) and resolve its remaining open sub-questions: first LLM provider, Next.js↔`companyd` transport, Supabase RLS design, first Retry policy defaults.
- [ ] **Slice 1 — Choose the first fixtures.** Decide the first concrete Organization, Objective, and WorkflowDefinition (open question below), and confirm the first CapabilityDefinition = the minimal `IntelligenceCapability` from `ADR-0004`. These can start as hardcoded Go values — no schema needed yet.
- [ ] **Slice 2 — Schema + persistence adapter.** Write the first Supabase migration (Workflow, ExecutionIntent, Result, DomainEvent at minimum) and implement `AuthoritativeStateRepository` for the Workflow aggregate in `apps/companyd/internal/adapters/persistence/supabase/`, committing state and DomainEvents atomically (state-plus-outbox, per `ADR-0004`).
- [ ] **Slice 3 — `CREATE_WORKFLOW`.** Kernel proposal + legality validation (absent → `PLANNED`), a real (if trivial) Governance `ALLOW`/`DENY`/`REQUIRE_APPROVAL` decision point — not bypassed — and Application sequencing. Prove a failed precondition produces a stable rejection reason with no transition or event.
- [ ] **Slice 4 — `START_WORKFLOW` + Runtime.** `PLANNED` → `READY` plus exactly one `ExecutionIntent`; the first internal Go Runtime dispatches it to the first `ProviderAdapter`; Runtime submits a Result back through Application only (never mutates Workflow state directly).
- [ ] **Slice 5 — Result acceptance.** `ACCEPT_WORKFLOW_RESULT` (`READY` → `COMPLETED` on `SUCCEEDED`) and `REJECT_WORKFLOW_RESULT` (`READY` → `FAILED` on `FAILED`/`TIMED_OUT`/`PARTIAL`); `INDETERMINATE` causes no transition.
- [ ] **Slice 6 — `CANCEL_WORKFLOW`.** `PLANNED` or `READY` → `CANCELLED`, authorized-Principal only.
- [ ] **Slice 7 — Expose it through `web`.** A `companyd` endpoint accepting a `WorkflowCommandEnvelope`, and a minimal Next.js page that triggers `CREATE_WORKFLOW`/`START_WORKFLOW` and displays the resulting state — `web` calls `companyd` for this, it never writes Supabase directly (per the guardrail already in `apps/web/lib/supabase/*.ts`).

## Phase 2 — Governed execution

- [ ] Exercise the full Governance decision pipeline for the first slice, including `REQUIRE_APPROVAL` and `DENY` paths end to end
- [ ] Implement Identity's authentication evidence flow into Governance
- [ ] Wire dispatch-time re-evaluation immediately before governed external effects

## Phase 3 — Adaptive organization

- [ ] Stand up Research, Monitoring & Evaluation, and Finance against the shared [adaptive-loop contracts](docs/architecture/departments.md#adaptive-feedback-loop)
- [ ] Implement the Objective-creation gate from a Finding, Recommendation, or Evaluation

## Beyond Phase 3

- Provider-independent Intelligence routing ([ADR-0003](docs/adr/ADR-0003-model-independent-intelligence.md), currently `PROPOSED`)
- `CodingAgentRuntime` and `Workspaces` for engineering execution

## Open questions

- Which concrete Organization, Objective, and WorkflowDefinition prove the first vertical slice?
- Which Runtime implementation/adapter, persistence adapter, and first CapabilityDefinition are selected?
- Which components run in one process versus separate deployment boundaries for the first Daemon? (see [Daemon — Open questions](docs/architecture/daemon.md#open-questions))

## Dependencies

- [README.md — Roadmap](README.md#roadmap) (summary shown to readers first)
- [.companyos/agent-memory/current-state.md](.companyos/agent-memory/current-state.md) (live status; authoritative over this file)
- [docs/adr/README.md](docs/adr/README.md)
