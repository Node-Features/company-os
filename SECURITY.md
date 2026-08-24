# Security Policy

Status: DRAFT

## Project state

CompanyOS is pre-alpha. A real runtime (`companyd`) implementing governed execution exists and runs against a real database in development (see [README.md — Project status](README.md#project-status)), but nothing is deployed to production or otherwise publicly reachable yet (`ROADMAP.md` Phase 9, unstarted) — so there is currently no hosted service, released package, or live attack surface an external party could reach. This policy governs the repository (documentation and code) until a first production deployment exists, at which point it must be revised.

## Reporting a vulnerability

Report suspected vulnerabilities privately through GitHub's security advisory feature on this repository ([Node-Features/company-os](https://github.com/Node-Features/company-os) — Security tab → "Report a vulnerability"), not through public issues, discussions, or pull requests.

Include affected file(s) or component(s), the concern, and reproduction or reasoning steps where applicable. Response time, triage process, and disclosure timeline are not yet defined.

- OPEN QUESTION: Who triages reports, and what response-time commitment applies?
- OPEN QUESTION: What coordinated-disclosure timeline applies once implementation exists?

## Security posture already established by architecture

These boundaries are specified in the canonical architecture documents below. They describe intended design, not a verified implementation, since no runtime exists yet.

- **Untrusted external input:** all external content, model output, tool output, callbacks, and provider events are untrusted until validated. See [System context — Trust and authority boundaries](docs/architecture/system-context.md#trust-and-authority-boundaries).
- **Authentication is not authorization:** Identity establishes attributable actor identity; only Governance decides `ALLOW`, `DENY`, or `REQUIRE_APPROVAL`. Authentication success never implies a permit. See [Identity](docs/architecture/identity.md) and [Governance](docs/architecture/governance.md).
- **Default-deny policy:** absence of a matching permit never implies access; any matching forbid overrides permits and approvals; evaluation errors and uncertainty fail closed. See [Governance — Policy composition](docs/architecture/governance.md#policy-composition).
- **No self-approval:** agents can never approve their own actions; a human requester cannot approve their own request except under an explicit future separation-of-duties policy. See [Governance invariants](docs/architecture/governance.md#invariants).
- **Agents cannot self-authorize:** an agent may propose commands and invoke permitted capabilities but cannot grant itself authority or directly mutate authoritative organizational state. See [System context — Agent](docs/architecture/system-context.md#agent).
- **Least-privilege, scoped credentials:** credentials remain scoped to the smallest provider, capability, workspace, action, and lifetime required, and are never embedded in prompts, repositories, images, checkpoints, or logs. See [Workspaces — Isolation and authority boundaries](docs/architecture/workspaces.md#isolation-and-authority-boundaries).
- **Workspace isolation:** engineering execution is confined by default-deny network policy, filesystem scoping to the task checkout, and process/resource confinement; containerization alone is not proof of isolation. See [Workspaces](docs/architecture/workspaces.md).
- **Persistence before execution:** accepted state, domain events, and execution intent commit atomically before dependent execution continues, so provider failure, delay, duplication, or compromise cannot corrupt authoritative state. See [System context — Trust and authority boundaries](docs/architecture/system-context.md#trust-and-authority-boundaries).
- **Human override:** humans can pause, redirect, or override execution through governed operations; emergency pause and revocation are explicit, fail-closed operations. See [Identity — Delegation and revocation](docs/architecture/identity.md#delegation-and-revocation) and [Governance invariants](docs/architecture/governance.md#invariants).

## Detailed security documentation

Threat models, agent-authority limits, tool-access controls, and workspace-isolation requirements are detailed in [docs/security/](docs/security/README.md): [`threat-model.md`](docs/security/threat-model.md), [`agent-authority.md`](docs/security/agent-authority.md), [`tool-security.md`](docs/security/tool-security.md), and [`workspace-isolation.md`](docs/security/workspace-isolation.md), all `APPROVED` as of 2026-08-22 (`ROADMAP.md` Phase 2). This file remains the project-wide summary; those documents own the actual mechanism and reasoning.

## Supported versions

Not applicable. No versioned release exists yet.
