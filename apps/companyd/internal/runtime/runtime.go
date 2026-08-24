package runtime

import (
	"context"
	"log"
	"math/rand"
	"runtime/debug"
	"sync"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/capability"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/execution"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

const workerID = "companyd-runtime"

// defaultMaxConcurrentDispatch is used when Runtime.MaxConcurrentDispatch is
// left unset (zero value) — see that field's doc comment.
const defaultMaxConcurrentDispatch = 10

// Runtime is the 8-step execution model in docs/architecture/runtime.md:
// claim due intent under lease, dispatch, capture evidence, submit a
// proposed Result through Application — never advancing Workflow state
// itself. Wakeup is the in-process notification hint (first-slice plan
// decision #8); PollInterval is the durable fallback sweep.
type Runtime struct {
	Exec ports.ExecutionRepository
	App  *application.Application
	// Provider is the first-slice IntelligenceCapability's dispatch port
	// (ADR-0004) — provider-independent by construction (ADR-0003).
	// cmd/companyd/main.go wires in internal/adapters/intelligence/fallback's
	// composite adapter here, so a single dispatch can transparently try
	// several concrete providers in priority order on rate-limit/outage.
	// ProviderName/ModelID are only the label used when a dispatch fails
	// before any provider produced a result to read Provider/ModelID from
	// (ports.IntelligenceResult); a successful call always reports the
	// actual provider that served it.
	Provider      ports.ProviderAdapter
	ProviderName  string
	ModelID       string
	Fixtures      fixtures.Registry
	PollInterval  time.Duration
	LeaseDuration time.Duration
	Wakeup        <-chan uuid.UUID
	// Notifier is an optional best-effort ping to web's per-Workflow
	// realtime channel on each ExecutionUnit (ExecutionIntent/
	// ExecutionAttempt) transition — CLAIMED, DISPATCHED, terminal — none
	// of which produce a DomainEvent of their own. A nil Notifier is never
	// an error; the receiver's reconciliation poll is the durable
	// fallback, same as Application.Notify's in-process wake-up hint.
	Notifier ports.ChangeNotifier

	// MaxConcurrentDispatch bounds how many execute() calls may run at once,
	// across all Sweeps combined — independent of PollInterval/batch size,
	// so a pile-up of overlapping sweeps under a slow-provider window can't
	// drive unbounded concurrent provider calls or DB connections
	// (docs/audit/gap-runtime-resilience.md). Zero/negative uses
	// defaultMaxConcurrentDispatch. This alone is why overlapping Sweep
	// calls need no separate in-flight-sweep guard: claiming is already
	// exclusive per row (FOR UPDATE SKIP LOCKED), and execution concurrency
	// is capped here regardless of how many sweeps' goroutines are queued
	// waiting for a slot.
	MaxConcurrentDispatch int

	wg  sync.WaitGroup
	sem chan struct{}

	// workCtx is the context passed to execute() for every in-flight
	// dispatch — deliberately NOT the ctx passed to Start. Start's ctx
	// governs the poll loop only (stop claiming new work on shutdown
	// signal); workCtx lives independently so that signal does not also
	// cancel dispatches already in flight, which would otherwise fail every
	// one of them on context.Canceled the instant a routine deploy sends
	// SIGTERM — turning an ordinary restart into the same abandonment
	// lease-expiry exists to recover from. workCancel lets Shutdown give up
	// waiting after its bound elapses without leaking the context forever.
	// workOnce makes lazy init race-safe regardless of whether Start,
	// Sweep, or StopWork happens to run first.
	workOnce   sync.Once
	workCtx    context.Context
	workCancel context.CancelFunc
}

func (r *Runtime) ensureWorkContext() {
	r.workOnce.Do(func() {
		r.workCtx, r.workCancel = context.WithCancel(context.Background())
		limit := r.MaxConcurrentDispatch
		if limit <= 0 {
			limit = defaultMaxConcurrentDispatch
		}
		r.sem = make(chan struct{}, limit)
	})
}

// dispatchBounded runs fn on its own goroutine, tracked by r.wg (so Wait
// drains it) and gated by r.sem (so no more than MaxConcurrentDispatch run
// concurrently, regardless of how many Sweeps overlap). The slot is
// acquired inside the goroutine, after wg.Add, so a call still queued
// behind a full semaphore counts as in-flight work Shutdown must drain —
// not work that was silently dropped.
//
// It recovers any panic from fn, logging it instead of letting it crash the
// whole process — a single malformed provider response or unhandled edge
// case in one dispatch no longer takes every other concurrently-dispatching
// intent down with it (docs/audit/gap-runtime-resilience.md). No separate
// "panicked" bookkeeping is needed: a panicked execute() call leaves its
// ExecutionAttempt wherever it had gotten to (CLAIMED or DISPATCHED, lease
// still ticking), and reclaimAbandoned's normal lease-expiry sweep recovers
// it exactly like a crashed process — the same recovery path, not a second
// one.
func (r *Runtime) dispatchBounded(fn func()) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.sem <- struct{}{}
		defer func() { <-r.sem }()
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("runtime: recovered panic in dispatch: %v\n%s", rec, debug.Stack())
			}
		}()
		fn()
	}()
}

