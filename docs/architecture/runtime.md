# CompanyOS Workflow Runtime

Status: DRAFT

## Responsibility

The Runtime executes CompanyOS workflows without owning their organizational meaning. It converts persisted, Kernel-authorized work into resumable execution and reports evidence back through controlled Application use cases.

The Runtime owns:

- workflow execution instances and execution-attempt identity;
- dispatch to registered capability executors and workers;
- scheduling mechanics, durable timers, waiting, wake-up, cancellation, and resume;
- retry classification, attempt limits, backoff, jitter, timeouts, and dead-letter or terminal-failure routing;
- checkpoint coordination and recovery after worker or process failure;
- correlation among CompanyOS workflow identity, execution identity, provider identifiers, and attempts;
- idempotency keys, leases, heartbeats, concurrency limits, and duplicate-delivery handling;
- execution events and evidence needed for audit and metrics;
- adapters for an internal or external workflow engine.

## Non-responsibilities

The Runtime does not own:

- mission, policy meaning, objective semantics, department authority, capability meaning, or legal domain transitions;
- approval decisions or the classification of actions as automatic, approval-required, or human-only;
- authoritative organizational schemas or persistence technology;
- continuous process supervision, deployment, or machine lifetime;
- provider selection policy, agent reasoning, workspace isolation policy, or external-service semantics;
- treating its checkpoint, event history, queue, or provider run as authoritative organizational state.
- application request validation, authoritative-state loading, Governance orchestration, Kernel invocation, or atomic business-state commits.

## Execution model

1. An [Application use case](application.md) loads authoritative state, asks the Kernel for non-mutating proposal validation, submits the exact proposal to Governance, and only on current `ALLOW` asks the Kernel for its final decision before atomically persisting accepted state, domain events, and execution intent.
2. After commit, the Application layer notifies Runtime; notification is only a wake-up hint because the persisted intent remains discoverable.
3. Runtime claims the due persisted intent using a bounded lease and creates or resumes an execution attempt.
4. Runtime dispatches a capability request to an eligible executor through a provider-independent contract.
5. Results, failures, heartbeats, and provider identifiers are captured as execution evidence.
6. Runtime submits the result and execution evidence through a new Application use case.
7. The Application layer reloads state, obtains a Kernel-validated proposal for interpreting the result, submits that exact proposal to Governance, and only on current `ALLOW` asks the Kernel for its final decision and persists any authoritative transition before dependent work is scheduled.
8. Transient failures produce a durable next-attempt time; permanent or exhausted failures produce a terminal execution result for Kernel interpretation.

Waiting is persisted state plus a wake condition, not a sleeping process. Resume reconstructs execution from persisted state and is safe after duplicate wake-ups.

## State and identity

| Concept | Owner | Meaning |
|---|---|---|
| Organizational workflow ID | Kernel/domain | Stable identity of the governed organizational process |
| Execution ID | Runtime | One resumable execution of that process |
| Attempt ID | Runtime | One bounded try of a step or capability dispatch |
| Provider run ID | External adapter | Correlation with a provider; never domain identity |
| Checkpoint | Runtime/persistence contract | Recovery material for execution mechanics |
| Authoritative state | Kernel/domain via persistence | Organizational truth; never inferred solely from a checkpoint |

## Failure semantics

- Every failure is classified as retryable, non-retryable, cancelled, timed out, lease-lost, policy-denied, approval-required, or unknown before progression.
- Unknown failures fail safely and do not authorize a transition.
- Retry policy is explicit, bounded, observable, and attached to an operation type; retries preserve idempotency identity.
- A worker crash cannot lose a persisted intent or make an unpersisted result authoritative.
- At-least-once delivery is assumed at every queue and callback boundary.
- Runtime records late and duplicate results but accepts them only when the execution version and lease permit it.
- Recovery is driven from persisted intent, checkpoints, and execution records, never agent conversation memory.

## Invariants

