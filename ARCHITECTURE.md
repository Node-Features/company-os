# CompanyOS Architecture

Status: DRAFT

This file is the canonical top-level architecture document for CompanyOS. It remains a draft until approved by the project owner.

## System boundary

CompanyOS is an organizational runtime for semi-autonomous AI companies. It owns organizational semantics, governance, provider-independent capabilities, legal workflow state, and authoritative organizational records. It coordinates external models, coding agents, source control, publishing platforms, deployment infrastructure, databases, workflow engines, workspace providers, and future systems through CompanyOS-owned ports and governed operations.

External execution never transfers ownership of organizational meaning or authority. Agent messages, model responses, provider sessions, workspace files, caches, and external engine state are not authoritative organizational state unless validated and committed through CompanyOS domain operations.

See [System Context](docs/architecture/system-context.md) for actors, terminology, trust boundaries, external systems, inspected references, and open questions.

## Proposed top-level responsibilities

- **Kernel:** organizational semantics and invariants.
- **Application:** use-case orchestration across Governance, Kernel, Persistence, and Runtime.
- **Runtime:** workflow execution mechanics.
- **Daemon:** continuous availability and supervision.
- **Departments:** pluggable organizational functions using shared contracts.
- **Intelligence:** provider-independent intelligence capability and routing.
- **CodingAgentRuntime:** provider-independent engineering execution.
- **Workspaces:** isolated execution environments behind a stable contract.
- **Governance:** authority, policy, approvals, overrides, and external-action gates.
- **Persistence:** authoritative durable state behind CompanyOS-owned transactional contracts.

These responsibilities require further specification in [detailed architecture documents](docs/architecture/README.md). No concrete database, workflow engine, agent framework, model provider, coding-agent provider, or workspace provider is selected by this draft.

## Boundary invariants

- CompanyOS owns organizational meaning even when execution is delegated.
- External providers implement replaceable adapters and cannot become domain authorities.
- Departments depend on capabilities and shared contracts, not vendor SDKs or department internals.
- Agents may propose actions but cannot grant themselves authority or directly mutate authoritative workflow state.
- Domain rules determine legal transitions; governance determines whether external actions are allowed, denied, approval-required, or human-only.
- The Application layer coordinates requests but owns neither domain legality nor authorization policy.
- Accepted state, resulting domain events, and caused execution intent are persisted atomically before Runtime is notified.
- Persistence succeeds before dependent execution continues.
- Provider, agent, and transport state is not authoritative merely because it exists.
- No external framework or infrastructure is adopted without a concrete responsibility and documented decision.
