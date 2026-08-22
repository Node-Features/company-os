# ADR-0005: Kernel Interface Contracts

Status: PROPOSED

## Context

[`docs/architecture/kernel.md`](../architecture/kernel.md) (`APPROVED`) already fixes, in prose, what the Kernel owns and does not own, as part of the layer boundary [`ADR-0001`](ADR-0001-kernel-runtime-daemon.md) established. It names five areas of Kernel ownership — Organization, Objective, Department, Workflow, Capability — each a distinct aggregate boundary per kernel.md's "First aggregate boundary" section. Only one, Workflow, has a Go implementation: `internal/kernel/workflow` (`proposal.go`, `decision.go`, `transitions.go`, `digest.go`), verified end-to-end against a real database and a real provider call (Phase 1, `ROADMAP.md`).

No document pins the exact Go-level contract — function or interface signatures, package layout, parameter and return shapes — that each aggregate's Kernel logic must expose. As Phase 3 (Organization), Phase 4 (Objective proposal gate), and department registration bring more aggregates under Kernel decision-making, each risks inventing its own ad hoc shape unless one contract is fixed now, before the code exists — mirroring how `docs/domain/workflow.md` and `docs/domain/command.md` pinned Workflow's shape before `internal/kernel/workflow` was written.

This ADR does not revisit what the Kernel owns — that is settled by kernel.md and ADR-0001. It fixes how that ownership is expressed in Go.

## Decision

### Ownership restated from kernel.md, not re-decided here

The Kernel owns exactly what [kernel.md](../architecture/kernel.md) already specifies:

| Area | Kernel owns | Package |
|---|---|---|
| Organization | identity, mission, vision, principles, policy semantics, legal transitions | `internal/kernel/organization` |
| Objective | identity, lifecycle, relationships, success criteria, legal transitions | `internal/kernel/objective` |
| Department | identity, responsibilities, authority boundaries, extension-contract validation | `internal/kernel/department` |
| Workflow | enforcement of canonical definitions, command preconditions, legal transitions | `internal/kernel/workflow` (implemented) |
| Capability | identity, required inputs/outputs, evidence requirements, failure semantics | `internal/kernel/capability` |

The Kernel does **not** own, per kernel.md's non-responsibilities and the layer that owns it instead:

| Not Kernel's | Actually owned by |
|---|---|
| Authority, policy evaluation, ALLOW / DENY / REQUIRE_APPROVAL | Governance ([governance.md](../architecture/governance.md)) |
| Scheduling, queues, timers, retries, dispatch | Runtime ([runtime.md](../architecture/runtime.md)) |
| Process lifetime, startup/shutdown, supervision | Daemon ([daemon.md](../architecture/daemon.md)) |
| Event routing, delivery, publication | Events architecture ([events.md](../architecture/events.md)) |
| State loading, transaction coordination, use-case sequencing | Application ([application.md](../architecture/application.md)) |
| Business logic (sales, marketing, finance decisioning) | Individual departments — Kernel validates a `DepartmentDefinition` registration, never a department's internal logic ([departments.md](../architecture/departments.md) dependency rule) |
| Individual agent prompts, model/provider selection | Intelligence routing and the Agent domain, not Kernel |
| UI, transport protocols | Application's HTTP adapters and `web` |
| Department-specific workflows | The owning department's own workflow contracts, coordinated only through shared events/capabilities per `departments.md` |

Both tables restate already-approved text; this ADR adds no new ownership claim.

### Contract shape: per-command functions, not a Go `interface`

The request behind this ADR asked for "the exact Go interface it exposes." The existing, shipped, tested `internal/kernel/workflow` package does not use a Go `interface` type — it exposes free functions, one pair per command type:

- `ValidateXProposal(cmd, ...current state...) (*command.GovernedCommandProposal, []string)` — non-mutating, called by Application before Governance evaluates.
- `FinalizeX(cmd, proposal, decision, ...current state..., declaredTime) (*command.KernelDecisionEnvelope, []string)` — mutating, called by Application only after Governance returns current `ALLOW`.

