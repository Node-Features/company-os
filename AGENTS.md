# CompanyOS Agent Instructions

CompanyOS is an organizational runtime for semi-autonomous AI companies, not merely a multi-agent framework.

## Read first

1. Read this file.
2. Read [the context map](docs/INDEX.md).
3. Read [current state](.companyos/agent-memory/current-state.md).
4. Use the context map to select only files relevant to the task.
5. Read accepted ADRs that affect the task.

Do not read the entire repository by default.

## Canonical ownership

- [ARCHITECTURE.md](ARCHITECTURE.md) owns top-level architecture.
- `docs/architecture/` owns detailed architecture.
- `docs/domain/` owns domain definitions and invariants.
- `docs/adr/` owns accepted architectural decisions.
- `docs/features/` owns feature-specific evidence.
- `docs/references/` owns external implementation research.
- `.companyos/agent-memory/` contains routing summaries, not canonical truth.

Only approved canonical documents and accepted ADRs establish project truth.

## Scope and safety

- Inspect before editing; preserve useful existing content.
- Modify only files in scope and report necessary out-of-scope changes.
- Mark unresolved matters `OPEN QUESTION`; do not invent decisions.
- Do not claim planned functionality is implemented.
- Introduce infrastructure only for a concrete responsibility.
- Link to canonical definitions instead of duplicating them.

## Validation

Before completion, inspect the diff; check links, terminology, ownership, applicable invariants, and unsupported implementation claims; report open questions without reproducing complete files.
