# Current State

- **Last updated:** 2026-08-19
- **Current phase:** Architecture audit correction
- **Architecture approver:** Project owner (`Node-Features`)
- **Next task:** Correct Approval persistence ownership and canonicalize the shared command/proposal contract, then run a fresh read-only architecture audit

## Approved material

None.

## Draft material

- `ARCHITECTURE.md` and all completed `docs/architecture/*.md` documents are drafts awaiting final audit and explicit owner approval.
- Application orchestration now uses two-stage Kernel validation, exact Governance evaluation, atomic pending approval, and post-commit Runtime notification.
- Organization, Identity, Principal, Agent, Capability, Evaluation, Metric, Resource, Event, Evidence, Artifact, Objective, Workflow, Result, and Knowledge contracts are drafts.
- The Events/Persistence/Knowledge dependency direction and adaptive-department boundaries have been normalized.
- Workflow is the first aggregate; `StartWorkflow` and the pending-approval, accepted-transition, and Runtime execution transaction boundaries are specified as drafts.
- `docs/adr/ADR-0003-model-independent-intelligence.md` is proposed and not accepted.
- Research, Monitoring & Evaluation, and Finance department documents are drafts awaiting review.
- `AGENTS.md`, `docs/INDEX.md`, `.companyos/agent-memory/`, and directory indexes await approval.
- `docs/references/feature-provenance.md` is proposed pending source analysis and architecture review.

## Blockers

- Architecture documents cannot be approved until applicable `CRITICAL` and `MAJOR` audit findings are resolved.
- `docs/domain/approval.md` still assigns persistence directly to Governance instead of Application coordination.
- The command/proposal envelope is specified in Application architecture, while Governance still awaits a canonical shared domain contract.
- A fresh read-only architecture audit is required after the remaining corrections before document approval or implementation.

## Unresolved implementation choices

- The initial Runtime implementation or adapter is not selected.
- The minimal post-commit notification recovery mechanism is not selected.
- The persistence adapter and state-plus-outbox versus event-sourcing choice are not selected.
- The first concrete CapabilityDefinition satisfying `StartWorkflow` is not selected.

## Open questions

- OPEN QUESTION: What license terms govern reuse from `vierisid/jarvis`?
- OPEN QUESTION: Should existing workflow, department, security, and testing directories remain as justified extensions to the core layout?
- OPEN QUESTION: When should obsolete predecessor files be deleted after migration review?
