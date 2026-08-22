# CompanyOS Node

Status: DRAFT

## Responsibility

A CompanyOS Node is a runtime instance that owns resources, executes workloads, participates in the CompanyOS protocol, and communicates with other nodes through authenticated messages. A Node is the **execution boundary** — the addressable unit a scheduler places work onto — not an organizational unit and not a physical-machine synonym.

This is the standard distributed-systems sense of "node" (an independently addressable computational participant with its own resources), made specific to CompanyOS: a Node advertises what it can do (capabilities) and what it has (resources), and the Kernel/scheduler resolves workload requirements against the current Node population rather than a workload declaring which Node it runs on.

A Node owns:

- its own identity, trust level, and authentication material for node-to-node and node-to-control-plane messages;
- its resource inventory (CPU, memory, GPU, storage, network) and current utilization;
- its advertised capabilities (e.g. `inference`, `research`, `data-processing`, `workflow-execution`) — a declaration of what classes of workload it is eligible to run, not a binding to any one workload forever;
- accepting, executing, and reporting on workloads placed on it by the scheduler;
- its own health/liveness signal and runtime version.

A Node does not own:

- organizational meaning (mission, department charter, policy) — see [Non-responsibilities](#node--these-things) below;
- the decision that a workload *should* run — Governance and the Kernel decide legality; a scheduler decides placement; the Node only executes what it is handed;
- other nodes' state, other than what it learns through authenticated protocol messages.

## Node ≠ these things

The word "node" collides easily with other CompanyOS and infrastructure vocabulary. These are deliberately **not** synonyms:

| Not a Node | Why |
|---|---|
| **Agent** | An Agent is organizational intelligence — a role-scoped reasoning/acting identity (e.g. Research Agent, Finance Agent). Agents run *as workloads on* Nodes; a Node has no organizational role of its own. |
| **Department** | Departments are an organizational topology (mission, roles, policy, capability *contracts* — see [`departments.md`](departments.md)). Nodes are a compute topology (physical/virtual resource + capability *inventory*). A department's agents and workloads may run on any eligible Node; a department does not own a Node. |
| **Workflow** | A Workflow is a governed organizational process instance ([`domain/workflow.md`](../domain/workflow.md)). A Workflow's execution produces workloads that a scheduler places on Nodes; the Workflow itself has no compute identity. |
| **Process / Container** | A process or container is one way a Node's runtime might be packaged and deployed. A Node is the logical participant; a process or container is (part of) its physical realization. |
| **Server** | A physical or virtual server can *host* a Node, and could host more than one. The Node is the addressable participant in the CompanyOS protocol; the server is infrastructure underneath it. |
| **[`Workspace`](workspaces.md)** | A Workspace is a task-scoped, isolated execution environment provisioned for one Engineering task and destroyed after — ephemeral, single-purpose. A Node is a longer-lived participant that may host many Workspaces (or other workloads) over its lifetime, across many tasks. |

The important part, per the architecture this generalizes from: the Node is the execution boundary, not the organizational department, workflow, or agent.

## Node is a capability, not a permanent role

A Node should not be permanently tied to one function (e.g. "the Research node"). Instead, the Kernel/scheduler sees a Node as a bundle of currently-advertised capabilities plus currently-available resources:

```
Node
├── resources:  cpu, memory, gpu, storage, network
├── capabilities:  [inference, research, data-processing, workflow-execution, ...]
├── workloads:  [currently placed work]
└── identity, trust_level, status, health, runtime_version
```

A scheduler request is then framed as *"place this workload on a Node that has capability X and capacity Y,"* never *"run this on Node A specifically."* This mirrors a pattern CompanyOS has already accepted for a narrower case: [`intelligence.md`](intelligence.md)'s Intelligence Router does exactly this — departments/agents describe the outcome and constraints they require, never select a concrete provider or model directly, and eligibility filtering + multi-factor routing resolves the request against registered Model/Provider Profiles. This document generalizes that same accepted shape from "which model/provider" to "which Node," rather than introducing an unrelated pattern.

Consequences:

- A workload is not permanently bound to the Node that first ran it. The scheduler may place a retry, a continuation, or a related workload on a different eligible Node.
- Losing a Node degrades capacity, not organizational function, as long as another Node advertises the same capability — this is where elasticity and fault tolerance come from.
- Organizational topology (Company → Organization → Department → Agent → Workload) and compute topology (Runtime → Node → Node → Node) are tracked separately. A Department's Workload requiring capability `treasury-analysis` might land on Node 2 today and Node 7 tomorrow; the Department does not track or care which.

## Node identity (draft shape)

```
Node
├── node_id            — stable identity, assigned once
├── identity            — authentication material for node-to-node / node-to-control-plane messages
├── trust_level          — OPEN QUESTION: see below
├── status              — e.g. healthy, degraded, draining, unreachable
├── resources            — cpu, memory, gpu, storage, network (advertised + currently available)
├── capabilities         — declared classes of workload this Node is eligible to run
├── workloads            — currently placed work (identity only; workload semantics live elsewhere)
├── location             — OPEN QUESTION: region/zone/rack, or just an opaque placement hint?
├── health               — liveness signal, last heartbeat
└── runtime_version       — the Node runtime's own version, for rollout/compatibility
```

This is a draft shape, not a schema decision — see [Open questions](#open-questions).

## Relationship to existing boundaries

- **[Kernel](kernel.md)**: continues to own organizational legality (what transition is allowed) and does not gain node-placement responsibility. Whether "the scheduler" is a new boundary, a Runtime sub-responsibility, or a distinct component is unresolved (see below) — but it is not the Kernel's concern either way, matching [`kernel.md`](kernel.md)'s existing non-responsibilities for execution mechanics.
- **[Runtime](runtime.md)**: already owns "dispatch to registered capability executors and workers" and "scheduling mechanics" in the existing first-slice scope, but today that means claiming a due `ExecutionIntent` and calling a fixed, in-process `ProviderAdapter` (ADR-0004) — there is no Node population to choose among yet. A capability-based scheduler as described here is a significant expansion of Runtime's dispatch step, not something the current implementation does.
- **[Daemon](daemon.md)**: today supervises one co-located process (ADR-0004; `daemon.md`'s own open question is "single process vs. separate deployment boundaries"). A world with multiple Nodes most likely means multiple Daemon-supervised runtime instances — this document does not resolve that open question, but it sharpens it: a Node, concretely, is probably realized as one Daemon-supervised runtime process (or a small number of them).
- **[Workspaces](workspaces.md)**: a Workspace's `WorkspaceProvider` already provisions an isolated environment *somewhere*; a Node is a candidate answer for "where," but `workspaces.md` does not currently model Nodes and is not changed by this draft.
- **[Departments](departments.md)**: unaffected. Departments remain a pure organizational topology; this document is explicit that Nodes must not become department-scoped compute (no "the Finance department's Node").
- **[Identity](identity.md)**: a Node's own authentication ("communicates with other nodes through authenticated messages") is a new Principal-adjacent concept `identity.md` does not currently cover (that doc's Principal kinds are Human/Agent/Service/Provider-oriented). Whether a Node needs its own Principal kind, or authenticates as a Service Principal, is an open question for that doc, not decided here.

## Failure semantics (draft)

- An unhealthy or unreachable Node must be excluded from scheduling before its declared capabilities are trusted again.
- Losing a Node must not lose a workload's authoritative record — this mirrors [`runtime.md`](runtime.md)'s existing invariant that a worker crash cannot lose a persisted intent; a Node is a worker in that same sense, just capability-addressed rather than fixed.
- A Node rejoining after a health flap must reconcile its current workload/resource state before being trusted with new placements, not assume its last-known state is still accurate.

## Invariants (draft)

- A Node's advertised capabilities are declarative and verified, not assumed; a scheduler does not place a workload on a Node for a capability it has not currently advertised.
- No workload is permanently bound to the Node that first executed it.
- No component grants a Node authority beyond what its trust level and authenticated identity permit, regardless of which process or machine it happens to run in.
- Organizational topology and compute topology are never merged into one hierarchy — a Department, Agent, or Workflow must not hard-reference a specific `node_id`.

## Open questions

- OPEN QUESTION: Does the first slice need multiple Nodes at all, given ADR-0004's single co-located process? If not, is this document scoped for a future phase, and which one?
- OPEN QUESTION: Where does "the scheduler" that resolves workload capability requirements against the Node population live — a new architectural boundary, or an expansion of Runtime's existing dispatch responsibility?
- OPEN QUESTION: What is `trust_level` — a fixed enum, a computed score, something Governance evaluates per placement? Who assigns and revokes it?
- OPEN QUESTION: What authenticates a Node's messages to the control plane and to other Nodes — a new Principal kind in `identity.md`, or reuse of an existing one (e.g. Service Principal)?
- OPEN QUESTION: What is the Node registry's consistency/storage model — authoritative persistence like every other CompanyOS record, or a separate operational store (e.g. a heartbeat-driven ephemeral registry) with different durability guarantees?
- OPEN QUESTION: What heartbeat/health-check protocol and failure-detection timeout apply, and how do they interact with Runtime's existing lease/fencing-token mechanics for in-flight work?
- OPEN QUESTION: Is `location` needed for the first Node model, and if so, at what granularity (region, zone, opaque placement hint)?
- OPEN QUESTION: How does a capability-based scheduler's placement decision get recorded as evidence for M&E, the way `intelligence.md`'s routing decisions already are?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [System context](system-context.md)
- [Kernel](kernel.md)
- [Runtime](runtime.md)
- [Daemon](daemon.md)
- [Departments](departments.md)
- [Workspaces](workspaces.md)
- [Identity](identity.md)
- [Model-independent intelligence](intelligence.md) — the accepted capability-routing pattern this document generalizes