This is deliberate, not an omission: `ValidateCreateProposal`, `ValidateStartProposal`, and `ValidateResultProposal` each take genuinely different parameters — a fresh command has no current state to pass; a result-acceptance command needs the `Result` being accepted. Forcing these into one Go `interface` method would require an `any`-typed parameter, erasing the type safety the concrete signatures already provide. This ADR treats "one function-signature family per command type, in one package per aggregate" as the Kernel's Go contract, not the `interface` keyword — and flags this substitution explicitly rather than silently complying with the literal request.

Every Kernel decision function, implemented or proposed below:

- takes `declaredTime` as an explicit parameter and performs no I/O, no wall-clock reads, and no environment reads (kernel.md invariant; already enforced by comment in `internal/kernel/workflow`: "Kernel functions never read wall-clock time, network state, or environment variables implicitly");
- returns either rejection reasons (`[]string`, from a fixed vocabulary like `command.ReasonIllegalState`) or an accepted decision — never both, never a partial result;
- consumes and produces only the shared, aggregate-agnostic types already defined in `internal/domain/command`: `GovernedCommandProposal`, `KernelDecisionEnvelope`, and `policy.GovernanceDecision`.

### `internal/kernel/workflow` — implemented

Signatures only, reproduced from the real package (bodies omitted; see [proposal.go](../../apps/companyd/internal/kernel/workflow/proposal.go) and [decision.go](../../apps/companyd/internal/kernel/workflow/decision.go)):

```go
package workflow // internal/kernel/workflow

func ValidateCreateProposal(cmd command.WorkflowCommandEnvelope, reg fixtures.Registry) (*command.GovernedCommandProposal, []string)

func ValidateStartProposal(cmd command.WorkflowCommandEnvelope, reg fixtures.Registry, current workflow.Workflow) (*command.GovernedCommandProposal, []string)

func ValidateResultProposal(cmd command.WorkflowCommandEnvelope, current workflow.Workflow, res result.Result, attemptWorkflowVersion int64) (*command.GovernedCommandProposal, []string)

func FinalizeCreate(cmd command.WorkflowCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string)

func FinalizeStart(cmd command.WorkflowCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, current workflow.Workflow, reg fixtures.Registry, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string)

func FinalizeAccept(cmd command.WorkflowCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, current workflow.Workflow, res result.Result, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string)

func FinalizeReject(cmd command.WorkflowCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, current workflow.Workflow, res result.Result, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string)
```

`CANCEL_WORKFLOW` is named in `command.CommandType` but has no `ValidateCancelProposal` / `FinalizeCancel` pair yet — `ROADMAP.md` Phase 1 Slice 6, not started.

### `internal/kernel/organization` — not implemented, proposed shape only

No `command.OrganizationCommandEnvelope` exists. `organization.Organization` (`internal/domain/organization/types.go`) has only `OrganizationID`, `Name`, `Status` (`ACTIVE`/`INACTIVE`) — today's fixture never legally transitions it. `ROADMAP.md` Phase 3 Slice 6 is the first real driver ("governed Organization-creation use case"). Illustrative shape, following the Workflow pattern exactly:

```go
package organization // internal/kernel/organization — PROPOSED, NOT IMPLEMENTED

func ValidateCreateProposal(cmd command.OrganizationCommandEnvelope, reg fixtures.Registry) (*command.GovernedCommandProposal, []string)

func FinalizeCreate(cmd command.OrganizationCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string)
```

Flagged, not guessed: the full legal-transition table — is `INACTIVE` a reachable state, and via which command? — is undefined. `docs/domain/organization.md` was not re-derived for this ADR and should be consulted before implementation.

### `internal/kernel/objective` — not implemented, proposed shape only

