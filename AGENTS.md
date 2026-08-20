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

## Audit finding severity

An audit or review classifies each finding so the approval gates in `docs/architecture/README.md` and `docs/adr/README.md` can be checked consistently rather than by unwritten judgment:

- `CRITICAL`: an active contradiction, a bypassable authority/approval boundary, or two documents that could both be treated as authoritative for the same concept. Approval cannot proceed until resolved.
- `MAJOR`: blocks confident implementation without yet demonstrating an active contradiction — for example a contract used by multiple documents is defined in only one, a load-bearing cross-reference is missing, or a lifecycle/vocabulary mismatch exists between an owning document and its consumers. Approval cannot proceed until resolved.
- `MINOR`: worth fixing but does not block approval — stale wording, a harmless unresolved `OPEN QUESTION`, or a citation inconsistency that does not change meaning.

State the severity explicitly when reporting audit or review findings rather than leaving it to be inferred.
