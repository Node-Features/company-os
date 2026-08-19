# Workspace Domain

Status: DRAFT

## Definition

A `Workspace` is a leased, organization-scoped execution boundary assigned to one declared purpose. An `EngineeringWorkspace` is the Workspace specialization assigned to one EngineeringTask attempt. This contract owns Workspace identity, lifecycle vocabulary, legal transitions, and invariants; architecture owns provider mechanisms, isolation controls, and operational coordination.

## Minimum contract

A Workspace contains:

- stable Workspace identity and organization;
- purpose, task, and attempt references;
- immutable specification version and provider-class requirement;
- repository and exact base-revision references when applicable;
- isolation, tool, network, credential, resource, evidence, and retention policy references;
- current lifecycle state and opaque version;
- active lease identity, owner, fencing token, and expiry when leased;
- provisioned-resource, checkpoint, cleanup-evidence, and failure references when applicable.

An EngineeringWorkspace additionally binds one engineering task version, attempt, repository, immutable base revision, and task-specific branch policy. It cannot be reassigned across organizations or unrelated tasks.

## Lifecycle

Legal transitions are:

```text
REQUESTED -> PROVISIONING
PROVISIONING -> READY | FAILED
READY -> LEASED
LEASED -> ACTIVE | SEALED
ACTIVE -> SUSPENDED | SEALED | FAILED
SUSPENDED -> ACTIVE
SEALED -> RETAINED | DESTROYING
RETAINED -> DESTROYING
FAILED -> DESTROYING
DESTROYING -> DESTROYED
```

- `REQUESTED`: an accepted Workspace specification exists; no provider resource is assumed.
- `PROVISIONING`: an authorized provider operation is in progress.
- `READY`: provider identity and required isolation controls have been verified.
- `LEASED`: one current attempt holds a fenced lease.
- `ACTIVE`: the bound session may perform permitted work.
- `SUSPENDED`: execution stopped at a verified checkpoint while identity and retention obligations remain.
- `SEALED`: execution stopped and mutable work is closed for result verification.
- `RETAINED`: sealed material is held under an explicit evidence or retention requirement.
- `FAILED`: the requested lifecycle operation failed; cleanup remains required where resources may exist.
- `DESTROYING`: credentials are revoked and resource cleanup is in progress.
- `DESTROYED`: cleanup evidence confirms the provider resource is no longer usable.

The Kernel enforces legal authoritative lifecycle transitions. Application coordinates their persistence. Runtime owns lease, checkpoint, wait, retry, and attempt mechanics and their execution-state persistence.

## Invariants

- One Workspace is bound to one organization, purpose, task/attempt where applicable, specification version, and current lifecycle version.
- One EngineeringWorkspace is bound to one immutable repository base revision and task attempt.
- A Workspace cannot become `READY` until required identity and isolation evidence is verified.
- `FAILED` does not imply `DESTROYED`; cleanup is explicit and evidenced.
- A stale or expired lease cannot authorize further Workspace activity.
- Resume cannot weaken the accepted specification, Governance decision, resource constraints, or isolation policy.
- Provider state and session history are observations, not authoritative Workspace lifecycle state.
- Every authoritative lifecycle transition is accepted and persisted before its dependent external action.
- Cleanup is idempotent, attributable, and auditable.

## Open questions

- OPEN QUESTION: Which lifecycle transitions require a fresh Governance evaluation by risk class?
- OPEN QUESTION: Under what verified conditions may an EngineeringWorkspace be reused across attempts for the same task?

## Dependencies

- [Organization](organization.md)
- [Principal](principal.md)
- [Resource](resource.md)
- [Evidence](evidence.md)