// StopWork cancels the context passed to in-flight execute() calls. Called
// by Daemon.Shutdown once its bounded wait for Wait() has elapsed, so any
// dispatch still running past the shutdown deadline at least observes
// cancellation on its next context-aware call rather than running
// unbounded — it cannot be force-killed, only asked to stop. Safe to call
// before Start or multiple times.
func (r *Runtime) StopWork() {
	r.ensureWorkContext()
	r.workCancel()
}

// notifyChanged is a non-blocking best-effort ping — errors are logged, not
// surfaced, since ports.ChangeNotifier's contract is a recoverable hint,
// never a dependency (mirrors ports.Publisher's same contract).
func (r *Runtime) notifyChanged(ctx context.Context, workflowID uuid.UUID) {
	if r.Notifier == nil {
		return
	}
	if err := r.Notifier.NotifyWorkflowChanged(ctx, workflowID); err != nil {
		log.Printf("runtime: notify workflow changed: %v", err)
	}
}

// Start runs the poll-and-wake loop until ctx is cancelled. ctx governs
// claiming new work only — see workCtx's doc comment for why in-flight
// dispatches use a separately-lived context instead.
func (r *Runtime) Start(ctx context.Context) {
	r.ensureWorkContext()
	ticker := time.NewTicker(r.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.Sweep(ctx)
		case <-r.Wakeup:
			r.Sweep(ctx)
		}
	}
}

// Wait blocks until every in-flight execute() call started by the most
// recent Sweep has returned — used by Daemon.Shutdown to drain bounded
// in-flight work.
func (r *Runtime) Wait() { r.wg.Wait() }

// Sweep reclaims abandoned leases, then claims due intents and dispatches
// each in its own goroutine. ctx bounds the claim/reclaim queries
// themselves (correct to abandon these on shutdown — declining to claim
// more work is exactly "stop accepting new work"); dispatched goroutines
// run on r.workCtx instead, independent of ctx's cancellation.
func (r *Runtime) Sweep(ctx context.Context) {
	r.ensureWorkContext()
	r.reclaimAbandoned(ctx)

	claims, err := r.Exec.ClaimDueIntents(ctx, r.Fixtures.Organization().OrganizationID, 10, r.LeaseDuration, workerID)
	if err != nil {
		log.Printf("runtime: claim due intents: %v", err)
		return
	}
	for _, c := range claims {
		r.notifyChanged(ctx, c.Intent.WorkflowID)
		c := c
		r.dispatchBounded(func() { r.execute(r.workCtx, c.Attempt, c.Intent) })
	}
}