- Persistence of accepted state and execution intent succeeds before external execution begins.
- Runtime notification, queue delivery, or polling never substitutes for persisted execution intent.
- Runtime cannot create, waive, or reinterpret Kernel legality, policies, capabilities, or approvals.
- A legal transition is applied once even when dispatch or delivery occurs more than once.
- Waiting consumes no dedicated worker and survives process restart.
- Timers use durable due times; wall-clock polling is only a wake mechanism.
- Worker ownership is leased and expires; no worker owns durable truth.
- Checkpoints are versioned and correlated with the workflow and execution versions that produced them.
- A retry reuses the logical operation identity while receiving a new attempt identity.
- Side effects require idempotency support, reconciliation, or explicit compensation semantics.
- Cancellation is durable intent; completion races are resolved by persisted versions and Kernel rules.
- Provider history may aid recovery but cannot replace CompanyOS authoritative persistence.
- Runtime adapters expose equivalent CompanyOS semantics or declare unsupported behavior explicitly.

## OSS evidence

The links below target the revisions pinned in [`references.lock.md`](../references/references.lock.md), except Temporal samples, whose inspected revision was `7fddb7e772d2f4f2f1941c9d3e99216620e0af3b`.

| Project and relevant files | Core abstraction and models | Borrow / reject |
|---|---|---|
| **Temporal TypeScript:** [`workflow-client.ts`](https://github.com/temporalio/sdk-typescript/blob/6df7e47797eab21ebd4644d0a2f5365a44032025/packages/client/src/workflow-client.ts), [`worker.ts`](https://github.com/temporalio/sdk-typescript/blob/6df7e47797eab21ebd4644d0a2f5365a44032025/packages/worker/src/worker.ts), [`workflow.ts`](https://github.com/temporalio/sdk-typescript/blob/6df7e47797eab21ebd4644d0a2f5365a44032025/packages/workflow/src/workflow.ts), worker and workflow tests | Workflow ID/run ID, deterministic replay, task queues, workers, durable history, timers, signals, cancellation, activity retry. Service persists history; workers are disposable and recover by replay. | **Borrow:** stable identity, durable waiting, worker/activity separation, explicit retry and cancellation. **Reject:** making Temporal history the organizational source of truth or leaking SDK types into Kernel/domain. |
| **Temporal samples:** [`sleep-for-days`](https://github.com/temporalio/samples-typescript/tree/7fddb7e772d2f4f2f1941c9d3e99216620e0af3b/sleep-for-days), [`schedules`](https://github.com/temporalio/samples-typescript/tree/7fddb7e772d2f4f2f1941c9d3e99216620e0af3b/schedules), [`worker-versioning`](https://github.com/temporalio/samples-typescript/tree/7fddb7e772d2f4f2f1941c9d3e99216620e0af3b/worker-versioning), [`continue-as-new`](https://github.com/temporalio/samples-typescript/tree/7fddb7e772d2f4f2f1941c9d3e99216620e0af3b/continue-as-new) | Executable examples show timers versus conditions, client-assigned workflow IDs, worker task queues, scheduled starts, long-history rollover, and version-aware recovery. Persistence is delegated to Temporal service history. | **Borrow:** testable lifecycle examples and time-skipping tests. **Reject:** sample defaults as production policy and cron as domain semantics. |
| **LangGraph.js:** [`pregel/index.ts`](https://github.com/langchain-ai/langgraphjs/blob/a86f813954e010fbf30711c37baa5c53444613d5/libs/langgraph-core/src/pregel/index.ts), [`checkpoint/base.ts`](https://github.com/langchain-ai/langgraphjs/blob/a86f813954e010fbf30711c37baa5c53444613d5/libs/checkpoint/src/base.ts), [`pregel/retry.ts`](https://github.com/langchain-ai/langgraphjs/blob/a86f813954e010fbf30711c37baa5c53444613d5/libs/langgraph-core/src/pregel/retry.ts), [`interrupt.ts`](https://github.com/langchain-ai/langgraphjs/blob/a86f813954e010fbf30711c37baa5c53444613d5/libs/langgraph-core/src/interrupt.ts), checkpoint and retry tests | Pregel-style supersteps over channels; thread/checkpoint identity; interrupts and commands resume graph state; pluggable checkpointers persist snapshots and pending writes; node retries govern failures. | **Borrow:** explicit checkpoint interface, interrupt/resume, versioned state snapshots. **Reject:** graph/thread state as organizational truth, node topology as CompanyOS workflow legality, and in-memory saving for durable operation. |
| **Inngest:** [`executor`](https://github.com/inngest/inngest/tree/1f91829a35cccf2372768fef4aa275f56fbd4843/pkg/execution/executor), [`queue`](https://github.com/inngest/inngest/tree/1f91829a35cccf2372768fef4aa275f56fbd4843/pkg/execution/state/redis_state), [`state`](https://github.com/inngest/inngest/tree/1f91829a35cccf2372768fef4aa275f56fbd4843/pkg/execution/state), [`retry.go`](https://github.com/inngest/inngest/blob/1f91829a35cccf2372768fef4aa275f56fbd4843/pkg/util/retry.go) | Event-triggered function runs; runner creates run state; queues apply flow control; executor writes incremental step state and schedules retries; pauses are matched by events. Failures carry retryability and retry-after data. | **Borrow as concepts only:** persisted intent before enqueue, fair flow control, event-resumable waits, explicit retry classification. **Reject:** source reuse without license approval, event stream as authority, and coupling CompanyOS contracts to Inngest functions/steps. |
| **Trigger.dev:** [`runEngine`](https://github.com/triggerdotdev/trigger.dev/tree/b93904526c6f7181c9381d3483d135d915c61aa6/apps/webapp/app/runEngine), [`timerWheel.ts`](https://github.com/triggerdotdev/trigger.dev/blob/b93904526c6f7181c9381d3483d135d915c61aa6/apps/supervisor/src/services/timerWheel.ts), [`worker heartbeat route`](https://github.com/triggerdotdev/trigger.dev/blob/b93904526c6f7181c9381d3483d135d915c61aa6/apps/webapp/app/routes/engine.v1.worker-actions.heartbeat.ts), and snapshot attempt/suspend routes under [`routes`](https://github.com/triggerdotdev/trigger.dev/tree/b93904526c6f7181c9381d3483d135d915c61aa6/apps/webapp/app/routes) | Task runs dispatched to managed workers with queues, concurrency, retries, waits/checkpoints, heartbeats, and deployment/version coordination. Durable platform records support recovery and observability. | **Borrow:** deployment-aware workers, heartbeats, concurrency controls, resumable waits. **Reject:** assuming hosted task semantics are organizational semantics or requiring a deployment platform before a slice demonstrates need. |
| **JARVIS:** [`src/daemon`](https://github.com/vierisid/jarvis/tree/6e144520c747a6e0b8673ba9b75769d5d5f10a9c/src/daemon), [`src/workflows`](https://github.com/vierisid/jarvis/tree/6e144520c747a6e0b8673ba9b75769d5d5f10a9c/src/workflows), [`WORKFLOW_AUTOMATION.md`](https://github.com/vierisid/jarvis/blob/6e144520c747a6e0b8673ba9b75769d5d5f10a9c/docs/WORKFLOW_AUTOMATION.md), authority tests | A persistent Bun daemon starts background services and an embedded visual workflow engine; SQLite stores system/run data; triggers initiate work; retry/fallback/self-heal handle failures. Process and workflow concerns are comparatively integrated. | **Borrow as evidence:** explicit always-on lifecycle, startup coordination, health/kill controls, authority gating. **Reject:** a monolithic daemon as domain authority, agent self-heal changing workflow legality, and code reuse while its source-available license remains incompatible/unapproved. |

No inspected project is selected as a dependency. Temporal offers the strongest evidence for durable execution semantics; LangGraph.js is narrower graph/checkpoint evidence; Inngest and Trigger.dev inform queue and worker operations; JARVIS informs always-on product lifecycle.

## Open questions

- OPEN QUESTION: Will the first Runtime be an internal implementation or an adapter over a durable engine?
- OPEN QUESTION: What transaction/outbox boundary atomically persists domain state and execution intent?
- OPEN QUESTION: What is the minimal checkpoint envelope and compatibility/versioning policy?
- OPEN QUESTION: Which operations require compensation versus idempotent retry?
- OPEN QUESTION: What maximum attempts, elapsed time, and backoff defaults apply by capability class?
- OPEN QUESTION: How are externally completed operations reconciled after timeout or cancellation?
- OPEN QUESTION: What fairness and concurrency guarantees are required across organizations, departments, and objectives?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [System context](system-context.md)
- [Application layer](application.md)
- [Persistence](persistence.md) execution-state port
- [Events](events.md)
- [Workflow and execution-intent contracts](../domain/workflow.md)
- [Result](../domain/result.md)
- [Evidence](../domain/evidence.md)
- Future Capability domain contract
