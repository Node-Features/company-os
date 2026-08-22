# CompanyOS Workspace Isolation

Status: APPROVED

## Purpose

This document specifies Engineering workspace isolation requirements and answers [`workspaces.md`](../architecture/workspaces.md)'s open question of which initial provider mechanism satisfies the required isolation controls, so Phase 6's Workspace-lifecycle slice does not have to invent that decision mid-implementation.

## Decision: containers as the first `WorkspaceProvider` mechanism

The first `WorkspaceProvider` implementation provisions one OCI container (Docker) per `EngineeringWorkspace`, running on the same infrastructure `companyd` already occupies (the DigitalOcean VPS from [`ADR-0004`](../adr/ADR-0004-first-slice-technology-stack.md)), not a separate VM or third-party cloud sandbox service.

### Why containers, and why first

- **Matches the project's established low-infrastructure-footprint posture.** `ADR-0004` deliberately chose one VPS + systemd over a heavier orchestration platform for the first slice. A container runtime on that same VPS is the smallest addition consistent with that choice — a VM-per-task or third-party sandbox service would add a second infrastructure dependency before the simpler option has been shown insufficient.
- **Satisfies `workspaces.md`'s isolation table with standard, well-understood primitives**, not a novel mechanism this document would have to invent controls for:
  - *Filesystem:* a container's own filesystem/bind-mount scoping confines the task checkout and approved caches; nothing outside the mount is reachable.
  - *Processes:* container process-tree confinement (PID namespace) plus cgroup resource limits satisfy process/resource-consumption confinement, including descendant termination.
  - *Network:* container network namespaces default to no external reachability; an explicit allowlist opens only what the task declares, matching [`tool-security.md`](tool-security.md)'s default-deny network posture.
  - *Credentials:* short-lived, task-scoped credentials are injected as container environment/secret-mount material at start and revoked at destroy, never baked into a shared image.
  - *Tenancy:* one container per workspace, one workspace per task — no image or cache is shared in a way that crosses organization boundaries by default.
- **It is explicitly not claimed to be sufficient by itself.** Per `workspaces.md`: "Containerization alone is not proof of isolation." This document's job is to require verification, not to declare containers automatically safe — see Verification below.

### What this decision does not close

- **Container escape is a real residual risk**, not eliminated by this choice. A workload with attacker-influenced code (an untrusted agent's own output, or a compromised dependency it installs) run inside a container shares the host kernel. This document accepts that risk for the first slice, scoped narrowly to CompanyOS's own coding-agent tasks (not arbitrary third-party workloads), and defers stronger isolation (gVisor/Kata/microVM, or a dedicated VM per task) to a later phase if evidence shows container isolation insufficient for the actual threat profile.
- **Multi-tenant hosting is not this decision's scope.** This mechanism is chosen for CompanyOS's own single-organization-per-database first slice ([Phase 9 Slice 4](../../ROADMAP.md) still pending real multi-tenant RLS). Revisit this decision before onboarding a second organization's Engineering workloads onto shared container infrastructure.

## Verification before `Ready`

A workspace becomes `Ready` (per [`domain/workspace.md`](../domain/workspace.md)'s lifecycle) only after the provider's isolation claims are checked, not merely asserted:

- Filesystem scope: confirm the container cannot read/write paths outside its mounted checkout and declared cache paths.
- Process confinement: confirm resource limits (CPU, memory, execution time) are enforced, not merely configured.
- Network policy: confirm default-deny is actually in effect (a probe to a non-allowlisted destination fails) before trusting the allowlist.
- Credential scope: confirm no credential broader than the task's declared need is present in the container's environment.
- Tenancy: confirm no other organization's cache, image layer, or artifact is reachable from inside the container.

A `WorkspaceProvider` that cannot produce evidence for each of these does not satisfy this document, regardless of its underlying technology.

## Recovery and lifecycle interaction

This document does not redefine [`workspaces.md`](../architecture/workspaces.md#lifecycle)'s lease/checkpoint/recovery mechanics. It adds one isolation-specific requirement: a recovered or resumed workspace must re-verify network/credential/filesystem scope (above) rather than assuming a prior verification still holds, since a container's underlying host or runtime may have changed between suspend and resume.

## Invariants

- No workspace reaches `Ready` without passing the verification checks above.
- No workspace shares a container, image, or credential scope with another organization's workspace.
- No credential broader than the current task's declared need is ever present inside the workspace.
- A container escape or isolation-verification failure is treated as a security incident (see `threat-model.md`), not a retryable task failure.
- Choosing containers here does not preclude a stronger mechanism (VM, microVM, remote workspace service) for a specific task class later; this document fixes the *first* mechanism, not the only one ever allowed.

## Open questions

- OPEN QUESTION: at what evidence threshold (a specific incident class, a specific task risk tier) does this decision get revisited toward VM-level or microVM isolation?
- OPEN QUESTION: what concrete container runtime and image-build pipeline is used, and who owns base-image supply-chain integrity?
- OPEN QUESTION: how is per-task resource-limit sizing decided, and does it draw on Finance's `ResourceConstraint` the way `CodingAgentRouter` already does for eligibility?
- OPEN QUESTION: what is the destroy-time guarantee — is a destroyed container's filesystem provably wiped, or only unmounted/garbage-collected?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [`threat-model.md`](threat-model.md)
- [`tool-security.md`](tool-security.md)
- [Workspaces](../architecture/workspaces.md)
- [Workspace domain](../domain/workspace.md)
- [Coding-agents](../architecture/coding-agents.md)
- [`ADR-0004`](../adr/ADR-0004-first-slice-technology-stack.md)
