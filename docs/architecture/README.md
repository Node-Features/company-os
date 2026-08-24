# Architecture Documentation

This directory owns detailed architectural responsibilities and boundaries; `ARCHITECTURE.md` remains the canonical overview.

## Approval authority

The CompanyOS project owner, `Node-Features`, is the initial architecture approver. Approval must be explicit and attributable; authorship, merge, silence, or elapsed time does not imply approval.

An architecture document moves from `DRAFT` to `APPROVED` only when:

1. its responsibility, non-responsibilities, dependencies, invariants, failure semantics, and open questions have been reviewed;
2. conflicts with `ARCHITECTURE.md`, approved documents, and accepted ADRs have been resolved;
3. applicable links, terminology, diagrams, and implementation claims have been validated;
4. all blocking [`CRITICAL` and `MAJOR` audit findings](../../AGENTS.md#audit-finding-severity) affecting it are resolved;
5. the project owner explicitly records approval by changing its status to `APPROVED` in a dedicated review change.

Approval applies to the reviewed version. A material boundary or invariant change returns the document to `DRAFT` unless an accepted ADR already authorizes that change. Documents may be reviewed together, but each status changes explicitly. No architecture document is approved merely by appearing in this index.

| Document | Purpose | Status | Read when |
|---|---|---|---|
| `system-context.md` | System actors, boundaries, and external relationships | APPROVED | Read when establishing system scope. |
| `identity.md` | Authentication, durable Principal identity, delegation, and trusted claims | APPROVED | Read when establishing actor identity or authentication evidence. |
| `kernel.md` | Kernel ownership and non-responsibilities | APPROVED | Read when changing organizational semantics. |
| `application.md` | Governed use-case orchestration and atomic transition coordination | APPROVED | Read when changing request-to-persistence orchestration. |
| `runtime.md` | Execution mechanics and lifecycle | APPROVED | Read when changing workflow execution. |
| `daemon.md` | Continuous availability and supervision | APPROVED | Read when changing background operation. |
| `departments.md` | Department extension architecture | APPROVED | Read when designing department contracts. |
| `intelligence.md` | Provider-independent intelligence routing | APPROVED | Read when working on model capabilities. |
| `coding-agents.md` | Vendor-independent coding-agent execution | APPROVED | Read when working on engineering agents. |
| `workspaces.md` | Isolated execution environments | APPROVED | Read when designing workspace lifecycle or isolation. |
| `governance.md` | Policy, authority, and approval boundaries | APPROVED | Read when changing governed actions. |
| `events.md` | Architectural event responsibilities | APPROVED | Read when changing event flow. |
| `knowledge.md` | Organizational knowledge ownership | APPROVED | Read when designing knowledge handling. |
| `persistence.md` | Authoritative persistence responsibilities | APPROVED | Read when changing durable state. |
| `node.md` | Runtime/compute node identity, capabilities, and scheduler placement | DRAFT | Read when designing multi-node execution capacity, distinct from organizational topology. |
| `ui-ux.md` | Product UI screen inventory, data-contract mapping, visual direction, and cross-cutting interaction invariants | APPROVED | Read before building or extending any `web` (or successor) product screen. |
| `authority-model.md` | Formalized `Principal + Organization + Department + Role + Capability + Action + Resource + Scope + Autonomy Class + Constraints + Evidence` tuple; the LEGALITY/AUTHORITY/APPROVAL layer split and evaluator pipeline `governance.md`'s Decision pipeline maps onto | APPROVED | Read when changing Governance's decision model, autonomy levels, or the Approval domain's structural invariants. |
| `observability.md` | Correlation identity, structured logging, and operational metrics over the execution lifecycle — diagnostic only, never authoritative | APPROVED | Read when adding a log/metric emission point, or reasoning about what `/health`/`/metrics` report. |
