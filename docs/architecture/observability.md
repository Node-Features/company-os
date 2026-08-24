# CompanyOS Observability Architecture

Status: APPROVED (2026-08-24)

## Responsibility

Observability gives every execution a stable correlation identity and makes
its progress through the system inspectable — via structured logs and
operational metrics — without becoming a second source of organizational
truth. It answers `ADR-0006`'s open question ("no structured logging,
metrics, or tracing exists today — is this in scope?") and implements the
"observability wiring" [`daemon.md`](daemon.md) has named as a Daemon
responsibility since its own approval, unimplemented until now.

Observability owns:

- a correlation-identity model propagated through `context.Context` across
  HTTP, Application, Governance, and Runtime;
- structured (key-value, not string-interpolated) logging of that identity
  and the lifecycle state at each stage;
- operational metrics — counts, durations, backlogs — over Runtime, the
  outbox sweeper, and Governance decisions;
- redaction rules for what may never appear in a log record or metric
  label.

Observability does **not** own:

- organizational truth of any kind. A log line or metric is diagnostic
  output derived from an already-committed decision; it never causes,
  represents, or substitutes for one. See Invariants.
- M&E's `Metric`/`Evaluation`/`PerformanceProfile` domain
  ([`domain/metric.md`](../domain/metric.md),
  [`domain/evaluation.md`](../domain/evaluation.md)) — that is organizational
  measurement of AI capability *quality*, owned by the Monitoring &
  Evaluation department, with its own confidence/provenance/independence
  contract. This document's "metric" means an infrastructure counter or
  histogram (executions, retries, latency) and never overlaps that domain's
  vocabulary or storage. `persistence.md`'s "Metrics" record class
  (M&E/observability definition owner) refers to the M&E domain concept;
  operational metrics defined here are not persisted as CompanyOS records at
  all — they are ephemeral process state exported for scraping, same
  authority class as the `event_outbox`/provider-cooldown state
  `findings.md` §5 already classifies as disposable.
- distributed tracing/spans across process boundaries. `apps/companyd` is
  one process today (`ADR-0004`); see Open questions.
- `apps/web`'s own logging, or any correlation-ID propagation across the
  `web` → `companyd` HTTP boundary. Scoped to `apps/companyd` only this
  slice.

## Correlation identity

Every domain envelope already carries the IDs needed to correlate a command
through its full lifecycle — this document propagates and surfaces them
through `context.Context` and log/metric output; it does not mint a second,
competing ID scheme.

| Field | Source (already exists) |
|---|---|
| CorrelationID | `command.WorkflowCommandEnvelope.CorrelationID` |
| CommandID | `command.WorkflowCommandEnvelope.CommandID` |
| OrganizationID | every envelope/aggregate |
| WorkflowID | `workflow.Workflow` / `ExecutionIntent.WorkflowID` |
| ExecutionIntentID | `workflow.ExecutionIntent.IntentID` |
| ExecutionAttemptID | `execution.ExecutionAttempt.AttemptID` |
| PrincipalID | `RequestingPrincipalID` / the resolved `principal.Principal` |
| GovernanceDecisionID, Outcome | `policy.GovernanceDecision.DecisionID`, `.Outcome` |
| Provider | `result.Result.ProviderAdapter` / `Runtime.ProviderName` |
| RetryCount | `execution.ExecutionAttempt.AttemptNumber` |
| LeaseState | derived from `LeaseExpiresAt` / `LeaseFencingToken` / `AttemptStatus` |
| FailureReason | `result.Result.ErrorClassification` / `command.RejectionReasons` |
| LifecycleState | `execution.AttemptStatus` / `workflow.State` |
| StartedAt, CompletedAt | the record's own existing timestamp fields |

The identity is assembled progressively, never invented ahead of the data
that would populate it: the HTTP layer seeds a fresh `CorrelationID` and,
once resolved, `PrincipalID`/`OrganizationID`; an Application use case
overwrites `CorrelationID` with the envelope's own value (the domain's
`CorrelationID`, not the HTTP layer's provisional one, is what's threaded
through Governance/Runtime/persistence) and adds `CommandID`/`WorkflowID`;
Governance's decision adds
`GovernanceDecisionID`/`Outcome`; Runtime's claim adds
`ExecutionIntentID`/`ExecutionAttemptID`/`Provider`/`RetryCount`/
`LeaseState`. A field absent at a given stage is simply omitted from that
stage's log record, not synthesized.

