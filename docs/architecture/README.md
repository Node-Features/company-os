# Architecture Documentation

This directory owns detailed architectural responsibilities and boundaries; `ARCHITECTURE.md` remains the canonical overview.

## Approval authority

The CompanyOS project owner, `Node-Features`, is the initial architecture approver. Approval must be explicit and attributable; authorship, merge, silence, or elapsed time does not imply approval.

An architecture document moves from `DRAFT` to `APPROVED` only when:

1. its responsibility, non-responsibilities, dependencies, invariants, failure semantics, and open questions have been reviewed;
2. conflicts with `ARCHITECTURE.md`, approved documents, and accepted ADRs have been resolved;
3. applicable links, terminology, diagrams, and implementation claims have been validated;
4. all blocking `CRITICAL` and `MAJOR` audit findings affecting it are resolved;
5. the project owner explicitly records approval by changing its status to `APPROVED` in a dedicated review change.

Approval applies to the reviewed version. A material boundary or invariant change returns the document to `DRAFT` unless an accepted ADR already authorizes that change. Documents may be reviewed together, but each status changes explicitly. No architecture document is approved merely by appearing in this index.

| Document | Purpose | Status | Read when |
|---|---|---|---|
| `system-context.md` | System actors, boundaries, and external relationships | DRAFT | Read when establishing system scope. |
| `identity.md` | Authentication, durable Principal identity, delegation, and trusted claims | DRAFT | Read when establishing actor identity or authentication evidence. |
| `kernel.md` | Kernel ownership and non-responsibilities | DRAFT | Read when changing organizational semantics. |
| `application.md` | Governed use-case orchestration and atomic transition coordination | DRAFT | Read when changing request-to-persistence orchestration. |
| `runtime.md` | Execution mechanics and lifecycle | DRAFT | Read when changing workflow execution. |
| `daemon.md` | Continuous availability and supervision | DRAFT | Read when changing background operation. |
| `departments.md` | Department extension architecture | DRAFT | Read when designing department contracts. |
| `intelligence.md` | Provider-independent intelligence routing | DRAFT | Read when working on model capabilities. |
| `coding-agents.md` | Vendor-independent coding-agent execution | DRAFT | Read when working on engineering agents. |
| `workspaces.md` | Isolated execution environments | DRAFT | Read when designing workspace lifecycle or isolation. |
| `governance.md` | Policy, authority, and approval boundaries | DRAFT | Read when changing governed actions. |
| `events.md` | Architectural event responsibilities | DRAFT | Read when changing event flow. |
| `knowledge.md` | Organizational knowledge ownership | DRAFT | Read when designing knowledge handling. |
| `persistence.md` | Authoritative persistence responsibilities | DRAFT | Read when changing durable state. |
