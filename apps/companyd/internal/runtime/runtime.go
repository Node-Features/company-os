package runtime

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/capability"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/execution"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/policy"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/fixtures"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

const workerID = "companyd-runtime"

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

	wg sync.WaitGroup
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

// Start runs the poll-and-wake loop until ctx is cancelled.
func (r *Runtime) Start(ctx context.Context) {
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

// Sweep claims due intents and dispatches each in its own goroutine.
func (r *Runtime) Sweep(ctx context.Context) {
	claims, err := r.Exec.ClaimDueIntents(ctx, fixtures.OrganizationID, 10, r.LeaseDuration, workerID)
	if err != nil {
		log.Printf("runtime: claim due intents: %v", err)
		return
	}
	for _, c := range claims {
		r.notifyChanged(ctx, c.Intent.WorkflowID)
		r.wg.Add(1)
		go func(c execution.ClaimedExecution) {
			defer r.wg.Done()
			r.execute(ctx, c.Attempt, c.Intent)
		}(c)
	}
}

func (r *Runtime) execute(ctx context.Context, attempt execution.ExecutionAttempt, intent workflow.ExecutionIntent) {
	decision, err := r.App.AuthorizeDispatch(ctx, intent)
	if err != nil {
		log.Printf("runtime: authorize dispatch: %v", err)
		return
	}
	if decision.Outcome != policy.DecisionAllow {
		// REQUIRE_APPROVAL/DENY/stale-version blocks dispatch
		// (application.md#dispatch-time-governance). This slice's
		// always-ALLOW policy makes this a rare race (a stale version),
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
			StartedAt:            attempt.CreatedAt,
			ObservedAt:           now,
			ReportedAt:           now,
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