`internal/observability.ExecutionContext` is the Go type carrying this;
`WithExecutionContext`/`FromContext` move it through `context.Context`, the
same mechanism `httpapi.EvidenceFromContext`/`PrincipalFromContext` already
use for authenticated evidence and the resolved Principal — this is an
additional, independent context value, not a replacement for those.

## Structured logging

`log/slog` (Go standard library since 1.21; `apps/companyd` already targets
1.25 — zero new dependency). Chosen over a third-party logging library
because `slog`'s `Handler` interface is itself the vendor-neutral
abstraction: swapping output format or destination later is a `Handler`
change, not a call-site change, and every call site already depends only on
`*slog.Logger`.

- `internal/observability.Init(w io.Writer, level slog.Level)` configures
  the process-wide handler once, at boot (`main.go` boot stage 3,
  `LOG_LEVEL` env var, default `info`). JSON output — this is a service log,
  read by tooling, not a human terminal.
- `internal/observability.Logger(ctx) *slog.Logger` returns the configured
  logger enriched with `FromContext(ctx).Attrs()` — every log line emitted
  through it automatically carries whatever correlation identity is known
  at that point, as structured attributes, never interpolated into the
  message string.
- Every existing `log.Print`/`log.Printf`/`log.Println` call site in
  `apps/companyd` (`main.go`, `runtime.go`, `application.go`,
  `sweeper.go`, `fallback/adapter.go`, `httpapi/principal.go`) is replaced
  by a call through this logger. `cmd/migrate` (a one-shot CLI, not the
  daemon) is unaffected — its `fmt.Println` narration is untouched.

## Metrics

No metrics library or `/metrics` endpoint exists today. New
`ports.MetricsRecorder` port:

```go
type MetricsRecorder interface {
    IncrCounter(name string, labels map[string]string)
    ObserveHistogram(name string, seconds float64, labels map[string]string)
    SetGauge(name string, value float64, labels map[string]string)
}
```

Implemented by `internal/adapters/observability/prometheus.Recorder`, using
`github.com/prometheus/client_golang` — the OSS reference implementation of
the Prometheus exposition format, which every major observability vendor
(and OpenTelemetry Collector) already knows how to scrape. Chosen over
pulling in the full OpenTelemetry SDK because no concrete responsibility
here needs OTel's additional machinery yet (multi-exporter routing,
resource detection, span propagation across processes) — `AGENTS.md`:
"introduce infrastructure only for a concrete responsibility." Vendor
neutrality is enforced at the code boundary, not just the wire format:
`Application`, `Runtime`, and the outbox `Sweeper` depend only on
`ports.MetricsRecorder`; `client_golang` is imported by exactly one adapter
package, the same shape as `ports.ProviderAdapter` → the
Gemini/OpenAI/Anthropic adapters.

The Recorder pre-registers a fixed metric vocabulary at construction —
metrics are never created dynamically from request data, which is what
keeps Prometheus label cardinality bounded:

| Metric | Type | Labels | Emitted from |
|---|---|---|---|
| `governance_decisions_total` | Counter | `outcome`, `action` | every Governance evaluation call site, right after `SaveGovernanceDecision` commits |
| `executions_total` | Counter | `state` — the exact `execution.AttemptStatus` value (`CLAIMED`, `SUCCEEDED`, `FAILED_RETRYABLE`, `FAILED_TERMINAL`, `CANCELLED`, `LEASE_EXPIRED`) | Runtime claim and terminal recording — reuses the domain's own bounded enum as the label instead of inventing a second vocabulary |
| `retries_total` | Counter | — | Runtime retry scheduling (provider failure or lease-reclaim path) |
| `abandoned_total` | Counter | — | Runtime `failExhausted` (lease-reclaim retries exhausted) |
| `lease_expirations_total` | Counter | — | Runtime `reclaimAbandoned` |
| `provider_latency_seconds` | Histogram | `provider`, `outcome` (`succeeded`\|`failed`) | around `Runtime.Provider.Generate` |
| `approval_latency_seconds` | Histogram | — | `ResolveApproval` (`decided_at` − `PendingCommand.CreatedAt`) |
| `outbox_backlog` | Gauge | — | outbox `Sweeper.Sweep`, set to the unpublished count it just loaded |
| `reconciliation_runs_total` | Counter | — | outbox `Sweeper.Sweep`, once per pass |

`GET /metrics` is mounted unauthenticated, like `/health` — it exposes
operational counters and durations, never business data, subject IDs, or
content.

