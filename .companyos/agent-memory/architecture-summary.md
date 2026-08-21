# Architecture Summary

Status: APPROVED (top-level and all 14 `docs/architecture/*.md` documents)

`ARCHITECTURE.md` and all 14 detailed architecture documents (`system-context.md`, `identity.md`, `kernel.md`, `application.md`, `runtime.md`, `daemon.md`, `departments.md`, `intelligence.md`, `coding-agents.md`, `workspaces.md`, `governance.md`, `events.md`, `knowledge.md`, `persistence.md`) were approved by the project owner on 2026-08-20. See `current-state.md` for the review details and the gaps resolved before approval.

First-slice implementation technology (Runtime hosting, persistence adapter, notification path, first capability) is approved via `ADR-0004` (2026-08-21) — see `decisions.md`. Four implementation sub-questions within that decision remain open (LLM provider, transport, RLS design, retry defaults).

Canonical sources:

- [Top-level architecture](../../ARCHITECTURE.md)
- [Detailed architecture index](../../docs/architecture/README.md)
- [Accepted decision index](decisions.md)
- [Current state](current-state.md)

This file is a routing summary and must not establish architectural decisions.