// reclaimAbandoned recovers ExecutionAttempts whose lease expired without a
// terminal report — a worker that crashed, was killed, or lost its process
// entirely, per invariant 7 ("lost workers must not permanently strand
// work"). Reclaiming is itself bounded (limit 10, same as ClaimDueIntents'
// batch size) so a large backlog of abandoned work is worked off
// incrementally across sweeps rather than in one large transaction. Each
// reclaimed attempt already has its lease_fencing_token invalidated by
// ports.ExecutionRepository.ReclaimExpiredLeases before this method ever
// sees it — the only remaining decision is retry-vs-exhausted, which needs
// capability.RetryPolicy (a Go value from Fixtures), not something the
// persistence layer should decide.
func (r *Runtime) reclaimAbandoned(ctx context.Context) {
	reclaimed, err := r.Exec.ReclaimExpiredLeases(ctx, r.Fixtures.Organization().OrganizationID, 10)
	if err != nil {
		log.Printf("runtime: reclaim expired leases: %v", err)
		return
	}
	capDef := r.Fixtures.Capability()
	for _, c := range reclaimed {
		log.Printf("runtime: reclaimed lease-expired attempt %s (intent %s, workflow %s, attempt #%d)",
			c.Attempt.AttemptID, c.Intent.IntentID, c.Intent.WorkflowID, c.Attempt.AttemptNumber)
		if c.Attempt.AttemptNumber < capDef.Retry.MaxAttempts {
			backoff := computeBackoff(c.Attempt.AttemptNumber, capDef.Retry)
			dueAt := time.Now().UTC().Add(backoff)
			if err := r.Exec.ScheduleRetry(ctx, c.Intent.OrganizationID, c.Intent.IntentID, dueAt); err != nil {
				log.Printf("runtime: schedule retry after lease reclaim: %v", err)
			}
			r.notifyChanged(ctx, c.Intent.WorkflowID)
			continue
		}
		r.failExhausted(ctx, c.Attempt, c.Intent)
	}
}

// failExhausted submits a synthetic FAILED Result for an ExecutionIntent
// whose lease has now expired capDef.Retry.MaxAttempts times in a row —
// consistently unable to complete, not just unlucky once. This is the same
// terminal shape a real provider failure takes in execute() (SaveResult
// then Application.SubmitResult so the Workflow actually transitions to
// FAILED via the governed REJECT_WORKFLOW_RESULT path), with one
// deliberate difference: it does NOT call RecordTerminal on this attempt.
// ReclaimExpiredLeases already transitioned it to LEASE_EXPIRED, which is
// already a terminal AttemptStatus (execution.AttemptStatus.IsTerminal) —
// re-marking it FAILED_TERMINAL here would not be a legal transition per
// execution.AttemptStatus.CanTransitionTo's table (nothing transitions out
// of LEASE_EXPIRED) and would falsify the historical record: this specific
// attempt didn't fail, it was never heard from again, which LEASE_EXPIRED
// says truthfully and FAILED_TERMINAL would not.
func (r *Runtime) failExhausted(ctx context.Context, attempt execution.ExecutionAttempt, intent workflow.ExecutionIntent) {
	now := time.Now().UTC()
	resultID := uuid.New()
	errClass := "lease_expired_max_attempts"
	retryable := false
	res := &result.Result{
		ResultID:             resultID,
		OrganizationID:       intent.OrganizationID,
		ResultType:           "INTELLIGENCE_TEXT_GENERATION",
		WorkflowID:           intent.WorkflowID,
		ObjectiveID:          r.Fixtures.Objective().ObjectiveID,
		IntentID:             intent.IntentID,
		AttemptID:            attempt.AttemptID,
		CapabilityRequestID:  attempt.CapabilityRequestID,
		IdempotencyKey:       intent.IdempotencyKey + ":" + resultID.String(),
		ProducingPrincipalID: r.Fixtures.TriggerPrincipal().PrincipalID,
		ProviderAdapter:      r.ProviderName,
		ModelID:              r.ModelID,
		Outcome:              result.OutcomeFailed,
		ErrorClassification:  &errClass,
		Retryable:            &retryable,
		StartedAt:            attempt.CreatedAt,
		ObservedAt:           now,
		ReportedAt:           now,
	}
	if err := r.Exec.SaveResult(ctx, res); err != nil {
		log.Printf("runtime: save result for exhausted lease-expired intent %s: %v", intent.IntentID, err)
		return
	}
	r.notifyChanged(ctx, intent.WorkflowID)
	outcome := r.App.SubmitResult(ctx, application.SubmitResultRequest{
		RequestID:       uuid.New(),
		IdempotencyKey:  res.IdempotencyKey,
		ResultID:        res.ResultID,
		ExpectedVersion: intent.WorkflowVersion,
	})
	if outcome.Outcome != application.Accepted {
		log.Printf("runtime: submit result for exhausted lease-expired workflow %s: %s %v", intent.WorkflowID, outcome.Outcome, outcome.Reasons)
	}
}

