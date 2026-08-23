# Architecture Decision Records

This directory owns accepted architectural decisions and their rationale; proposals are not authoritative until accepted.

## Acceptance authority and process

The CompanyOS project owner, `Node-Features`, is the initial ADR approver. An ADR is accepted only when all of the following are true:

1. it states the context, decision, alternatives, consequences, compatibility impact, and unresolved follow-up work;
2. affected architecture and domain owners have been checked for contradictions;
3. applicable evidence, links, terminology, security implications, and migration consequences have been reviewed;
4. all blocking [`CRITICAL` and `MAJOR` audit findings](../../AGENTS.md#audit-finding-severity) affecting the decision are resolved;
5. the project owner explicitly records acceptance by changing `Status: PROPOSED` to `Status: APPROVED` in a dedicated review change.

For ADRs, `APPROVED` means accepted and authoritative. Merge, implementation, or reference from a draft document does not imply acceptance. An accepted ADR is immutable except for non-semantic corrections; a changed decision requires a new ADR that identifies the previous ADR as `SUPERSEDED` after the replacement is approved.

| Document | Purpose | Status | Read when |
|---|---|---|---|
| `ADR-0001-kernel-runtime-daemon.md` | Separate semantics, execution, and availability | APPROVED | Read when changing Kernel, Runtime, or Daemon boundaries. |
| `ADR-0002-pluggable-departments.md` | Establish department extension contracts | APPROVED | Read when changing department architecture. |
| `ADR-0003-model-independent-intelligence.md` | Establish provider-independent intelligence | APPROVED | Read when changing model routing or adapters. |
| `ADR-0004-first-slice-technology-stack.md` | Select first-slice Runtime, persistence, notification, and capability technology | APPROVED | Read when changing the first-slice deployment topology or technology adapters. |
| `ADR-0005-kernel-interface-contracts.md` | Fix the Kernel's Go-level contract (function signatures, package layout) per aggregate | PROPOSED | Read when implementing or reviewing Kernel decision logic for a new aggregate. |
| `ADR-0006-daemon-boot-sequence.md` | Fix the Daemon's concrete boot sequence and tie it to the real `companyd` entrypoint | PROPOSED | Read when changing `cmd/companyd/main.go` or the process startup/shutdown sequence. |
| `ADR-0007-concurrency-model.md` | State which operations are concurrent vs. serialized and why, with a compiled worked example | PROPOSED | Read when adding concurrent dispatch, shared mutable state, or evaluating a synchronization choice. |
| `ADR-0008-authority-capability-model.md` | Add role-scoped policy matching to Governance; map Identity/Role/Authority/Action/Resource onto real types | PROPOSED | Read when adding a role-scoped permission or before naming anything "Capability" outside `docs/domain/capability.md`'s meaning. |
| `ADR-0009-caching-and-agent-messaging-infrastructure.md` | Adopt Redis as a disposable cache layer; scope QStash to `web`-side async work pending the Node multi-process question | PROPOSED | Read before adding a cache layer, or before using Redis/QStash anywhere in this codebase. |
