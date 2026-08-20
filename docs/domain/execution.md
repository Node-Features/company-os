# Execution Domain

Status: APPROVED

## Definition

This is the canonical minimum contract for Runtime execution-state mechanics: attempts, leases, checkpoints, waits, retries, and resume. It is Runtime-owned execution bookkeeping, persisted through [Persistence](../architecture/persistence.md#required-persistence-ports) execution-state ports — it is not Kernel-decided organizational truth, and it cannot substitute for authoritative Workflow, Objective, or other aggregate state.

[Kernel](../architecture/kernel.md), [Workflow](workflow.md), and [Workspace](workspace.md#lifecycle) already assume these mechanics exist ("Runtime owns lease, checkpoint, wait, retry, and attempt mechanics and their execution-state persistence" — `workspace.md`) without a shared contract defining their minimum fields. This document is that contract. Subject-specific mechanics (for example, `EngineeringWorkspace` leases and checkpoints) may specialize these shapes; they do not redefine identity, fencing, or versioning rules.

## Minimum contracts

### ExecutionAttempt

One bounded try of a persisted [`ExecutionIntent`](workflow.md#minimum-contracts) or equivalent capability dispatch. It contains:

- stable attempt identity and organization;
- the referenced ExecutionIntent (or equivalent request) identity and the workflow/state version it was issued against;
- CapabilityRequest reference identifying what is being dispatched;
- logical-operation identity (stable idempotency identity shared across retries of the same operation) and attempt number within it;
- current Lease reference while claimed;
- current status (see Lifecycle);
- checkpoint reference when the attempt has produced recoverable progress;
- provider run ID or other correlation identifiers, which are correlation data, never CompanyOS domain identity;
- created, last-heartbeat, next-attempt-due, and terminal timestamps;
- terminal Result reference once the attempt reaches an outcome the Result domain records.

An ExecutionAttempt is immutable once terminal. A retry does not mutate a terminal attempt; it creates a new ExecutionAttempt under the same logical-operation identity.

### Lease

A time-bounded, fenced claim of ownership over one leasable subject (an ExecutionAttempt, a Workspace, or another Runtime-owned resource). It contains:

- lease identity, subject type, and subject reference;
- owning worker/process identity;
- monotonic fencing token;
- expiry and heartbeat/renewal policy;
- creation, last-renewal, and release/expiry timestamps.

A stale or expired fencing token is rejected by every subsequent operation against that subject; an expired worker cannot commit late progress. Renewal issues a new expiry under the same fencing token lineage; it does not change subject identity or ownership silently.

### Checkpoint

Versioned, integrity-checked recovery material for one ExecutionAttempt. It contains:

- checkpoint identity, organization, and schema/compatibility version;
- the workflow ID/version and execution/attempt ID it was captured against;
- an integrity digest over its recovery payload;
- an opaque recovery payload reference, interpreted only by the owning Runtime adapter or subject-specific specialization (for example, the workspace-spec version, environment digest, and repository state a `EngineeringWorkspace` checkpoint records);
- a capture timestamp and resumption cursor/sequence marker.

A checkpoint recovers execution mechanics; it never becomes authoritative business state. Resumption rejects a checkpoint that is stale, corrupt, incompatible, or bound to a different execution.

### Wait

Persisted state plus a wake condition — not a sleeping process or a held worker. It contains:

- wait identity and the ExecutionAttempt (or Workflow-level wait) it applies to;
- wake-condition type: durable due time, external event/signal, approval resolution, or dependency completion;
- the due time or event/dependency reference that satisfies it;
- created and satisfied/cancelled timestamps;
- the wake evidence reference once satisfied.

Waiting consumes no dedicated worker and survives process restart. A due time is a durable value evaluated on wake, not a live timer; wall-clock polling is only a wake mechanism, never the source of truth for whether a wait is satisfied.

### Retry

Retry mechanics reuse the logical-operation identity while issuing a new ExecutionAttempt identity. A retry record references:

- the logical-operation identity being retried and the terminal (failed-retryable) ExecutionAttempt that preceded it;
- the retry policy applied, which is declared per operation/capability type rather than globally — exact attempt limits, backoff basis, and jitter are owned by the applicable CapabilityDefinition or operation-class policy, not by this contract;
- the failure classification that authorized the retry (retryable, per [Runtime failure semantics](../architecture/runtime.md#failure-semantics));
- elapsed-time and attempt-count accounting against the applicable policy bound.

A retry cannot relax the inputs, constraints, acceptance criteria, authority requirements, or failure meaning of the operation being retried.

### Resume

Resume is a behavior, not a separate record: reconstructing ExecutionAttempt, Lease, and Wait/Checkpoint state after a process restart or wake-up. Resume:

1. reloads persisted ExecutionAttempt, Lease, and applicable Checkpoint state — never agent conversation memory or provider session state;
2. verifies the current fencing token is still valid before acting;
3. verifies checkpoint compatibility (workflow, execution, and authoritative-state versions) before continuing dispatch;
4. is safe under duplicate wake-ups: waking an already-progressed or already-terminal attempt is a no-op against that attempt's persisted status.

## Lifecycle

ExecutionAttempt legal transitions:

```text
CLAIMED -> DISPATCHED
DISPATCHED -> WAITING | SUCCEEDED | FAILED_RETRYABLE | FAILED_TERMINAL | CANCELLED
WAITING -> DISPATCHED | CANCELLED
CLAIMED -> LEASE_EXPIRED
DISPATCHED -> LEASE_EXPIRED
WAITING -> LEASE_EXPIRED
```

- `CLAIMED`: a bounded Lease was obtained over the due persisted ExecutionIntent and this attempt was created or resumed.
- `DISPATCHED`: a capability request was sent to an eligible executor; heartbeats and evidence accumulate here.
- `WAITING`: a durable Wait condition is pending; no worker is held.
- `SUCCEEDED` / `FAILED_TERMINAL` / `CANCELLED`: terminal; a terminal Result reference is recorded through the owning Application use case.
- `FAILED_RETRYABLE`: terminal for this attempt only; a new ExecutionAttempt may be created under the same logical-operation identity, subject to retry policy.
- `LEASE_EXPIRED`: the Lease expired or was fenced out before reaching a terminal outcome; recovery treats this as an indeterminate attempt outcome to be reconciled, never as success.

Terminal states are immutable. Reaching one does not itself advance Workflow, Objective, or other authoritative state — only an Application/Kernel-accepted transition using the resulting Result can do that.

## Invariants

- ExecutionAttempt, Lease, Checkpoint, Wait, and Retry state are Runtime execution mechanics, never inferred as authoritative organizational state.
- A worker crash cannot lose a persisted ExecutionIntent, and an unpersisted result never becomes authoritative.
- At-least-once dispatch and delivery are assumed; duplicate claims, dispatches, or wake-ups cannot duplicate a legal organizational transition.
- Every operation against a leased subject is rejected if its fencing token is stale or expired.
- A logical-operation identity is stable across retries; each retry receives a new, distinct ExecutionAttempt identity.
- A checkpoint is accepted for resume only when its workflow, execution, and state versions match the resuming context.
- Recovery is driven from persisted ExecutionAttempt, Lease, and Checkpoint records, never from agent conversation memory or provider session state alone.
- Retry policy specifics (limits, backoff, jitter) are owned by the applicable capability or operation-class policy, not invented ad hoc by a Runtime adapter.
- `LEASE_EXPIRED` and `FAILED_RETRYABLE` outcomes never imply success and never authorize skipping reconciliation.

## OPEN QUESTIONS

- What is the minimum checkpoint envelope and compatibility/versioning policy for the first Runtime implementation? (Also tracked in [Runtime — Open questions](../architecture/runtime.md#open-questions).)
- What default attempt limits, backoff basis, and jitter apply per capability class absent an explicit policy?
- Which subject types beyond ExecutionAttempt and Workspace require their own Lease specialization?
- What is the minimum reconciliation procedure for a `LEASE_EXPIRED` attempt whose external effect outcome is unknown?

## Dependencies

- [Workflow](workflow.md)
- [Capability](capability.md)
- [Result](result.md)
- [Workspace](workspace.md)
- [Organization](organization.md)
- [Runtime](../architecture/runtime.md)
- [Persistence](../architecture/persistence.md)