func (r *Runtime) execute(ctx context.Context, attempt execution.ExecutionAttempt, intent workflow.ExecutionIntent) {
	decision, err := r.App.AuthorizeDispatch(ctx, intent)
	if err != nil {
		log.Printf("runtime: authorize dispatch: %v", err)
		return
	}
	if !decision.Outcome.Allows() {
		// REQUIRE_APPROVAL/DENIED/stale-version blocks dispatch
		// (application.md#dispatch-time-governance). This slice's
		// always-AUTOMATIC policy makes this a rare race (a stale version),
		// so it's treated as terminal rather than looping.
		log.Printf("runtime: dispatch not authorized for intent %s: %s", intent.IntentID, decision.Outcome)
		_ = r.Exec.RecordTerminal(ctx, attempt.AttemptID, *attempt.LeaseFencingToken, execution.StatusFailedTerminal, nil)
		r.notifyChanged(ctx, intent.WorkflowID)
		return
	}

	if err := r.Exec.RecordDispatched(ctx, attempt.AttemptID, *attempt.LeaseFencingToken, attempt.AttemptID.String()); err != nil {
		log.Printf("runtime: record dispatched: %v", err)
		return
	}
	r.notifyChanged(ctx, intent.WorkflowID)

	capDef := r.Fixtures.Capability()
	dispatchCtx, cancel := context.WithTimeout(ctx, capDef.Timeout)
	defer cancel()

	prompt, _ := intent.Inputs["prompt"].(string)
	maxTokens := 256
	if v, ok := intent.Inputs["maxOutputTokens"].(float64); ok {
		maxTokens = int(v)
	} else if v, ok := intent.Inputs["maxOutputTokens"].(int); ok {
		maxTokens = v
	}

	genResult, genErr := r.Provider.Generate(dispatchCtx, ports.IntelligenceRequest{Prompt: prompt, MaxOutputTokens: maxTokens})

	now := time.Now().UTC()
	resultID := uuid.New()

	if genErr == nil {
		res := &result.Result{
			ResultID:             resultID,
			OrganizationID:       intent.OrganizationID,
			ResultType:           "INTELLIGENCE_TEXT_GENERATION",
			WorkflowID:           intent.WorkflowID,
			ObjectiveID:          r.Fixtures.Objective().ObjectiveID,
			IntentID:             intent.IntentID,
			AttemptID:            attempt.AttemptID,
			CapabilityRequestID:  attempt.CapabilityRequestID,
			IdempotencyKey:       intent.IdempotencyKey + ":" + resultID.String(),
			ProducingPrincipalID: r.Fixtures.TriggerPrincipal().PrincipalID,
			ProviderAdapter:      genResult.Provider,
			ModelID:              genResult.ModelID,
			Outcome:              result.OutcomeSucceeded,
			Output: map[string]any{
				"text":         genResult.Text,
				"inputTokens":  genResult.Usage.InputTokens,
				"outputTokens": genResult.Usage.OutputTokens,
			},
			StartedAt:  attempt.CreatedAt,
			ObservedAt: now,
			ReportedAt: now,
		}
		r.submitResult(ctx, attempt, intent, res, execution.StatusSucceeded)
		return
	}

	retryable := ports.IsRetryable(genErr)
	if retryable && attempt.AttemptNumber < capDef.Retry.MaxAttempts {
		if err := r.Exec.RecordTerminal(ctx, attempt.AttemptID, *attempt.LeaseFencingToken, execution.StatusFailedRetryable, nil); err != nil {
			log.Printf("runtime: record failed_retryable: %v", err)
			return
		}
		backoff := computeBackoff(attempt.AttemptNumber, capDef.Retry)
		if err := r.Exec.ScheduleRetry(ctx, intent.OrganizationID, intent.IntentID, now.Add(backoff)); err != nil {
			log.Printf("runtime: schedule retry: %v", err)
		}
		r.notifyChanged(ctx, intent.WorkflowID)
		return
	}

	errClass := genErr.Error()
	res := &result.Result{
		ResultID:             resultID,
		OrganizationID:       intent.OrganizationID,
		ResultType:           "INTELLIGENCE_TEXT_GENERATION",
		WorkflowID:           intent.WorkflowID,
		ObjectiveID:          r.Fixtures.Objective().ObjectiveID,
		IntentID:             intent.IntentID,
		AttemptID:            attempt.AttemptID,
		CapabilityRequestID:  attempt.CapabilityRequestID,
		IdempotencyKey:       intent.IdempotencyKey + ":" + resultID.String(),
		ProducingPrincipalID: r.Fixtures.TriggerPrincipal().PrincipalID,
		ProviderAdapter:      r.ProviderName,
		ModelID:              r.ModelID,
		Outcome:              result.OutcomeFailed,
		ErrorClassification:  &errClass,
		Retryable:            &retryable,
		StartedAt:            attempt.CreatedAt,
		ObservedAt:           now,
		ReportedAt:           now,
	}
	r.submitResult(ctx, attempt, intent, res, execution.StatusFailedTerminal)
}

