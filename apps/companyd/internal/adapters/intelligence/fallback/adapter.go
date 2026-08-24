// Package fallback composes multiple internal/adapters/intelligence/*
// ProviderAdapters into one ProviderAdapter that dynamically picks which
// concrete provider serves a dispatch based on recent rate-limit/outage
// evidence — not the full model/provider Router docs/architecture/intelligence.md
// defines (Task Analyzer, Governance eligibility, Finance budget, M&E
// evidence, and a persisted RoutingDecision are all Phase 3+ work, tracked
// there, not reimplemented here). This is a narrower, purely operational
// concern: given a fixed, already-Governance-approved set of provider
// adapters for the one first-slice IntelligenceCapability, avoid dispatching
// to one that just failed with a retryable error, in favor of the next.
package fallback

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/observability"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
)

// Provider names one ProviderAdapter in priority order.
type Provider struct {
	Name    string
	Adapter ports.ProviderAdapter
}

type entry struct {
	Provider
	unhealthyUntil time.Time
}

// Adapter implements ports.ProviderAdapter by trying Providers in the
// order given to New, skipping any still inside its cooldown window from a
// previous retryable failure.
type Adapter struct {
	mu       sync.Mutex
	entries  []*entry
	cooldown time.Duration
	now      func() time.Time // overridable for tests
}

// New builds an Adapter. cooldown is how long a provider is skipped after
// a retryable (rate-limit/outage) failure — first-slice default is 60s,
// set by cmd/companyd/main.go. Order is priority order: providers[0] is
// tried first whenever it isn't in cooldown.
func New(providers []Provider, cooldown time.Duration) *Adapter {
	entries := make([]*entry, len(providers))
	for i, p := range providers {
		entries[i] = &entry{Provider: p}
	}
	return &Adapter{entries: entries, cooldown: cooldown, now: time.Now}
}

// Names returns the configured providers in priority order, for building a
// descriptive Runtime.ProviderName label.
func (a *Adapter) Names() []string {
	names := make([]string, len(a.entries))
	for i, e := range a.entries {
		names[i] = e.Name
	}
	return names
}

func (a *Adapter) String() string { return strings.Join(a.Names(), "+") }

// Generate tries each configured provider in priority order (skipping ones
// currently in cooldown), returning the first success. A terminal error
// (not rate-limit/outage) stops immediately — trying another provider
// can't fix a malformed request. A retryable error puts that provider in
// cooldown and moves to the next; if every provider is exhausted, the last
// retryable error is returned so Runtime's own cross-attempt backoff
// (ScheduleRetry) applies to the whole dispatch, not just one provider.
func (a *Adapter) Generate(ctx context.Context, req ports.IntelligenceRequest) (ports.IntelligenceResult, error) {
	order := a.orderedEntries()
	if len(order) == 0 {
		return ports.IntelligenceResult{}, fmt.Errorf("intelligence/fallback: no provider configured")
	}

	var lastErr error
	for _, e := range order {
		result, err := e.Adapter.Generate(ctx, req)
		if err == nil {
			a.markHealthy(e)
			return result, nil
		}
		lastErr = err

		if !ports.IsRetryable(err) {
			observability.Logger(ctx).Error("intelligence/fallback: provider failed, not trying further providers",
				"provider", e.Name, "failure_reason", observability.SafeProviderError(err, false))
			return ports.IntelligenceResult{}, err
		}

		observability.Logger(ctx).Warn("intelligence/fallback: provider failed, cooling down and trying next",
			"provider", e.Name, "cooldown", a.cooldown.String(), "failure_reason", observability.SafeProviderError(err, true))
		a.markUnhealthy(e)
	}
	return ports.IntelligenceResult{}, lastErr
}

// orderedEntries returns providers not currently in cooldown first (in
// configured priority order), then providers in cooldown (also in
// configured priority order) — so a dispatch is never refused just because
// every provider has recently failed once.
func (a *Adapter) orderedEntries() []*entry {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := a.now()
	var healthy, unhealthy []*entry
	for _, e := range a.entries {
		if e.unhealthyUntil.After(now) {
			unhealthy = append(unhealthy, e)
		} else {
			healthy = append(healthy, e)
		}
	}
	return append(healthy, unhealthy...)
}

func (a *Adapter) markHealthy(e *entry) {
	a.mu.Lock()
	defer a.mu.Unlock()
	e.unhealthyUntil = time.Time{}
}

func (a *Adapter) markUnhealthy(e *entry) {
	a.mu.Lock()
	defer a.mu.Unlock()
	e.unhealthyUntil = a.now().Add(a.cooldown)
}
