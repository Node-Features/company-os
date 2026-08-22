# CompanyOS Documentation

This directory indexes CompanyOS documentation without duplicating its canonical content.

| Area | Responsibility | Status | Read when |
|---|---|---|---|
| [Architecture](architecture/README.md) | Architectural boundaries and component ownership | APPROVED | Read when making or reviewing architectural decisions. |
| [Domain](domain/README.md) | Domain definitions and invariants | APPROVED | Read when working with organizational concepts or state. |
| [Workflows](workflows/README.md) | End-to-end workflow semantics | NOT YET SPECIFIED | Read when defining or changing workflow behavior. |
| [Departments](departments/README.md) | Department responsibilities and authority | PARTIAL (3 of 7 approved) | Read when defining departmental behavior or integration. |
| [References](reference-implementations/README.md) | Source-based study of mature systems | NOT YET SPECIFIED | Read when evaluating an implementation pattern. |
| [ADRs](adr/README.md) | Accepted architectural decisions | APPROVED | Read ADRs relevant to the current task. |
| [Security](security/README.md) | Detailed security requirements | APPROVED | Read for authority, tools, isolation, or threat analysis. |
| [Testing](testing/README.md) | Testing and failure-validation strategy | APPROVED | Read when designing validation or implementation tests. |
| [Features](features/README.md) | Significant feature documentation | NOT YET SPECIFIED | Read when planning or delivering a significant feature. |

Note (2026-08-21): the References row links `reference-implementations/README.md`, which conflicts with the separate, more current-looking `references/README.md` (see `.companyos/agent-memory/current-state.md#open-questions`) — left unresolved pending a decision on which directory is canonical.