func (r *Runtime) submitResult(ctx context.Context, attempt execution.ExecutionAttempt, intent workflow.ExecutionIntent, res *result.Result, terminal execution.AttemptStatus) {
	if err := r.Exec.SaveResult(ctx, res); err != nil {
		log.Printf("runtime: save result: %v", err)
		return
	}
	if err := r.Exec.RecordTerminal(ctx, attempt.AttemptID, *attempt.LeaseFencingToken, terminal, &res.ResultID); err != nil {
		log.Printf("runtime: record terminal: %v", err)
		return
	}
	r.notifyChanged(ctx, intent.WorkflowID)
	outcome := r.App.SubmitResult(ctx, application.SubmitResultRequest{
		RequestID:       uuid.New(),
		IdempotencyKey:  res.IdempotencyKey,
		ResultID:        res.ResultID,
		ExpectedVersion: intent.WorkflowVersion,
	})
	if outcome.Outcome != application.Accepted {
		log.Printf("runtime: submit result for workflow %s: %s %v", intent.WorkflowID, outcome.Outcome, outcome.Reasons)
	}
}

// computeBackoff implements first-slice plan decision #4: exponential
// backoff from RetryPolicy.BackoffBase, capped at BackoffMax, full jitter.
func computeBackoff(attemptNumber int, rp capability.RetryPolicy) time.Duration {
	backoff := rp.BackoffBase << uint(attemptNumber-1) // 1st retry: base, 2nd: base*2, ...
	if backoff > rp.BackoffMax || backoff <= 0 {
		backoff = rp.BackoffMax
	}
	if !rp.FullJitter {
		return backoff
	}
	return time.Duration(rand.Int63n(int64(backoff) + 1))
}
