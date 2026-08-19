# Resource Domain

Status: DRAFT

## Ownership

Finance is the organizational owner of `ResourceConstraint` and its specializations. This document is their canonical domain contract. Governance, Runtime, workspaces, Intelligence, Coding Agents, departments, and providers may enforce or consume constraints; they cannot redefine their meaning, units, lifecycle, or overrun behavior.

Resource constraints limit otherwise eligible work. They do not authorize an action, make a domain transition legal, select a provider, or replace a Budget.

## Type hierarchy

```text
ResourceConstraint
├── CostConstraint
├── ComputeConstraint
├── TimeConstraint
├── StorageConstraint
└── ConcurrencyConstraint
```

Every constraint uses the base contract plus exactly one specialization. A request may carry a versioned `ResourceConstraintSet` containing several non-conflicting constraints.

## ResourceConstraint

The immutable base contract contains:

- stable constraint identity, version, type, and Finance owner;
- organization and budget/account scope;
- subject scope such as objective, department, workflow, capability, request, execution, attempt, provider profile, or workspace;
- quantity, canonical unit, comparison operator, and hard or advisory enforcement mode;
- applicability conditions, effective time, expiry, and accounting period;
- reservation requirement and permitted estimate uncertainty;
- warning and exhaustion thresholds;
- action on approach, exhaustion, breach, and unavailable measurement;
- source Budget, policy, approval, exception, and PriceProfile references;
- issuer, provenance, recorded time, status, and supersession links.

Constraints are immutable once used by a routing or execution decision. A changed limit, scope, unit, enforcement mode, or behavior creates a new version.

## CostConstraint

A monetary ceiling or range. It adds currency, maximum estimated cost, maximum actual cost, price and exchange-rate profile references, budget account, reservation/contingency amount, allocation treatment, and reconciliation tolerance.

CostConstraint contains money only. Tokens, CPU, memory, elapsed time, storage, or concurrency remain separate constraints even when Finance later converts their usage into cost.

## ComputeConstraint

A ceiling on metered computation. It adds one or more canonical dimensions such as input/output tokens, accelerator time, CPU time, memory quantity, invocation count, or provider compute units; measurement source; aggregation rule; burst allowance; and enforcement granularity.

Provider-specific units require a versioned Finance normalization or remain explicitly provider-specific. A router cannot silently compare incompatible compute units.

## TimeConstraint

A ceiling or required window for resource consumption such as maximum wall-clock execution duration, billable duration, or cumulative attempt time. It adds clock basis, start/stop conditions, pause treatment, retry aggregation, deadline reference, and tolerance.

Finance owns the resource meaning and limit. Runtime owns timers, timeouts, scheduling, and enforcement mechanics. A business due date or workflow schedule is not a TimeConstraint unless explicitly translated by an owning use case.

## StorageConstraint

A ceiling on retained or transient storage. It adds storage class, byte unit, region or residency restriction, retention interval, snapshot/cache/artifact inclusion rules, encryption class, and cleanup behavior.

Security, Knowledge, Evidence, and artifact retention requirements may establish minimum retention that a cost-saving constraint cannot override. Conflicts fail closed and require an explicit governed resolution.

## ConcurrencyConstraint

A ceiling on simultaneous resource use. It adds concurrency subject, counting key, maximum active count, fairness scope, queue behavior, burst policy, and lease/slot expiry rules.

Finance owns the allocated limit and accounting meaning. Runtime or WorkspaceManager owns leases, queues, and enforcement mechanics. ConcurrencyConstraint cannot define workflow priority or legal execution order.

## ResourceConstraintSet

A `ResourceConstraintSet` identifies the exact constraint versions applicable to one request or intent, its resolution timestamp, Finance evidence version, and any governed exception. Resolution composes constraints using the most restrictive compatible bound unless an approved domain rule states otherwise.

Conflicting units, scopes, time windows, or enforcement instructions produce an invalid set; routers and executors cannot guess. Absence of a required constraint or stale Finance evidence produces ineligibility or explicit approval according to policy, never an unlimited default.

## Reservation and reconciliation

A `ResourceReservation` holds capacity against a Budget for one logical operation and constraint set. It records reservation identity, estimates, contingency, expiry, status, and idempotency identity. Reservation is not expenditure and not authorization.

Before governed execution, the Application layer records required Finance reservation evidence with the intent. Runtime and adapters report `ResourceUsage`; Finance reconciles usage against the reservation and constraint versions. Retries and fallbacks preserve the logical-operation identity while reporting distinct attempts.

If measurement or billing outcome is indeterminate, the reservation remains held or enters reconciliation according to Finance policy. A router cannot free, enlarge, or transfer it to make another candidate eligible.

## Enforcement outcomes

Constraint evaluation returns one of:

- `WITHIN_LIMIT`: all required evidence is current and the proposed or observed use satisfies the constraint;
- `APPROACHING_LIMIT`: still eligible, with the declared warning action;
- `LIMIT_EXCEEDED`: hard limit blocks new consumption and triggers the declared stop/reconcile behavior;
- `APPROVAL_REQUIRED`: only when applicable policy permits a governed exception path;
- `MEASUREMENT_UNAVAILABLE`: required usage or estimate evidence is absent or unverifiable;
- `NOT_APPLICABLE`: the constraint does not cover the evaluated subject.

These are Finance constraint outcomes, not Governance decisions. `APPROVAL_REQUIRED` must be submitted to Governance and never counts as `ALLOW`.

## Invariants

- Finance owns ResourceConstraint definitions, versions, composition, reservations, and reconciliation semantics.
- Routers consume exact constraint versions and Finance outcomes; they never create, reinterpret, relax, or transfer them.
- Resource limits constrain eligible work but grant no authority and legalize no transition.
- Cost, compute, time, storage, and concurrency remain distinct even when correlated.
- Hard constraints filter candidates before optimization; scoring cannot compensate for a breach.
- Units, scope, applicability, evidence, enforcement mode, and breach behavior are explicit.
- Budget, constraint, reservation, usage, actual cost, and forecast are distinct records.
- Retry and fallback consumption is included under the original logical-operation scope unless Finance explicitly issues a new governed constraint set.
- Unknown, stale, conflicting, or unavailable required resource evidence fails closed according to policy.
- Constraint and reservation state is persisted before dependent consumption proceeds.

## OPEN QUESTIONS

- Which canonical units and conversion authorities apply to each resource type initially?
- Which constraints are mandatory for the first Intelligence and Coding Agent slices?
- Which advisory limits may continue after a warning without approval?
- How are shared caches, infrastructure, and human-review resources allocated?
- What reservation expiry and reconciliation rules apply after indeterminate provider outcomes?

## Dependencies

- Future Budget, PriceProfile, ResourceUsage, and Principal domain definitions
