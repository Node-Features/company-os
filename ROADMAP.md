# CompanyOS Roadmap

Status: DRAFT

This expands the phase summary in [README.md — Roadmap](README.md#roadmap). It tracks intended sequencing, not committed dates. CompanyOS has no implementation yet, so no later phase begins in earnest before Phase 0 is complete. Live blockers and next tasks are tracked in [.companyos/agent-memory/current-state.md](.companyos/agent-memory/current-state.md), which is authoritative for current status; this file is not.

## Phase 0 — Foundation (current phase)

- [x] Define project direction and core principles (`README.md`)
- [x] Establish documentation hierarchy (`AGENTS.md`, `docs/INDEX.md`)
- [x] Establish open-source provenance process (`docs/references/`)
- [ ] Complete architecture reconciliation across `docs/architecture/` and `docs/domain/`
- [ ] Resolve remaining ownership boundaries and open blockers (see `current-state.md`)
- [ ] Accept foundational ADRs, starting with `ADR-0001` (Kernel/Runtime/Daemon separation) and `ADR-0002` (pluggable departments)
- [ ] Complete a fresh read-only architecture audit
- [ ] Project owner (`Node-Features`) explicitly approves architecture and domain documents

## Phase 1 — First vertical slice

Grounded in the [Application layer's first vertical slice](docs/architecture/application.md#first-vertical-slice) and the [Workflow first-slice lifecycle](docs/domain/workflow.md#first-slice-lifecycle).

- [ ] Select the first concrete Organization, Objective, and WorkflowDefinition
- [ ] Select the first CapabilityDefinition satisfying `START_WORKFLOW` (open question — see [Capability domain](docs/domain/capability.md#open-questions))
- [ ] Implement `CREATE_WORKFLOW`, `START_WORKFLOW`, and `ACCEPT_WORKFLOW_RESULT` through the Application/Kernel/Governance/Persistence sequence
- [ ] Implement the minimum Runtime execution-state contract (currently a tracked blocker)
- [ ] Select the first Runtime implementation or adapter (open question — see [Runtime](docs/architecture/runtime.md#open-questions))
- [ ] Select the first persistence adapter (open question — see [Persistence](docs/architecture/persistence.md#open-questions))

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
