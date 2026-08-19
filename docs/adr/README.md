# Architecture Decision Records

This directory owns accepted architectural decisions and their rationale; proposals are not authoritative until accepted.

## Acceptance authority and process

The CompanyOS project owner, `Node-Features`, is the initial ADR approver. An ADR is accepted only when all of the following are true:

1. it states the context, decision, alternatives, consequences, compatibility impact, and unresolved follow-up work;
2. affected architecture and domain owners have been checked for contradictions;
3. applicable evidence, links, terminology, security implications, and migration consequences have been reviewed;
4. all blocking `CRITICAL` and `MAJOR` audit findings affecting the decision are resolved;
5. the project owner explicitly records acceptance by changing `Status: PROPOSED` to `Status: APPROVED` in a dedicated review change.

For ADRs, `APPROVED` means accepted and authoritative. Merge, implementation, or reference from a draft document does not imply acceptance. An accepted ADR is immutable except for non-semantic corrections; a changed decision requires a new ADR that identifies the previous ADR as `SUPERSEDED` after the replacement is approved.

| Document | Purpose | Status | Read when |
|---|---|---|---|
| `ADR-0001-kernel-runtime-daemon.md` | Separate semantics, execution, and availability | NOT YET SPECIFIED | Read when changing Kernel, Runtime, or Daemon boundaries. |
| `ADR-0002-pluggable-departments.md` | Establish department extension contracts | NOT YET SPECIFIED | Read when changing department architecture. |
| `ADR-0003-model-independent-intelligence.md` | Establish provider-independent intelligence | PROPOSED | Read when changing model routing or adapters. |
