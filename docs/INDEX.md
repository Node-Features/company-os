# CompanyOS Context Map

Use this map to load the smallest sufficient context for a task. Summaries route readers to canonical sources and never replace them.

| Task type | Required files | Optional files | Normally excluded |
|---|---|---|---|
| Any task | [`AGENTS.md`](../AGENTS.md), [current state](../.companyos/agent-memory/current-state.md) | Files selected below | Entire documentation tree |
| Architecture | [`ARCHITECTURE.md`](../ARCHITECTURE.md), relevant files in [`architecture/`](architecture/README.md) | [Architecture summary](../.companyos/agent-memory/architecture-summary.md), relevant accepted ADRs | Unrelated domains, features, and references |
| Domain | Relevant files in [`domain/`](domain/README.md) | [Domain summary](../.companyos/agent-memory/domain-summary.md), relevant accepted ADRs | Unrelated architecture details and features |
| Architectural decision | [`ARCHITECTURE.md`](../ARCHITECTURE.md), relevant files in [`adr/`](adr/README.md) | [Decision index](../.companyos/agent-memory/decisions.md), relevant references | Unrelated ADRs and features |
| Feature | Relevant feature directory in [`features/`](features/README.md), affected canonical documents | [Feature index](../.companyos/agent-memory/feature-index.json), relevant references and ADRs | Unaffected features and reference studies |
| Reference research | Relevant files in [`references/`](references/README.md) | Architecture or domain owner affected by the research | Unrelated reference projects |
| Status update | [Current state](../.companyos/agent-memory/current-state.md) | Changed canonical documents and accepted ADRs | Unrelated project documentation |

## Core areas

- [Architecture](architecture/README.md)
- [Domain](domain/README.md)
- [Architectural decisions](adr/README.md)
- [Features](features/README.md)
- [External references](references/README.md)
