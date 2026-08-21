# Decision Index

All four ADRs are accepted (`Status: APPROVED`):

| ADR | Purpose | Status |
|---|---|---|
| `ADR-0001-kernel-runtime-daemon.md` | Separate semantics, execution, and availability | APPROVED (2026-08-20) |
| `ADR-0002-pluggable-departments.md` | Establish department extension contracts | APPROVED (2026-08-20) |
| `ADR-0003-model-independent-intelligence.md` | Establish provider-independent intelligence | APPROVED (2026-08-20) |
| `ADR-0004-first-slice-technology-stack.md` | Select first-slice Runtime, persistence, notification, and capability technology | APPROVED (2026-08-21) |

`ADR-0004` still has four open sub-questions (LLM provider, Next.js↔`companyd` transport, Supabase RLS design, first Retry policy defaults) — unresolved implementation detail within the decision, not a condition of its acceptance. See `current-state.md`.

See the [ADR index](../../docs/adr/README.md) for full acceptance criteria and the [ADR files themselves](../../docs/adr/) for rationale. This file indexes accepted decisions only and does not duplicate their rationale.
