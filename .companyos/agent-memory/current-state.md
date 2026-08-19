# Current State

- **Last updated:** 2026-08-19
- **Current phase:** Architecture audit correction
- **Architecture approver:** Project owner (`Node-Features`)
- **Next task:** Define the Application Use Case boundary and resolve the remaining major architecture-audit findings

## Approved material

None.

## Draft material

- `ARCHITECTURE.md` and all completed `docs/architecture/*.md` documents are drafts awaiting correction and review.
- `docs/adr/ADR-0003-model-independent-intelligence.md` is proposed and not accepted.
- Research, Monitoring & Evaluation, and Finance department documents are drafts awaiting correction and review.

- `AGENTS.md` — working agent entry point, awaiting approval.
- `docs/INDEX.md` — context-routing map, awaiting approval.
- `.companyos/agent-memory/` — operational summaries, awaiting approval.
- Existing documentation directory indexes — awaiting review and migration decisions.
- `docs/references/feature-provenance.md` — proposed borrowing map, awaiting source analysis and architecture review.

## Blockers

- Architecture documents cannot be approved until applicable `CRITICAL` and `MAJOR` audit findings are resolved.
- The Application Use Case orchestration boundary is not yet defined.
- Evaluation and performance contracts overlap across M&E, Intelligence, and Coding Agents.
- Finance resource-constraint terminology is not yet canonical.
- Adaptive department documents declare cyclic document dependencies that could imply implementation coupling.
- Trusted Principal identity and authentication authority are not yet defined.
- Adaptive-loop failure semantics and Knowledge approval authority remain incomplete.

## Open questions

- OPEN QUESTION: What license terms govern reuse from `vierisid/jarvis`?
- OPEN QUESTION: Should existing workflow, department, security, and testing directories remain as justified extensions to the core layout?
- OPEN QUESTION: When should obsolete predecessor files be deleted after migration review?
