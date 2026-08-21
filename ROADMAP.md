# CompanyOS Roadmap

Status: DRAFT

This expands the phase summary in [README.md — Roadmap](README.md#roadmap). It tracks intended sequencing, not committed dates. Live blockers and next tasks are tracked in [.companyos/agent-memory/current-state.md](.companyos/agent-memory/current-state.md), which is authoritative for current status; this file is not.

This edition (2026-08-21) extends the roadmap from "first vertical slice" through to a genuinely production-ready CompanyOS: Phases 2 and 5–9 are new, and Phases 3–4 are expanded from single vague bullets into concrete, doc-grounded slices. Where a phase's sequencing is a judgment call rather than something a canonical doc mandates, that's said explicitly — this file tracks *intended* sequencing, and judgment calls are expected here, not disguised as doc-mandated order.

## Phase 0 — Foundation (complete)

- [x] Define project direction and core principles (`README.md`)
- [x] Establish documentation hierarchy (`AGENTS.md`, `docs/INDEX.md`)
- [x] Establish open-source provenance process (`docs/references/`)
- [x] Complete architecture reconciliation across `docs/architecture/` and `docs/domain/`
- [x] Resolve remaining ownership boundaries and open blockers (see `current-state.md`)
- [x] Accept foundational ADRs — `ADR-0001` (Kernel/Runtime/Daemon separation), `ADR-0002` (pluggable departments), `ADR-0003` (model-independent intelligence)
- [x] Complete a fresh read-only architecture audit
- [x] Project owner (`Node-Features`) explicitly approves architecture and domain documents (2026-08-20; see `current-state.md`)

## Phase 1 — First vertical slice

Grounded in the [Application layer's first vertical slice](docs/architecture/application.md#first-vertical-slice) and the [Workflow first-slice lifecycle](docs/domain/workflow.md#first-slice-lifecycle). `ADR-0004` answers this phase's three long-standing open questions (Runtime, persistence adapter, first CapabilityDefinition) and was approved 2026-08-21.

Slices 0–5 and 7 are complete (2026-08-21): `CREATE_WORKFLOW → START_WORKFLOW → Runtime dispatch → ACCEPT/REJECT_WORKFLOW_RESULT` works end to end against the real database and a real provider call, triggered from `web`. Slice 6 (`CANCEL_WORKFLOW`) and Slice 8 (realtime activity layer) remain. The slices below are ordered so each one is independently buildable and testable — no slice depends on a later one.

- [x] **Slice 0 — Formalize the stack decision.** `ADR-0004` `Status: PROPOSED` → `APPROVED` (2026-08-21; the scaffold already matched it). Its four sub-questions remain open as implementation detail, not a blocker: first LLM provider, Next.js↔`companyd` transport, Supabase RLS design, first Retry policy defaults.
- [x] **Slice 1 — Choose the first fixtures.** `internal/fixtures.Registry`: hardcoded Organization, Objective, `WorkflowDefinition`, and the minimal `IntelligenceCapability` from `ADR-0004`.
- [x] **Slice 2 — Schema + persistence adapter.** `supabase/migrations/` (workflows, domain_events, event_outbox, execution_intents, execution_attempts, results, governance_decisions, pending_commands, approvals, idempotency_keys) plus `internal/adapters/persistence/supabase/{workflow_repo,execution_repo,pending_repo}.go` implementing the ports, committing state and DomainEvents atomically.
- [x] **Slice 3 — `CREATE_WORKFLOW`.** `internal/kernel/workflow` (proposal + final decision) + `internal/governance` (default-deny policy) + `internal/application.CreateWorkflow`. Covered by unit and integration tests.
- [x] **Slice 4 — `START_WORKFLOW` + Runtime.** `internal/runtime.Runtime` claims and dispatches via `internal/adapters/intelligence/fallback`, which tries Gemini/OpenAI/Anthropic in priority order and skips a provider that just rate-limited or went down (cooldown-based), rather than one fixed `ProviderAdapter`.
- [x] **Slice 5 — Result acceptance.** `internal/application.SubmitResult`: `SUCCEEDED` → `COMPLETED`, `FAILED`/`TIMED_OUT`/`PARTIAL` → `FAILED`, `INDETERMINATE` → no transition. Verified live end-to-end with a real provider call and real generated text.
- [ ] **Slice 6 — `CANCEL_WORKFLOW`.** `PLANNED` or `READY` → `CANCELLED`, authorized-Principal only. Not started.
- [x] **Slice 7 — Expose it through `web`.** `internal/adapters/httpapi/workflows.go` (`POST /v1/workflows`, `.../start`, `.../results/{accept,reject}`, `GET /v1/workflows/{id}`) plus a Next.js trigger page (`components/WorkflowTrigger.tsx`) that creates, starts, and polls a Workflow to a terminal state — `web` calls `companyd` for this, never writes Supabase directly.
- [ ] **Slice 8 — Realtime activity layer.** Replace `WorkflowTrigger`'s 2-second poll with a push update, so `web` shows Workflow state changes and Results as they happen — the transparency principle `README.md` already states: a human "should be able to leave CompanyOS running, return later, and immediately answer: What did the company discover? What did it decide? What did it do?" Grounded in `ADR-0004` item 4, which already named Supabase Realtime (`LISTEN`/`NOTIFY`) as the notification mechanism and deliberately deferred it — first-slice plan decision #7 shipped `event_outbox` as a real table with a **no-op publisher stub**, justified only because `companyd` runs as one process; that justification stops holding once this slice's UI needs to observe events happening in `companyd`, not just `companyd`'s own Runtime waking itself up. Scope note: start with one Workflow's live status (extends Slice 7 directly); a company-wide activity feed spanning multiple Workflows/departments is broader scope that depends on Phase 4 (departments) existing to have multiple activity sources to show.

## Phase 2 — Security & testing foundations

**Sequencing judgment:** placed before Phase 3 (Governed execution) and Phase 6 (Engineering) because both explicitly consume these docs — [`docs/architecture/workspaces.md`](docs/architecture/workspaces.md)'s isolation section assumes a `workspace-isolation.md` doc that doesn't exist yet, and CI/testing growth (Phase 8) needs `contract-tests.md`. All four `docs/security/*` docs and all three `docs/testing/*` docs are currently `NOT YET SPECIFIED` ([`docs/security/README.md`](docs/security/README.md), [`docs/testing/README.md`](docs/testing/README.md)). This mirrors Phase 0's ADR-approval gate: these docs must reach `APPROVED` before the work that depends on them is buildable-on, not just written. Slices 1–7 are independently authorable and approvable in parallel — no slice depends on another. Slice 8 depends on Slice 5.

- [ ] **Slice 1 — Write and approve `docs/security/threat-model.md`.** Assets, trust boundaries, threats, and mitigations. Grounds Governance's `DENY` semantics ([`governance.md`](docs/architecture/governance.md)) and Workspaces' isolation claims in an explicit threat model rather than ad hoc judgment.
- [ ] **Slice 2 — Write and approve `docs/security/agent-authority.md`.** Agent permission and authority limits. Grounds the Authority check in `governance.md`'s decision pipeline specifically for `AgentPrincipal` requests, which [`identity.md`](docs/architecture/identity.md)'s Agent flow leaves generic.
- [ ] **Slice 3 — Write and approve `docs/security/tool-security.md`.** Tool access and external-action controls. A named dependency of [`coding-agents.md`](docs/architecture/coding-agents.md)'s normalized-tool-policy invariant — needed before Phase 6's coding-agent tool execution.
- [ ] **Slice 4 — Write and approve `docs/security/workspace-isolation.md`.** Engineering workspace isolation requirements. Directly answers `workspaces.md`'s open question — which initial provider mechanism (local process sandbox, container, VM, remote workspace service) satisfies required isolation controls — that Phase 6's Workspace-lifecycle slice cannot answer without inventing the decision.
- [ ] **Slice 5 — Write and approve `docs/testing/strategy.md`.** Overall testing responsibilities and levels. Foundation for Phase 8 (CI/CD growth).
- [ ] **Slice 6 — Write and approve `docs/testing/contract-tests.md`.** Shared-contract and adapter verification — required before Phase 8's contract-test CI job, and before Phase 6 adds `WorkspaceProvider`/`CodingAgentRuntime` adapters that need the same discipline.
- [ ] **Slice 7 — Write and approve `docs/testing/failure-injection.md`.** Crash, retry, replay, recovery testing — grounds validation of `workspaces.md`'s recovery/reconciliation guarantees (lease fencing, checkpoint resume, corruption retry) before Phase 6 builds them.
- [ ] **Slice 8 — Retire first-slice fake test doubles before Phase 4's real operational modules begin.** `internal/application/fake_repo_test.go` (`fakeRepo`/`fakeExec`/`fakePending`, in-memory stand-ins for the persistence ports) was the right call for a single narrow vertical slice, but `internal/application/integration_test.go` now proves the same use cases against the *real* repositories, and Phase 4 is about to add several more operational modules (Research, M&E, Finance) that would otherwise each grow their own copy of the same fake-repo pattern. Once `docs/testing/strategy.md` (Slice 5) picks a testing discipline, apply it here: either retire the fakes in favor of integration-only tests, or make them a documented, sanctioned pattern other modules are expected to reuse — don't let each new module invent its own answer ad hoc. Depends on Slice 5; gates Phase 4's testing approach, not its department docs (Phase 4 Slice 0).

## Phase 3 — Governed execution

Grounded in [`docs/architecture/governance.md`](docs/architecture/governance.md)'s decision pipeline, [`docs/architecture/identity.md`](docs/architecture/identity.md)'s Human authentication flow, and `ADR-0004` item 6 (Supabase Auth as the first `Authenticator` adapter). `apps/web/lib/supabase/{client,server,middleware}.ts` already scaffold session refresh for `HumanPrincipal`; `apps/companyd/internal/adapters/identity/supabaseauth/doc.go` is a stub package with no implementation yet.

**Parallelism:** Slices 1, 2, and 3 can start immediately and in parallel. Slice 5 can start alongside 1–3 (it only needs *a* policy to re-evaluate, not the final one). Slices 4 and 6 both gate on Slice 3.

- [ ] **Slice 1 — Governance `DENY` path end-to-end.** Add at least one concrete policy that produces `DENY` (e.g., organization-mismatched Principal, or an unauthorized Principal attempting `CANCEL_WORKFLOW`) and verify it's surfaced through `companyd` HTTP and `web` without executing, per `governance.md`'s invariant that no governed action reaches an executor without a current persisted `ALLOW` decision.
- [ ] **Slice 2 — Governance `REQUIRE_APPROVAL` path + the Approval domain.** Classify one action (e.g., `CANCEL_WORKFLOW` by a non-owner) as `approval_required`; implement [`docs/domain/approval.md`](docs/domain/approval.md)'s canonical Approval record; wire Application-coordinated persistence so a human's Approval evidence lets a resubmitted request reach `ALLOW`.
- [ ] **Slice 3 — Identity: `HumanPrincipal` authentication evidence flow via Supabase Auth.** Implement `apps/companyd/internal/adapters/identity/supabaseauth` (currently a doc comment only) to verify Supabase JWTs into `AuthenticatedPrincipalEvidence` per `identity.md`'s Human flow and `ADR-0004` item 6. Scope note: only `HumanPrincipal` this phase — Agent/Service/Provider flows are deferred. Gates Slices 4 and 6.
- [ ] **Slice 4 — Real login UI in `web`.** Sign-in/sign-up pages on the existing `@supabase/ssr` scaffolding and session-refresh `middleware.ts`; gate `WorkflowTrigger` and all future authoritative actions behind an authenticated session; `companyd`'s HTTP API requires and validates the Supabase-issued JWT through Slice 3's adapter — never trusting a client-asserted Principal.
- [ ] **Slice 5 — Dispatch-time re-evaluation.** Wire `internal/runtime.Runtime`'s dispatch path to request a fresh Governance decision through an Application use case immediately before each governed external effect, per `governance.md`.
- [ ] **Slice 6 — Retire the fixture Organization; real Organization creation and onboarding.** Replace `internal/fixtures.Registry`'s hardcoded Organization with a governed Organization-creation use case ([`docs/domain/organization.md`](docs/domain/organization.md)) and an onboarding flow binding the first authenticated `HumanPrincipal` to it (`identity.md`'s `PrincipalOrganizationBinding`).

## Phase 4 — Adaptive organization

**Gate (mirrors Phase 0's ADR-approval precedent):** [`docs/departments/research.md`](docs/departments/research.md), [`monitoring-evaluation.md`](docs/departments/monitoring-evaluation.md), and [`finance.md`](docs/departments/finance.md) are all `Status: DRAFT` today. No slice below should begin implementation until the project owner moves each to `APPROVED` — the same review gate Phase 0's architecture and domain docs went through.

- [ ] **Slice 0 — Approve Research, M&E, and Finance department docs.** Blocks every slice below.
- [ ] **Slice 1 — Research: Signal → ResearchQuestion → Evidence → Finding → Recommendation for one signal source.** Implement `research.md`'s core contracts end to end for one narrow, manually-submitted signal type (not an external source integration yet). Write and approve `docs/workflows/research-loop.md` (currently `NOT YET SPECIFIED`) alongside or just before this slice — it's the canonical use-case doc for the continuous research feedback loop.
- [ ] **Slice 2 — M&E: Result → Metric → Evaluation for one Result source.** Implement `monitoring-evaluation.md`'s core contracts (`Result`, `MetricDefinition`/`Metric`, `Evaluation`, `PerformanceProfile`) against the Results Phase 1 already produces (`SubmitResult`'s `SUCCEEDED`/`FAILED`/... outcomes). Independent of Slice 1.
- [ ] **Slice 3 — Finance: Budget + ResourceConstraint + PriceProfile + ResourceUsage + ResourceEvaluation for the first slice's capability.** Grounded in `finance.md`'s core contracts, applied to the single `IntelligenceCapability`/`ModelProfile`/`ProviderAdapter` `ADR-0004` already fixed. Independent of Slices 1 and 2.
- [ ] **Slice 4 — Objective-creation gate from a Finding, Recommendation, or Evaluation.** Implement the governed proposal path in [`departments.md`](docs/architecture/departments.md)'s adaptive feedback loop (Recommendation/Evaluation → Proposal → Governance decision → Kernel → Objective) — a Finding/Recommendation/Evaluation may *propose* an Objective through a distinct Application use case, never create one directly. Depends on at least one of Slices 1–3 and reuses Phase 3's Governance `REQUIRE_APPROVAL`/`DENY` machinery.

Carried forward, not resolved by this phase: `research.md`'s "Which research classes require human review before a finding or recommendation is published?"; `monitoring-evaluation.md`'s "Which outcomes require human, automated, or dual evaluation?"; `finance.md`'s "Which resource limits are hard execution gates versus alert-only controls?"

## Phase 5 — Organizational knowledge

Grounded in [`docs/architecture/knowledge.md`](docs/architecture/knowledge.md) (APPROVED) and its `knowledge.approve` governance boundary. Depends on Phase 3 Slice 3 (real `HumanPrincipal` auth evidence — `knowledge.approve` is `human_only`) and Phase 4 Slice 1 (Research is the primary proposal source — "results become approved knowledge only through the knowledge-review lifecycle").

- [ ] **Slice 1 — KnowledgeItem ingestion and versioning for one source type.** Capture, preserve source integrity/classification, normalize into an immutable version, detect duplicates/contradictions as review signals — for one source, a Research Finding being the natural first candidate given Phase 4 Slice 1.
- [ ] **Slice 2 — `knowledge.review`/`knowledge.approve` governed action.** Governance verifies the human reviewer's Authority, separation-of-duties, and exact item-version/content digest; only a current `ALLOW` permits the Kernel transition to `APPROVED`.
- [ ] **Slice 3 — Retrieval contract, `APPROVED`-only by default.** Default organizational queries return only current `APPROVED` items; draft-inclusive queries require explicit purpose and labeling.
- [ ] **Slice 4 — Wire Research Finding → proposed KnowledgeItem end-to-end.** Connects Phase 4 Slice 1's Finding output to Slice 1's ingestion path.

Carried forward: which Governance-authorized human roles, independence rules, and review counts apply to each knowledge class; what taxonomy and scope hierarchy the first vertical slice requires — both unresolved in `knowledge.md`, do not invent an answer in implementation.

## Phase 6 — Engineering: Workspaces + CodingAgentRuntime

Grounded in [`docs/architecture/workspaces.md`](docs/architecture/workspaces.md) and [`docs/architecture/coding-agents.md`](docs/architecture/coding-agents.md) (both APPROVED). Depends on Phase 2 Slice 4 (`workspace-isolation.md`), Phase 3 (Governance/dispatch-time re-evaluation — commit/push/PR/merge are each distinct governed actions), and Phase 4 Slices 2–3 (M&E `Evaluation`/`PerformanceProfile` and Finance `ResourceConstraint`, both consumed read-only by the router).

- [ ] **Slice 0 — Write and approve `docs/departments/engineering.md`.** Currently `NOT YET SPECIFIED` — Engineering's department mission/responsibility/authority doc doesn't exist yet even though its architecture docs are approved.
- [ ] **Slice 1 — Workspace lifecycle for one isolated environment.** Create/lease/checkpoint/recover/destroy per [`docs/domain/workspace.md`](docs/domain/workspace.md)'s canonical lifecycle and `workspaces.md`'s provider/manager contracts and recovery/reconciliation section. Select the provider mechanism per Phase 2 Slice 4's `workspace-isolation.md`; if that doc leaves it open, carry the open question forward rather than choosing unilaterally.
- [ ] **Slice 2 — `CodingAgentRouter` minimal eligibility/routing for one coding-agent provider.** Eligibility-first filter (capabilities, language/repo fit, tool support, sandbox compatibility, Finance constraint outcomes, evidence freshness) for exactly one registered `CodingAgentProfile`, persisting the routing decision before dispatch. Initial ranking runs with no historical `PerformanceProfile` (M&E evidence doesn't exist yet on the first pass) — note that explicitly rather than fabricating evidence.
- [ ] **Slice 3 — Governed commit → push → PR pipeline.** `workspaces.md`'s Git discipline (clean checkout, feature branch, diff/status tracking, seal-before-result) plus commit/push/PR-preparation as separate governed actions with least-privilege, non-merge credentials.
- [ ] **Slice 4 — Independent review, lead review, and the governance merge gate.** `Independent → Lead → Gate → Merge`; the implementation agent cannot satisfy the independent-review requirement for its own result. `coding-agents.md`'s open question on the required independence boundary between implementation/independent/lead review is unresolved — carry it forward, don't invent a specific reviewer-assignment rule.
- [ ] **Slice 5 — `CodingAgentEvaluation`.** The M&E-owned specialization of [`docs/domain/evaluation.md`](docs/domain/evaluation.md) with subject type `CODING_AGENT_PROFILE`, feeding the `PerformanceProfile` Slice 2's router should eventually consume. Depends on Phase 4 Slice 2.

## Phase 7 — Remaining departments: Design, Deployment, Education & Engagement

All three department docs are `NOT YET SPECIFIED` and don't exist as files yet. Same gate pattern as Phase 4: write and get each `APPROVED` before implementing against it.

- [ ] **Slice 1 — Write and approve `docs/departments/design.md`.** Design responsibilities and outputs. No existing draft to build from.
- [ ] **Slice 2 — Write and approve `docs/departments/deployment.md`.** Deployment responsibilities and authority. **This is the *governed deployment lifecycle*** — who authorizes a release and under what policy (see `docs/workflows/deployment.md`, also `NOT YET SPECIFIED`) — distinct from Phase 9's *infrastructure execution* of deploying `companyd`/`web`, which `ADR-0004` already fixed the topology for and doesn't wait on this doc. Author this so Phase 9's *ongoing* (not first) deploys run under a defined governance owner, rather than duplicating deployment authorization logic ad hoc.
- [ ] **Slice 3 — Write and approve `docs/departments/education.md`.** Education and public-engagement responsibilities. Tie to `docs/workflows/education-publishing.md` (also `NOT YET SPECIFIED`, governed public-content publishing) — author both together, one is meaningless without the other.
- [ ] **Slice 4 — Stand up minimal execution for each approved department.** Same shared-contract pattern as Phase 4 (no direct department-to-department calls — only events, workflows, capabilities, artifacts/evidence, and canonical Evaluation/Metric/ResourceConstraint contracts). Scope per department is undecided until its doc is written — do not invent scope ahead of Slices 1–3.

## Phase 8 — CI/CD growth

Grounded in what `.github/workflows/ci.yml` actually runs today: two jobs, `companyd` (`go vet ./...`, `go test ./...`) and `web` (`npm install`, `npm run build`) — no database service, no e2e, no contract tests, no deploy step. Depends on Phase 2 Slices 5–6 (`testing/strategy.md`, `testing/contract-tests.md`). The deploy step is deliberately excluded from this phase (see Phase 9 Slice 6) so CI growth doesn't depend on production infrastructure existing first.

- [ ] **Slice 1 — Ephemeral Postgres/Supabase service in CI for real integration tests.** Add a database service (or the Supabase CLI's local stack) to the `companyd` CI job so integration tests exercise the real persistence adapters, not just unit tests with fakes.
- [ ] **Slice 2 — Contract tests per `docs/testing/contract-tests.md`.** Once that doc exists (Phase 2 Slice 6), add a CI job verifying adapter/port conformance — starting with today's persistence adapters, extending to the Identity `Authenticator` (Phase 3) and `WorkspaceProvider`/`CodingAgentRuntime` adapters (Phase 6) as they land.
- [ ] **Slice 3 — Web e2e tests.** Cover the login UI (Phase 3 Slice 4) and the `WorkflowTrigger` create→start→poll flow against a real ephemeral backend, per `docs/testing/strategy.md`'s eventual test-level definitions.

## Phase 9 — Production deployment

Grounded in `ADR-0004`'s already-decided topology (`Status: APPROVED`): `companyd` → DigitalOcean VPS + systemd; `web` → Vercel; Supabase remains a managed hosted project.

- [ ] **Slice 1 — Provision the DigitalOcean VPS.** Per `ADR-0004`. Firewall/network exposure to Supabase is explicitly implementation detail per the ADR, not an architectural decision — this slice makes that implementation choice, it isn't re-litigating the ADR.
- [ ] **Slice 2 — systemd unit + restart/backoff configuration.** The external-supervision layer `ADR-0004` names, distinct from Runtime's own operation-level retry (`execution.md`). Same implementation-detail caveat as Slice 1.
- [ ] **Slice 3 — Secrets management for `companyd`.** Supabase service credentials, LLM provider keys, any Phase 3 Governance/Approval signing material. No doc mandates a mechanism — tooling judgment, not a doc-derived decision.
- [ ] **Slice 4 — Supabase RLS policies for real multi-tenant isolation.** `ADR-0004`'s own open question. Phase 1 deliberately shipped with **zero RLS policies** — safe only because `internal/fixtures.Registry` hardcoded a single Organization. No longer safe once Phase 3 Slice 6 (retiring the fixture Organization) admits real multi-org data to the same database. **Must land before any second real organization's data is admitted to production** — sequence after Phase 3 Slice 6, before general availability.
- [ ] **Slice 5 — Vercel production deploy for `web`.** Per `ADR-0004`. Configure production environment variables (`NEXT_PUBLIC_SUPABASE_URL`, publishable key, `companyd` base URL); confirm the existing `npm run build` CI job (Phase 8) is what ships.
- [ ] **Slice 6 — CI deploy-step automation.** Extend `.github/workflows/ci.yml` to deploy `companyd` (systemd restart over SSH, or equivalent) and `web` (Vercel) on merge to `main`, now that Slices 1–2 and 5 give it somewhere to deploy to.
- [ ] **Slice 7 — "Production-ready" checklist.** A gate, not a single artifact: real Human authentication enforced (Phase 3 Slices 3–4); Supabase RLS active for multi-org isolation (Slice 4); Governance `DENY`/`REQUIRE_APPROVAL` exercised for real traffic (Phase 3 Slices 1–2); secrets managed and rotatable (Slice 3); Postgres backups exist for the hosted Supabase project; monitoring/alerting exists to whatever extent M&E (Phase 4 Slice 2) has produced by this point — this last item may still be partial, since M&E's own production-monitoring maturity is itself ongoing work, not something this phase can force to completion.

Once `docs/departments/deployment.md` (Phase 7 Slice 2) is written and approved, it becomes the governance owner for *authorizing* future releases through this pipeline. This phase's first production deploy doesn't wait for that — `ADR-0004` already authorized the topology — but later routine deploys should route through whatever governed action that department doc eventually defines.

## Beyond Phase 9

- The full evidence-based Intelligence Router [ADR-0003](docs/adr/ADR-0003-model-independent-intelligence.md) describes (Task Analyzer, Governance-eligible ranking, Finance budget, M&E evidence, persisted `RoutingDecision`) — `ADR-0003` itself is `APPROVED`; only this richer routing layer is undated future work. The first slice's `internal/adapters/intelligence/fallback` is a narrower, purely operational precursor: fixed-priority provider selection with rate-limit/outage cooldown, not evidence-based routing.

## Open questions

- Which components run in one process versus separate deployment boundaries for the first Daemon? (see [Daemon — Open questions](docs/architecture/daemon.md#open-questions))
- Phase-specific open questions carried forward from canonical docs are listed inline within Phases 4–6 above rather than duplicated here.

## Dependencies

- [README.md — Roadmap](README.md#roadmap) (summary shown to readers first)
- [.companyos/agent-memory/current-state.md](.companyos/agent-memory/current-state.md) (live status; authoritative over this file)
- [docs/adr/README.md](docs/adr/README.md)