No `command.ObjectiveCommandEnvelope` exists. `objective.Objective` (`internal/domain/objective/types.go`) has `Status`: `PROPOSED` / `APPROVED` / `RETIRED`. [departments.md](../architecture/departments.md)'s "Objective creation gate" already establishes that only a distinct, governed Application request may create one — never a Finding, Recommendation, or Evaluation directly. Illustrative shape:

```go
package objective // internal/kernel/objective — PROPOSED, NOT IMPLEMENTED

func ValidateProposeProposal(cmd command.ObjectiveCommandEnvelope, reg fixtures.Registry) (*command.GovernedCommandProposal, []string)

func ValidateApproveProposal(cmd command.ObjectiveCommandEnvelope, current objective.Objective) (*command.GovernedCommandProposal, []string)

func FinalizePropose(cmd command.ObjectiveCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string)

func FinalizeApprove(cmd command.ObjectiveCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, current objective.Objective, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string)
```

Flagged, not guessed: whether `RETIRED` is reachable by direct command or only as a side effect of another transition is undefined by any canonical doc read for this ADR.

### `internal/kernel/department` — blocked, no signature proposed

`internal/domain/department` contains only `doc.go` — no Go type exists for `Department` at all (confirmed by repository scan, 2026-08-21). [departments.md](../architecture/departments.md) describes a richer lifecycle than a simple status enum: registration → validation → a `VALIDATED` intermediate state (per `docs/domain/department.md#department-lifecycle`, not re-read for this ADR) → governed activation → optional deactivation, plus a `DepartmentRegistry` that resolves every referenced role, policy, workflow, metric, event, capability, and knowledge-namespace reference before activation. Writing a Go signature now would mean inventing struct fields this ADR has no authority to invent.

**Follow-up required before `internal/kernel/department` can exist:** port `docs/domain/department.md`'s lifecycle into `internal/domain/department/types.go`, then return to this contract.

### `internal/kernel/capability` — not implemented, proposed shape only

No `command.CapabilityCommandEnvelope` exists. `capability.CapabilityDefinition` (`internal/domain/capability/types.go`) has an `Active bool` but no richer lifecycle enum; today it is only ever created via the hardcoded `fixtures.Registry` (Phase 1 Slice 1), never through a governed path. Illustrative shape:

```go
package capability // internal/kernel/capability — PROPOSED, NOT IMPLEMENTED

func ValidateRegisterProposal(cmd command.CapabilityCommandEnvelope, reg fixtures.Registry) (*command.GovernedCommandProposal, []string)

func FinalizeRegister(cmd command.CapabilityCommandEnvelope, proposal command.GovernedCommandProposal, decision policy.GovernanceDecision, declaredTime time.Time) (*command.KernelDecisionEnvelope, []string)
```

