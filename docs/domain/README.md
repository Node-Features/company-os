# Domain Documentation

This directory owns canonical definitions, identities, lifecycles, relationships, and invariants for CompanyOS concepts.

Statuses mirror existing document metadata. Rows marked `NOT YET SPECIFIED` are planned contracts and may not have a file yet.

| Document | Purpose | Status | Read when |
|---|---|---|---|
| `organization.md` | Organization identity, lifecycle, and isolation boundary | APPROVED | Read when working with company-level state or tenant scope. |
| `objective.md` | Objective meaning and lifecycle | APPROVED | Read when creating or evaluating objectives. |
| `department.md` | Department identity and contracts | APPROVED | Read when changing department semantics. |
| `workflow.md` | Workflow state and transition invariants | APPROVED | Read when changing workflow semantics. |
| `execution.md` | Runtime execution-state mechanics: attempts, leases, checkpoints, waits, retries, resume | APPROVED | Read when changing Runtime recovery, retry, or dispatch mechanics. |
| `agent.md` | Agent definition, participation, and authority boundary | APPROVED | Read when defining agent behavior or lifecycle. Reopened and re-approved 2026-08-23 for a display-avatar addition. |
| `capability.md` | Provider-independent outcome and dispatch contract | APPROVED | Read when requesting, routing, or implementing capabilities. |
| `command.md` | Shared command, governed-proposal, Kernel-decision, and pending-command envelopes | APPROVED | Read when coordinating a state-changing request across Application, Kernel, or Governance. |
| `principal.md` | Durable actor identity and delegation references | APPROVED | Read when identifying human, agent, service, or provider actors. |
| `policy.md` | Organizational policy semantics | APPROVED | Read when defining governance rules. |
| `approval.md` | Approval requests and decisions | APPROVED | Read when adding approval gates. |
| `artifact.md` | Durable work-product semantics | APPROVED | Read when producing or consuming artifacts. |
| `evidence.md` | Attributable observations and provenance | APPROVED | Read when supporting claims, decisions, metrics, or results. |
| `result.md` | Reported capability and workflow-step outcomes | APPROVED | Read when accepting or rejecting execution outcomes. |
| `metric.md` | Metric definitions and observations | APPROVED | Read when measuring performance or outcomes. |
| `evaluation.md` | Evaluation and performance-profile semantics | APPROVED | Read when judging results or comparative performance. |
| `resource.md` | Provider-independent resource constraints | APPROVED | Read when constraining cost, compute, time, storage, or concurrency. |
| `workspace.md` | Workspace identity, lifecycle, transitions, and invariants | APPROVED | Read when changing Workspace or EngineeringWorkspace semantics. |
| `event.md` | Domain event semantics | APPROVED | Read when emitting or consuming domain events. |
| `knowledge.md` | Reviewed reusable claims and provenance | APPROVED | Read when curating or retrieving organizational knowledge. |
