# Feature Provenance

Status: DRAFT

This document records proposed influences, not implemented features or approved dependencies. CompanyOS owns every organizational abstraction and must approve borrowing through architecture review and, where durable, an ADR.

## Borrowing categories

- **Dependency candidate:** a library or service may implement a bounded CompanyOS port after evaluation.
- **Pattern:** CompanyOS implements the lesson behind its own contract without importing the reference architecture.
- **Original adaptation:** external work informs the design, but CompanyOS owns the feature and semantics.

## Proposed mapping

| CompanyOS feature | Owner | Primary references | Category | Borrow or adapt | Why / boundary |
|---|---|---|---|---|---|
| Workflow graph and typed state | Runtime | LangGraph.js, LangGraph | Dependency candidate | Nodes, edges, conditional flow, state/graph separation, compilation checks | Runtime may use graph mechanics; CompanyOS domain records remain authoritative. |
| Human pause and resume | Runtime + Kernel governance | LangGraph.js, Temporal, OpenAI Agents SDK | Pattern | Checkpoint before wait, explicit interruption, stable resumable identity | Approval meaning belongs to governance, not an agent or graph framework. |
| Workflow checkpointing | Runtime | LangGraph, Temporal | Pattern | Snapshot contracts, pending writes, recovery metadata | Checkpoints support execution and never replace authoritative organizational state. |
| Durable long-running execution | Runtime + Daemon | Temporal service and TypeScript SDK | Pattern initially | Workflow/activity separation, durable identity, history, signals, recovery | Do not add Temporal until a concrete durability requirement justifies its operational cost. |
| Retry and recovery | Runtime | Temporal; Inngest as secondary reference | Pattern | Explicit retry policy, backoff, non-retryable failures, side-effect separation | Domain rules, idempotency, and policy gates constrain retries; Inngest code reuse requires license review. |
| Scheduled and event-triggered work | Daemon | Inngest | Pattern only by default | Cron/event triggers, flow control, steps that wake work | Daemon should wake authorized work instead of keeping expensive agents continuously active; SSPL service obligations make dependency use subject to legal review. |
| Long-running background tasks | Daemon + Runtime | Trigger.dev | Pattern | Durable waits, retries, queue boundaries, observable run status | Trigger.dev is a study target, not an approved runtime dependency. |
| Named agents, tools, and handoffs | Agent runtime | OpenAI Agents SDK, JARVIS | Pattern | Minimal agent definition, typed tools, delegation, sessions | CompanyOS remains an organizational runtime, not an agent SDK wrapper. |
| Agent authority and approval delivery | Kernel governance + Agent runtime | JARVIS, OpenAI Agents SDK | Pattern | Spawn/delegation checks, explicit approval request lifecycle, controlled tool execution | Agents may request actions but cannot grant themselves authority or mutate workflow truth. |
| Agent tracing and audit evidence | Runtime + M&E | OpenAI Agents SDK, JARVIS | Pattern | Stable run/turn/tool identifiers and correlated outcomes | M&E and governance own interpretation, retention, and audit requirements. |
| Dynamic team planning | Runtime + departments | Open Multi-Agent, JARVIS | Pattern | Bounded task DAG generation and capability-based assignment | Dynamic planning must operate inside approved objectives, budgets, and authority. |
| Team budgets and capability matching | Finance + Runtime | Open Multi-Agent | Original adaptation | Execution budgets, structured capabilities, task requirements | Finance owns effective cost per result and resource governance, not the orchestration framework. |
| Policy authorization | Kernel governance | Cedar | Dependency candidate | Principal/action/resource/context requests, policy sets, schemas, validation | Cedar may answer allow/deny; CompanyOS owns organizational meaning and approval escalation. |
| `require_approval` outcome | Kernel governance | Cedar, JARVIS, OpenAI Agents SDK | Original adaptation | Extend binary authorization with an explicit approval-required result | This three-way organizational outcome is a CompanyOS responsibility. |
| Model-independent intelligence and routing | Intelligence | OpenAI Agents SDK provider boundaries; Open Multi-Agent routing | Original adaptation | Provider ports plus capability, complexity, quality, and cost evidence | Research, M&E, and Finance continuously influence routing; no framework owns the decision. |
| Provider selection under rate-limit/outage | Intelligence + Runtime | LiteLLM (`lowest_latency.py`, `budget_limiter.py`, health/cooldown routing — see `docs/architecture/intelligence.md`'s OSS evidence) | Pattern | Skip a provider that just failed, prefer a healthy one, cooldown window before retrying it | Narrower than the row above — no Governance/Finance/M&E evidence consulted, purely operational health. See "Implemented so far" below: this one has shipped. |
| Coding-agent routing | Engineering via CodingAgentRuntime | Open Multi-Agent, JARVIS | Original adaptation | Capability-based selection of Codex, Claude Code, Gemini CLI, and future adapters | Departments request engineering capability, never a concrete vendor. |
| Engineering workspace isolation | Engineering + workspace runtime | Agent sandbox patterns; future Codespaces study | Original adaptation | Isolated filesystem, controlled commands, patches, and snapshots | Concrete sandbox and cloud providers remain unresolved. |
| Feature branch through reviewed PR | Engineering + governance | GitHub engineering model and coding-agent projects | Pattern | Isolated change set, reproducible checks, independent review evidence | Implementers cannot approve their own governed work. |
| Organization Kernel and department plugins | Kernel | CompanyOS original | Original | Mission, objectives, departments, capabilities, policies, and organizational invariants | This is the defining CompanyOS abstraction and must not be delegated to a reference framework. |
| Research–M&E–Finance feedback loop | Kernel + departments | CompanyOS original | Original | Evidence to execution to evaluation to cost/value to new research | This is the adaptive heart of CompanyOS. |
| Organizational and engineering memory | Kernel knowledge + engineering context service | LangGraph persistence and agent sessions as lessons | Original adaptation | Approved knowledge separated from task context and transient conversations | Models and agents consume memory but never own authoritative organizational knowledge. |

## Implemented so far

Every row above is a proposal per this document's own rule — "Do not describe a proposed feature as implemented." One row is a genuine exception, flagged explicitly here rather than left as a silent contradiction between this doc and the actual codebase:

- **Provider selection under rate-limit/outage** (the LiteLLM-pattern row above) is implemented in `internal/adapters/intelligence/fallback` and wired into `cmd/companyd/main.go` — it composes the Gemini/OpenAI/Anthropic `ProviderAdapter`s, tries them in priority order, and puts one in cooldown after a retryable failure. It is first-slice operational scope under `ADR-0004` (already `APPROVED`), not the full evidence-based Intelligence Router `ADR-0003` describes (Task Analyzer, Governance-eligible ranking, Finance budget, M&E evidence, persisted `RoutingDecision`), which remains entirely future work (`ROADMAP.md`, "Beyond Phase 9").

This implementation did **not** go through this document's own approval requirement below (source inspection, license verification, adopted/rejected pattern documentation, failure/security analysis, an ADR where durable) — it was scoped and shipped as narrow first-slice technology under `ADR-0004`'s existing approval instead. That's a real gap in process, not just paperwork: nothing here has verified LiteLLM's license terms or analyzed failure modes of the borrowed pattern itself. Close it before extending this pattern further or implementing another OSS-derived pattern ahead of formal review.

## Explicit rejections

- Do not make agent messages, LangGraph state, or framework sessions authoritative organizational state.
- Do not let Temporal, Inngest, Trigger.dev, or JARVIS own CompanyOS domain semantics.
- Do not let Cedar policies replace governance concepts or approval records.
- Do not add a dependency merely because its pattern is useful.
- Do not describe a proposed feature as implemented.

## Approval requirement

Before a provenance entry becomes approved, inspect the pinned source, verify its license, document adopted and rejected patterns, analyze failure and security implications, and record a durable decision in an ADR when appropriate.
