package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/agent"
	"github.com/Node-Features/company-os/apps/companyd/internal/agent/examples"
	"github.com/Node-Features/company-os/apps/companyd/internal/concurrency"
)

// TestRuntime_EchoAgent_PublishesCompletionEvent is the end-to-end proof
// this package exists to give: start a Runtime, register EchoAgent, run it
// under a real context, and assert the STARTED-then-SUCCEEDED StatusEvent
// pair actually arrives on the bus — not just that Register/Start/Wait
// return without error, which would prove nothing about the reporting path.
func TestRuntime_EchoAgent_PublishesCompletionEvent(t *testing.T) {
	bus := concurrency.NewBus()
	sub := bus.Subscribe(agent.StatusTopic)
	defer sub.Unsubscribe()

	rt := agent.NewRuntime(bus)
	echo := &examples.EchoAgent{AgentName: "echo-1", Input: "hello"}
	if err := rt.Register(echo); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rt.Start(ctx)
	rt.Wait()

	started := recvStatus(t, sub)
	if started.AgentName != "echo-1" || started.Status != agent.StatusStarted {
		t.Fatalf("first event = %+v, want STARTED for echo-1", started)
	}

	completed := recvStatus(t, sub)
	if completed.AgentName != "echo-1" || completed.Status != agent.StatusSucceeded {
		t.Fatalf("second event = %+v, want SUCCEEDED for echo-1", completed)
	}
	if completed.Err != nil {
		t.Fatalf("completed.Err = %v, want nil", completed.Err)
	}
}

// TestRuntime_CancelledContext_AgentReportsFailure proves Runtime respects
// context cancellation end to end: EchoAgent given an already-cancelled ctx
// returns ctx.Err() from Run, and Runtime reports StatusFailed with that
// error rather than StatusSucceeded — status reflects what the Agent
// actually did, not just that Run returned.
func TestRuntime_CancelledContext_AgentReportsFailure(t *testing.T) {
	bus := concurrency.NewBus()
	sub := bus.Subscribe(agent.StatusTopic)
	defer sub.Unsubscribe()

	rt := agent.NewRuntime(bus)
	echo := &examples.EchoAgent{AgentName: "echo-2", Input: "unreachable"}
	if err := rt.Register(echo); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before Start, so Run must observe it immediately

	rt.Start(ctx)
	rt.Wait()

	_ = recvStatus(t, sub) // STARTED

	failed := recvStatus(t, sub)
	if failed.Status != agent.StatusFailed {
		t.Fatalf("status = %s, want FAILED (ctx was already cancelled)", failed.Status)
	}
	if failed.Err == nil {
		t.Fatalf("Err = nil, want context.Canceled")
	}
}

// TestRuntime_Register_DuplicateName_Fails proves Name is enforced as a
// unique key, not merely a display label — a second Agent registered under
// a name already in use must be rejected rather than silently replacing the
// first.
func TestRuntime_Register_DuplicateName_Fails(t *testing.T) {
	bus := concurrency.NewBus()
	rt := agent.NewRuntime(bus)

	first := &examples.EchoAgent{AgentName: "dup", Input: "one"}
	second := &examples.EchoAgent{AgentName: "dup", Input: "two"}

	if err := rt.Register(first); err != nil {
		t.Fatalf("register first: %v", err)
	}
	if err := rt.Register(second); err == nil {
		t.Fatalf("register second: got nil error, want error for duplicate name %q", "dup")
	}
}

func recvStatus(t *testing.T, sub *concurrency.Subscription) agent.StatusEvent {
	t.Helper()
	select {
	case evt, ok := <-sub.C():
		if !ok {
			t.Fatalf("subscription channel closed before expected event")
		}
		status, ok := evt.Payload.(agent.StatusEvent)
		if !ok {
			t.Fatalf("payload type = %T, want agent.StatusEvent", evt.Payload)
		}
		return status
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for status event")
	}
	return agent.StatusEvent{}
}
