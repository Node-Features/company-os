# CompanyOS Kernel

Status: DRAFT

## Responsibility

The Kernel is the provider-independent domain authority for CompanyOS. It defines organizational meaning and decides whether a requested organizational transition is legal. It is usable without a running scheduler, worker, daemon, model, or external workflow engine.

The Kernel owns:

- organization identity, mission, vision, principles, and policy semantics;
- objective identity, lifecycle, relationships, success criteria, and legal transitions;
- department identity, responsibilities, authority boundaries, and extension contracts;
- enforcement of the canonical Workflow definitions, command preconditions, and legal transitions owned by the Workflow domain;
- capability identity, required inputs and outputs, evidence requirements, and provider-independent failure semantics;
- validation of canonical domain commands, production of Kernel decisions and events, and invariant enforcement;
- the distinction between proposed, authorized, persisted, executing, completed, failed, and evaluated work.

## Non-responsibilities

The Kernel does not own:

- queues, worker pools, polling, clocks, cron parsing, timers, backoff calculations, or dispatch;
- process startup, shutdown, restart, leader election, liveness, readiness, or deployment topology;
- database drivers, transactions, migrations, event brokers, or workflow-engine SDKs;
- model inference, coding-agent sessions, workspaces, provider APIs, or tool execution;
- UI, transport protocols, notifications, metrics exporters, or logging infrastructure;
- application use-case sequencing, authoritative-state loading, transaction coordination, or Runtime notification;
- automatic approval of actions or alteration of policy because an execution provider requests it.

The Kernel may define information these mechanisms must preserve, but it cannot depend on their concrete implementations.

## Kernel decisions and effects

A Kernel operation consumes a command plus an explicitly loaded authoritative state snapshot or an explicit absent-aggregate expectation for creation. It returns either:

- a rejection with stable domain reasons and no state change; or
- a decision containing the next authoritative state, domain events, and requested effects.

Requested effects describe intent; they are not proof that work occurred. Persistence and effect dispatch belong outside the Kernel. A Runtime cannot turn a rejected command into a legal transition by retrying it.

## First aggregate boundary

The first implemented aggregate root is one [Workflow](../domain/workflow.md). Its boundary contains the Workflow identity, Organization, WorkflowDefinition/version, Objective reference, current organizational state, accepted inputs, wait or terminal reason, correlation/causation identities, and opaque monotonic Workflow version.

Objective, Organization, Principal, GovernanceDecision, Approval, CapabilityDefinition, ResourceConstraint, Result, Runtime execution state, checkpoint, Artifact, Evidence, Metric, and Knowledge records remain external aggregates or records referenced by immutable identity/version. The Workflow aggregate cannot mutate them.

For first-slice Workflow commands, proposal validation enforces the preconditions owned by the canonical [Workflow lifecycle](../domain/workflow.md#first-slice-commands-and-legal-transitions). It produces the immutable [GovernedCommandProposal](../domain/command.md#governedcommandproposal) without state, Events, or intent.

Final validation requires the unchanged proposal, current `ALLOW`, matching authoritative versions, and any applicable Approval evidence. Acceptance produces the transition and effects defined by the Workflow contract. A version mismatch, changed digest, inactive dependency, illegal state, or stale governance evidence rejects without a decision effect.

## Identity boundary

- Organizational workflow identity is assigned and interpreted by CompanyOS.
- A workflow execution attempt has a distinct Runtime identity.
- Provider workflow, run, job, step, thread, and task identifiers are correlation identifiers, not CompanyOS domain identity.
- Replaying or replacing an execution must not create a second organizational objective, approval, or action unless the Kernel explicitly authorizes one.

## Invariants

- Kernel decisions are deterministic for the same command, authoritative state, policy inputs, and declared time input.
- Only Kernel operations determine legal organizational state transitions.
- The Kernel never reads wall-clock time, network state, environment variables, or provider state implicitly.
- Departments extend CompanyOS through registered contracts; adding one does not require changing Kernel fundamentals.
- Departments request capabilities, never concrete providers.
- Agents and providers cannot directly mutate authoritative organizational state.
- Authorization and approval evidence required by a transition must exist before the transition is permitted.
- A requested effect is not recorded as completed until validated evidence is accepted through a Kernel operation.
- Runtime retries never repeat a non-idempotent organizational decision under a new identity.
- Unknown, stale, or conflicting state versions cause rejection rather than best-effort mutation.
- Provider-specific error types are translated before reaching Kernel rules.
- Kernel state contains organizational facts, not worker leases, queue offsets, sockets, or process health.

## Relationship to Application, Runtime, and Daemon

The [Application layer](application.md) loads authoritative state and first invokes the Kernel's non-mutating proposal validation. Governance evaluates that exact proposal. Only on current `ALLOW` does Application invoke the Kernel's final decision and coordinate atomic persistence of accepted state, domain events, and caused execution intent. The [Runtime](runtime.md) executes only persisted intent and returns evidence through a new Application use case. The [Daemon](daemon.md) keeps Runtime components available but has no path around Application orchestration or Kernel decisions.

The OSS evidence matrix is maintained in [Runtime: OSS evidence](runtime.md#oss-evidence). Those systems largely illuminate execution mechanics; none is adopted as CompanyOS domain authority.

## Open questions

- OPEN QUESTION: Which policy results are inputs to Kernel legality versus decisions produced by Governance?
- OPEN QUESTION: Which workflow definition changes are compatible with already-running organizational workflows?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [System context](system-context.md)
- [Workflow](../domain/workflow.md)
- [Organization](../domain/organization.md)
- [Objective](../domain/objective.md)
- [Department](../domain/department.md)
- [Capability](../domain/capability.md)
- [Command](../domain/command.md)
- [Policy](../domain/policy.md)
- [Approval](../domain/approval.md)
- [Event](../domain/event.md)
- [Resource](../domain/resource.md)
