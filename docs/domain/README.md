# Domain Documentation

This directory owns canonical definitions, identities, lifecycles, relationships, and invariants for CompanyOS concepts.

Statuses mirror existing document metadata. Rows marked `NOT YET SPECIFIED` are planned contracts and may not have a file yet.

| Document | Purpose | Status | Read when |
|---|---|---|---|
| `organization.md` | Organization identity, lifecycle, and isolation boundary | DRAFT | Read when working with company-level state or tenant scope. |
| `objective.md` | Objective meaning and lifecycle | DRAFT | Read when creating or evaluating objectives. |
| `department.md` | Department identity and contracts | DRAFT | Read when changing department semantics. |
| `workflow.md` | Workflow state and transition invariants | DRAFT | Read when changing workflow semantics. |
| `agent.md` | Agent definition, participation, and authority boundary | DRAFT | Read when defining agent behavior or lifecycle. |
| `capability.md` | Provider-independent outcome and dispatch contract | DRAFT | Read when requesting, routing, or implementing capabilities. |
| `command.md` | Shared command, governed-proposal, Kernel-decision, and pending-command envelopes | DRAFT | Read when coordinating a state-changing request across Application, Kernel, or Governance. |
| `principal.md` | Durable actor identity and delegation references | DRAFT | Read when identifying human, agent, service, or provider actors. |
| `policy.md` | Organizational policy semantics | DRAFT | Read when defining governance rules. |
| `approval.md` | Approval requests and decisions | DRAFT | Read when adding approval gates. |
| `artifact.md` | Durable work-product semantics | DRAFT | Read when producing or consuming artifacts. |
| `evidence.md` | Attributable observations and provenance | DRAFT | Read when supporting claims, decisions, metrics, or results. |
| `result.md` | Reported capability and workflow-step outcomes | DRAFT | Read when accepting or rejecting execution outcomes. |
| `metric.md` | Metric definitions and observations | DRAFT | Read when measuring performance or outcomes. |
| `evaluation.md` | Evaluation and performance-profile semantics | DRAFT | Read when judging results or comparative performance. |
| `resource.md` | Provider-independent resource constraints | DRAFT | Read when constraining cost, compute, time, storage, or concurrency. |
| `workspace.md` | Workspace identity, lifecycle, transitions, and invariants | DRAFT | Read when changing Workspace or EngineeringWorkspace semantics. |
| `event.md` | Domain event semantics | DRAFT | Read when emitting or consuming domain events. |
| `knowledge.md` | Reviewed reusable claims and provenance | DRAFT | Read when curating or retrieving organizational knowledge. |
