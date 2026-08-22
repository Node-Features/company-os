// Package agent provides a minimal in-process runtime for running Agent
// implementations as supervised goroutines. It is deliberately not the
// governed Agent described by docs/domain/agent.md — that record binds an
// AgentPrincipal, AgentDefinition, and Authority, none of which have a Go
// implementation yet (ADR-0007 flags this gap explicitly, and no
// internal/domain/agent package exists). Runtime here is a generic
// execution primitive one layer below that: it knows how to start a
// registered Agent under a cancellable context and report
// STARTED/SUCCEEDED/FAILED status onto a concurrency.Bus, nothing more. It
// does not authorize, govern, assign Capabilities, or persist anything an
// Agent does.
package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/Node-Features/company-os/apps/companyd/internal/concurrency"
)

// Agent is the minimal contract Runtime depends on. Run must return once
// its work is done or ctx is cancelled — Runtime does not kill a goroutine
// that ignores ctx.Done(), it can only wait for it (see Wait), so a
// well-behaved Agent is responsible for checking ctx itself.
type Agent interface {
	// Name identifies this Agent in status events and registration errors.
	// It must be unique among the Agents Registered on one Runtime.
	Name() string

	// Run performs the Agent's work. Returning a non-nil error reports
	// StatusFailed; returning nil reports StatusSucceeded. Run must respect
	// ctx cancellation itself — Runtime has no mechanism to forcibly stop a
	// goroutine that doesn't.
	Run(ctx context.Context) error
}

// Status is the lifecycle stage Runtime reports for one Agent's Run call.
type Status string

const (
	StatusStarted   Status = "STARTED"
	StatusSucceeded Status = "SUCCEEDED"
	StatusFailed    Status = "FAILED"
)

// StatusTopic is the concurrency.Bus topic Runtime publishes every
// StatusEvent to. Subscribe with bus.Subscribe(StatusTopic) before calling
// Start — concurrency.Bus does not replay events published before a
// subscriber joins.
const StatusTopic = "agent.status"

// StatusEvent is the Payload carried by every concurrency.Event Runtime
// publishes to StatusTopic.
type StatusEvent struct {
	AgentName string
	Status    Status
	// Err is set only when Status is StatusFailed — the error Run returned.
	Err error
}

// Runtime registers Agents and runs each as its own goroutine under a
// shared context.Context, publishing a StatusEvent to bus before and after
// each Run call. Like concurrency.Bus itself, status reporting is
// best-effort and in-memory: a subscriber that isn't listening when a
// status is published simply never sees it — see concurrency.Bus's doc
// comment for why that's an acceptable tradeoff for a live in-process hint,
// not a durable record.
type Runtime struct {
	bus *concurrency.Bus

	mu     sync.Mutex
	agents map[string]Agent

	wg sync.WaitGroup
}

// NewRuntime returns a Runtime that publishes status to bus. bus must not
// be nil.
func NewRuntime(bus *concurrency.Bus) *Runtime {
	return &Runtime{bus: bus, agents: make(map[string]Agent)}
}

// Register adds a to the set of Agents Start will run. It returns an error
// if an Agent with the same Name is already registered — Runtime treats
// Name as a unique key, not just a display label. Register must be called
// before Start; registering after Start has no effect on the in-flight run
// (Start snapshots the registered set once, at call time — see Start).
func (r *Runtime) Register(a Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.agents[a.Name()]; exists {
		return fmt.Errorf("agent: %q already registered", a.Name())
	}
	r.agents[a.Name()] = a
	return nil
}

// Start launches every currently-registered Agent in its own goroutine and
// returns immediately without waiting for any of them to finish — call
// Wait to block until they have. Each Agent receives the same ctx; when ctx
// is cancelled, Runtime itself does nothing further about it — it is up to
// each Agent's Run to notice ctx.Done() and return.
//
// Start reads the registered set once, at call time. Agents registered
// after Start returns are not launched by that call; a second Start call
// (after Wait) would launch them.
func (r *Runtime) Start(ctx context.Context) {
	r.mu.Lock()
	agents := make([]Agent, 0, len(r.agents))
	for _, a := range r.agents {
		agents = append(agents, a)
	}
	r.mu.Unlock()

	for _, a := range agents {
		r.wg.Add(1)
		go func(a Agent) {
			defer r.wg.Done()
			r.run(ctx, a)
		}(a)
	}
}

// Wait blocks until every Agent launched by the most recent Start call has
// returned. Like internal/runtime.Runtime.Wait, it does not wait for a
// Start call that begins after Wait is called.
func (r *Runtime) Wait() {
	r.wg.Wait()
}

func (r *Runtime) run(ctx context.Context, a Agent) {
	r.publish(StatusEvent{AgentName: a.Name(), Status: StatusStarted})
	if err := a.Run(ctx); err != nil {
		r.publish(StatusEvent{AgentName: a.Name(), Status: StatusFailed, Err: err})
		return
	}
	r.publish(StatusEvent{AgentName: a.Name(), Status: StatusSucceeded})
}

func (r *Runtime) publish(evt StatusEvent) {
	r.bus.Publish(concurrency.Event{Topic: StatusTopic, Payload: evt})
}