## Redaction

The following must never appear in a log record or metric label:

- raw JWTs or bearer tokens (`Authorization` header values);
- provider API keys or any request/response header carrying credentials;
- full provider request/response bodies — `internal/observability.SafeProviderError`
  classifies and truncates a provider error instead of logging it verbatim,
  closing the gap found in `fallback/adapter.go`'s prior `%v`-formatted
  provider-error logging, which could echo back a safety-filter or
  rate-limit message that itself quotes prompt content;
- `principal.AuthenticatedEvidence.Subject`/`.Email` unmasked — these are
  PII; if a future need arises to correlate by them, log a hash, not the
  raw value;
- a raw connection-establishment error string without confirming it cannot
  echo a DSN/connection string (`main.go`'s `supabase.Connect`/
  `supabaseauth.New`/`NewRegistryFromDB` error paths).

## Non-authority invariant

Restates and extends [`persistence.md`](persistence.md)'s existing list —
"Conversation, cache, search index, vector index, provider state,
checkpoint, and message history are never authoritative business state" —
to explicitly include logs and metrics, and restates
[`domain/event.md`](../domain/event.md)'s "Agent messages, provider
callbacks, logs, and telemetry become DomainEvents only through their
owning Application/Kernel operation." A log line or metric observation:

- is emitted only *after* the domain operation it describes has already
  committed (or, for a decision, after `SaveGovernanceDecision` has already
  persisted it) — never before, and never as a substitute for that
  persistence;
- cannot be replayed to reconstruct or repair authoritative state; recovery
  uses persisted `ExecutionAttempt`/`Lease`/`Checkpoint`/`Result` records
  per [`domain/execution.md`](../domain/execution.md), never log output;
- carries no fencing token, version, or compare-and-swap semantics of its
  own — it is not a locking or coordination mechanism;
- may be lost (dropped log line, process restart resetting an in-memory
  Prometheus counter) without corrupting or misrepresenting organizational
  truth, since every value it reports is derivable again from persisted
  records.

## OSS evidence

| Reference | Observed pattern | CompanyOS use / rejection |
|---|---|---|
| Go standard library [`log/slog`](https://pkg.go.dev/log/slog) | `Handler` interface separates structured record construction from output format/destination; `Logger.With` attaches persistent attributes | Borrow the `Handler` abstraction as the vendor-neutral logging boundary and `.With` for correlation-attribute propagation; reject adding a third-party logging framework where the stdlib abstraction already does the job |
| [Prometheus client_golang](https://github.com/prometheus/client_golang) | `Counter`/`Histogram`/`Gauge` collectors registered once, scraped via `/metrics` in a stable text format | Borrow the collector types and exposition format as the metrics wire contract; reject importing it outside one adapter package |
| [OpenTelemetry semantic conventions](https://opentelemetry.io/docs/specs/semconv/) | Named attribute conventions (`trace_id`, RED — rate/errors/duration — metrics) for cross-vendor interoperability | Borrow the naming discipline (stable, documented attribute/metric names) and the RED shape reflected in `executions_total`/`provider_latency_seconds`; reject adopting the full OTel SDK now — no concrete responsibility here needs multi-exporter routing or cross-process span propagation yet |

## Open questions

- Distributed tracing/span propagation once a second `companyd` process or
  worker exists — depends on [`node.md`](node.md)'s still-open
  single-vs-multi-process question. A single process's `ExecutionContext`
  correlation model is sufficient until that question resolves.
- Whether/how the `web` → `companyd` HTTP boundary should propagate a
  correlation header so a browser-triggered request's logs can be joined
  across both services — not built this slice.
- Log shipping/retention destination (stdout-only today vs. an external
  aggregator) is a Phase 9 deployment decision, not decided here.
- Whether `/metrics` needs authentication once `companyd` is
  internet-reachable in production (today it is not) — carried forward for
  Phase 9's production-readiness checklist.

## Dependencies

- [Top-level architecture](../../ARCHITECTURE.md)
- [Daemon](daemon.md)
- [Runtime](runtime.md)
- [Persistence](persistence.md)
- [Event](../domain/event.md)
- [Execution domain](../domain/execution.md)
- [Command domain](../domain/command.md)
- [Governance](governance.md)
- [Metric domain](../domain/metric.md) — the distinct M&E concept this document is not
- [ADR-0006](../adr/ADR-0006-daemon-boot-sequence.md) — the open question this document resolves