Flagged, not guessed: `Active bool` may not be sufficient once versioned CapabilityDefinitions can be superseded — whether activation/deprecation needs a richer state (mirroring `WorkflowDefinition.Active`'s same simplicity, and the same open risk) is undefined.

### A cross-cutting risk this ADR surfaces but does not resolve

`internal/kernel/workflow/digest.go`'s `canonicalDigest`/`canonicalJSON` and `decision.go`'s `verifyAllow` are workflow-package-private today. If Organization, Objective, and Capability each reimplement them, four copies of the same digest and allow-verification logic will exist. Flagged as needing a follow-up decision, not resolved here: should these move to a shared `internal/kernel` (or `internal/kernel/shared`) package once a second aggregate is implemented, given kernel.md never names a shared kernel-level package today?

## Consequences

### Positive

- Fixes one contract shape before four more aggregates get built independently, preventing five incompatible designs.
- Makes explicit, for the first time, exactly which Kernel aggregates have zero Go implementation today (Organization, Objective, Department, Capability) versus the one that is fully shipped (Workflow) — `ROADMAP.md` names the phases but not this gap in Go-level terms.
- Reuses `internal/domain/command`'s existing aggregate-agnostic envelope types (`GovernedCommandProposal`, `KernelDecisionEnvelope`) without modification.

### Costs and risks

- Four of five proposed signatures are illustrative, not implementable as-is — each blocks on a command-vocabulary decision this ADR deliberately does not make, so it cannot be treated as an implementation-ready spec for those four.
- `internal/kernel/department` cannot be scaffolded at all until `docs/domain/department.md` gets a Go type — this ADR cannot shortcut that.
- Naming every function `ValidateXProposal` / `FinalizeX` is a convention, not a compiler-enforced contract — this ADR rejects a Go `interface`, so nothing stops a future package from drifting. Enforcement is a follow-up (e.g., a contract test per `ROADMAP.md` Phase 2 Slice 6), not something Go's type system guarantees here.

## Alternatives rejected by this proposal

- **A single Go `interface` with `Propose(cmd Command, state any) (*Proposal, []string)` / `Decide(...) (*Decision, []string)` methods, implemented once per aggregate:** rejected because `any` erases the exact preconditions each command legitimately needs — a fresh `CREATE` has no current state; a result-acceptance command needs a `Result`. The real `internal/kernel/workflow` package already avoids this for a good reason; retrofitting an interface would be a regression, not a generalization.
- **One `internal/kernel` package for all five aggregates instead of one package per aggregate:** rejected because kernel.md's "First aggregate boundary" section already treats each aggregate as owning a distinct boundary ("The Workflow aggregate cannot mutate them \[other aggregates\]"), and `docs/architecture/departments.md`'s dependency rule — no department may import another's implementation — reflects the same instinct CompanyOS already applies elsewhere. One package per aggregate keeps that boundary from blurring the way a shared department package would.
- **Defining the full command vocabulary and struct fields for Organization, Objective, Department, and Capability now, in this ADR:** rejected. This ADR's mandate, per its Context, is the Go contract *shape*, not new domain decisions; inventing command names or Department's struct fields here would be exactly the guessing this ADR was asked not to do.

## Acceptance criteria

- [x] Cross-checked against `kernel.md`, `ADR-0001`, `governance.md`, `runtime.md`, `daemon.md`, and `events.md` for contradictions — none found. This is a single-session review, not the dedicated multi-day audit `ADR-0001`'s original acceptance went through.
- [ ] `internal/kernel/organization`, `objective`, and `capability`'s illustrative signatures are reconciled with their respective domain docs' actual legal-transition tables before implementation begins.
- [ ] `docs/domain/department.md`'s lifecycle is ported into `internal/domain/department/types.go` before `internal/kernel/department` is scaffolded.
- [ ] the project owner reviews and explicitly changes `Status: PROPOSED` to `Status: APPROVED`.

## Open questions

- OPEN QUESTION: Should `canonicalDigest`/`canonicalJSON`/`verifyAllow` move to a shared `internal/kernel` package once a second aggregate is implemented, rather than being reimplemented per package?
- OPEN QUESTION: What is Organization's full legal-transition table — is `INACTIVE` reachable, and by which command?
- OPEN QUESTION: Is `Objective.RETIRED` reachable by direct command, or only as a side effect of another transition?
- OPEN QUESTION: Does `CapabilityDefinition` need a richer lifecycle than `Active bool` once versions can be superseded?
- OPEN QUESTION: Should each aggregate's `XCommandEnvelope` be a distinct type (mirroring `WorkflowCommandEnvelope`), or is a shared envelope shape possible without carrying `WorkflowCommandEnvelope`'s workflow-specific fields (`WorkflowID`, `DefinitionID`, `ResultID`)?

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [Kernel](../architecture/kernel.md)
- [ADR-0001](ADR-0001-kernel-runtime-daemon.md)
- [Governance](../architecture/governance.md)
- [Runtime](../architecture/runtime.md)
- [Daemon](../architecture/daemon.md)
- [Events](../architecture/events.md)
- [Departments](../architecture/departments.md)
- [Command domain](../domain/command.md)
- [Organization domain](../domain/organization.md)
- [Objective domain](../domain/objective.md)
- [Department domain](../domain/department.md)
- [Capability domain](../domain/capability.md)
