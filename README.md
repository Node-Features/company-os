# CompanyOS

**An open-source organizational runtime for building and operating semi-autonomous AI companies.**

> **Project status:** documentation foundation / pre-alpha. CompanyOS is defining its architecture, domain model, governance boundaries, and reference provenance. Production runtime functionality is not implemented yet.

CompanyOS explores how an organization can coordinate people, AI agents, workflows, policies, capabilities, evidence, and resources as one governed system. It is not merely a multi-agent framework: its primary abstraction is the organization itself.

## Vision

CompanyOS is intended to connect an organization's:

```text
mission and vision
→ policies and objectives
→ departments and workflows
→ agents, teams, and capabilities
→ actions, artifacts, and results
→ metrics and evaluation
→ research and strategic adaptation
```

The system is designed for bounded autonomy. Authorized work may continue without constant human presence, while governance determines which actions are automatic, approval-gated, or human-only.

## Architectural direction

CompanyOS separates organizational meaning from execution infrastructure:

| Area | Responsibility |
|---|---|
| **Kernel** | Organization identity, semantics, policies, objectives, departments, governance, and invariants |
| **Runtime** | Workflow scheduling, dispatch, execution, retry, cancellation, checkpointing, and resume |
| **Daemon** | Continuous availability, event processing, waking due work, and health supervision |
| **Departments** | Pluggable organizational functions using shared contracts and capabilities |
| **Intelligence** | Provider-independent model capabilities, routing, and evidence-driven selection |
| **CodingAgentRuntime** | Vendor-independent engineering execution through coding-agent adapters |
| **Workspaces** | Isolated environments for controlled engineering work |

The canonical architecture will live in [ARCHITECTURE.md](ARCHITECTURE.md). It is currently not yet specified and must not be inferred from this overview.

## Adaptive organization

Three functions form the central learning loop:

- **Research** discovers opportunities, risks, customer needs, technology changes, models, coding agents, and open-source patterns.
- **Monitoring & Evaluation** independently measures effectiveness, quality, reliability, outcomes, and agent performance.
- **Finance** governs budgets, resource consumption, pricing intelligence, and effective cost per result.

Together they allow evidence from execution to influence future decisions without giving any model or agent unilateral organizational authority.

## Core principles

- Organizational semantics belong to the Kernel.
- Execution mechanics belong to the Runtime.
- Continuous availability belongs to the Daemon.
- Departments request capabilities, not concrete vendors.
- Agents cannot directly mutate authoritative workflow state.
- Domain rules determine legal transitions.
- Governed actions pass policy and approval checks.
- Authoritative state is persisted before execution continues.
- Humans can safely pause, override, or redirect execution.
- Infrastructure is introduced only for a demonstrated responsibility.
- Planned features are never presented as implemented features.

## Open-source provenance

CompanyOS studies mature projects to shorten the path to reliable implementation without surrendering its defining abstractions. Current references include:

- [JARVIS](https://github.com/vierisid/jarvis) for selected agent coordination, authority, approval, and audit patterns.
- [LangGraph](https://github.com/langchain-ai/langgraph) for graph execution and checkpoint concepts.
- [Temporal](https://github.com/temporalio/temporal) for durable execution and recovery semantics.
- [Cedar](https://github.com/cedar-policy/cedar) for authorization and policy validation.
- Inngest, Trigger.dev, OpenAI Agents SDK, and Open Multi-Agent as additional bounded references.

See the [feature-provenance map](docs/references/feature-provenance.md) and [pinned revision lock](docs/references/references.lock.md). References inform decisions; they do not automatically become dependencies.

## Repository guide

```text
AGENTS.md                         Agent working rules and context entry point
ARCHITECTURE.md                   Canonical top-level architecture
docs/INDEX.md                     Task-to-context routing map
docs/architecture/                Detailed architecture
docs/domain/                      Domain concepts and invariants
docs/adr/                         Accepted architectural decisions
docs/features/                    Significant feature evidence
docs/references/                  Pinned open-source research and provenance
.companyos/agent-memory/          Compact non-canonical routing summaries
```

Start with [AGENTS.md](AGENTS.md), then use [docs/INDEX.md](docs/INDEX.md) to load only the documentation relevant to a task.

## Current roadmap

1. Approve the documentation-control foundation.
2. Define mission, vision, principles, and non-goals.
3. Freeze Kernel, Runtime, Daemon, governance, and provider boundaries.
4. Define domain concepts and invariants.
5. Complete pinned source studies and architectural decisions.
6. Specify departments and vertical workflows.
7. Implement the smallest production-quality vertical slice.

Development will proceed through reviewed vertical slices rather than feature-count expansion.

## Contributing

CompanyOS is at an early architecture stage. Before proposing code:

1. Read [AGENTS.md](AGENTS.md) and the [context map](docs/INDEX.md).
2. Identify the owning module and applicable invariants.
3. Inspect relevant accepted ADRs and pinned references.
4. Propose the smallest vertical change, including failure, security, and validation considerations.
5. Avoid introducing infrastructure without a concrete requirement.

Contribution guidelines, security reporting instructions, and licensing terms will be added before accepting production contributions.

## Important note

CompanyOS is experimental, pre-alpha software. The repository currently contains architectural and research documentation, not a production-ready autonomous-company runtime.
