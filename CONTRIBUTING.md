# Contributing to CompanyOS

Status: DRAFT

CompanyOS's core architecture and domain documents are approved and Phases 1–5 of the roadmap (first vertical slice, security/testing foundations, governed execution, the adaptive-organization departments, and organizational knowledge) are implemented and running against a real database (see [README.md — Roadmap](README.md#roadmap) and [current state](.companyos/agent-memory/current-state.md)). Code contributions are accepted; the contribution workflow itself (this document) is still evolving, so expect it to change.

## Before proposing a change

1. Read [AGENTS.md](AGENTS.md) — the agent and contributor working rules and context entry point.
2. Use [docs/INDEX.md](docs/INDEX.md) to load only the documentation relevant to your task; do not read the entire repository by default.
3. Identify the module or document that owns the concept you're changing (see "Canonical ownership" in `AGENTS.md`).
4. Read applicable domain invariants ([docs/domain/](docs/domain/README.md)) and accepted ADRs ([docs/adr/](docs/adr/README.md)) affecting the area.
5. Study relevant pinned open-source references ([docs/references/](docs/references/README.md)) when a design borrows an external pattern.
6. Propose the smallest defensible vertical change rather than a broad rewrite.
7. Include failure, security, test, and architectural considerations in the proposal.
8. Avoid introducing infrastructure, dependencies, or abstractions without a demonstrated responsibility.

## Documentation changes

- Inspect existing content before editing; preserve useful material rather than replacing it wholesale.
- Modify only files in scope, and report necessary out-of-scope changes rather than making them silently.
- Mark unresolved matters `OPEN QUESTION` instead of inventing a decision.
- Never present a planned feature as implemented.
- Link to canonical definitions instead of duplicating them across documents.

## Architecture and domain changes

- `ARCHITECTURE.md` and each `docs/architecture/*.md` document start `DRAFT` and require the project owner (`Node-Features`) to explicitly approve them before they're an accepted contract — most already are `APPROVED` (check the document's own `Status:` line, not this note); see [docs/architecture/README.md — Approval authority](docs/architecture/README.md#approval-authority).
- A change to a Kernel, Runtime, Daemon, or Governance boundary should go through the ADR process; see [docs/adr/README.md](docs/adr/README.md) for acceptance criteria.

## Before requesting review

Inspect the diff; check links, terminology, ownership, applicable invariants, and unsupported implementation claims — the same checklist `AGENTS.md` applies to any agent-authored change.

## Security

Report vulnerabilities privately as described in [SECURITY.md](SECURITY.md), not through a public issue or pull request.

## Roadmap

See [ROADMAP.md](ROADMAP.md) for the current phase and near-term priorities.
