# CompanyOS Tool Security

Status: APPROVED

## Purpose

This document specifies tool access and external-action controls: what a "tool" is allowed to do, how its scope is bounded, and how its use is verified. It is a named dependency of [`coding-agents.md`](../architecture/coding-agents.md)'s normalized-tool-policy invariant, and grounds the "Tools" row of [`workspaces.md`](../architecture/workspaces.md#isolation-and-authority-boundaries)'s isolation table before Phase 6 builds coding-agent tool execution against it.

## What a tool is, in this document's scope

A tool is any capability an agent or coding-agent session can invoke that produces an effect or observation outside pure reasoning: shell commands, file reads/writes, network calls, Git operations, external API/capability requests, and publication actions. This is deliberately broader than "Engineering tools" — the same discipline applies to any Intelligence-routed [`CapabilityRequest`](../domain/capability.md) that reaches an external system.

## Principle: tools execute, they do not authorize

A tool call is **mechanism**, never **authority**. Restating the structural boundary this document must not blur:

- A tool request is proposed by an agent, normalized, and — for anything governed — evaluated by Governance before it takes effect. The tool itself does not decide legality.
- A tool's success or failure report is evidence, never proof of correctness. [Coding-agents invariants](../architecture/coding-agents.md#invariants): "Agent plans, messages, memory, command output, and success claims are non-authoritative evidence." The same applies to any tool, not only coding-agent tools.
- A tool cannot expand the authority of the Principal invoking it. See [`agent-authority.md`](agent-authority.md) for that Principal-level constraint; this document constrains the tool's own operating envelope given whatever authority the Principal already has.

## Tool classification

Every tool must be classified along these axes before it is made available to any agent or coding-agent session:

| Axis | Values | Consequence |
|---|---|---|
| Effect reversibility | reversible / irreversible | Irreversible tools (publish, merge, pay, delete) default to `approval_required`/`human_only` per [`agent-authority.md`](agent-authority.md#autonomy-class-defaults-by-action-shape) |
| Visibility | internal-only / externally visible | Externally visible effects (a sent message, a public commit) require attribution to a specific governed action, not a generic "agent did something" |
| Data classification | none / internal / confidential | Confidential-data tools require the same organization-scoping Identity already enforces ([Identity — Organization scoping](../architecture/identity.md#organization-scoping)) |
| Credential requirement | none / scoped credential / privileged credential | A privileged-credential tool is disallowed inside a Workspace by default — see [Workspaces — Credentials](../architecture/workspaces.md#isolation-and-authority-boundaries) |

An unclassified tool is not available. This mirrors Governance's default-deny posture at the tool layer: the absence of a classification is not a permissive default.

## Least privilege and scoping

- Tool access is scoped per task/session, not per Principal globally — an `EngineeringTask`'s `allowed-tool` list ([Coding-agents — EngineeringTask](../architecture/coding-agents.md#engineeringtask)) is the concrete mechanism; other departments' capability requests carry the equivalent constraint through their own `CapabilityRequest`.
- Credentials issued for tool use are short-lived, task-scoped, and issued just in time, never embedded in prompts, repositories, images, checkpoints, or logs — restating [Workspaces — Credentials](../architecture/workspaces.md#isolation-and-authority-boundaries) as binding on every tool, not only Engineering's.
- A tool that can escalate its own privileges (e.g. a shell tool that can install software or modify its own sandbox) is treated as an irreversible, externally-visible tool for classification purposes regardless of its apparent primary function.
- Network-capable tools default to deny; specific destinations/protocols are allowlisted per task, per [Workspaces — Network](../architecture/workspaces.md#isolation-and-authority-boundaries).

## Verification, not trust

Restating [Coding-agents — Invariants](../architecture/coding-agents.md#invariants) as the general tool-security rule: required checks (tests, lints, builds, scans) are independently observed by CompanyOS, and a tool or the agent invoking it cannot waive or redefine them. A tool reporting "success" is not accepted as ground truth for:

- whether a file change is within the task's declared file-scope;
- whether a command's exit code and output constitute the claimed outcome;
- whether a network call reached only an allowlisted destination;
- whether a published/sent effect actually occurred as described.

Each of these is independently verified per [Coding-agents — Execution lifecycle](../architecture/coding-agents.md#execution-lifecycle) step 7, generalized here to every tool-using department, not only Engineering.

## Prohibited-operation policy

A task or session declares prohibited operations explicitly ([Coding-agents — EngineeringTask](../architecture/coding-agents.md#engineeringtask)). At minimum, every tool policy prohibits, by default, absent an explicit and reviewed exception:

- modifying files outside the task's declared file-scope;
- reading or exporting secrets, credentials, or another organization's data;
- disabling, skipping, or weakening required validation (tests, lints, security scans);
- direct commit/push/merge/publish without passing through the corresponding governed action;
- installing or invoking tooling not declared in the task's `allowed-tool` list.

## Invariants

- Every tool available to an agent or coding-agent session has a recorded classification (reversibility, visibility, data classification, credential requirement) before use.
- No tool call is treated as self-authorizing; governed effects still require a current `ALLOW` Governance decision.
- Tool-reported success or failure is evidence, never authoritative proof, for any tool, in any department.
- Credentials issued for tool use are scoped to the smallest provider, capability, workspace, action, and lifetime required, and never appear in prompts, repositories, artifacts, checkpoints, or logs.
- A prohibited-operation violation is a policy failure, not a tool bug to silently retry past.

## Open questions

- OPEN QUESTION: is tool classification a static registry (per adapter/provider) or a per-task declaration — and who is the reviewing authority for a new classification?
- OPEN QUESTION: how are non-Engineering departments' capability requests (Research web/data fetches, Finance external payment APIs) brought under this same classification discipline concretely — this document states the shape, not each department's registry.
- OPEN QUESTION: what automated enforcement exists versus what remains a documented expectation reviewed by a human — this document does not assume a policy engine exists for every axis above yet.

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [`threat-model.md`](threat-model.md)
- [`agent-authority.md`](agent-authority.md)
- [`workspace-isolation.md`](workspace-isolation.md)
- [Coding-agents](../architecture/coding-agents.md)
- [Workspaces](../architecture/workspaces.md)
- [Capability domain](../domain/capability.md)
- [Governance](../architecture/governance.md)
