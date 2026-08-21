package fallback

import (
	"context"
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
)

type retryableErr struct{ retryable bool }

func (e *retryableErr) Error() string   { return "fake provider error" }
func (e *retryableErr) Retryable() bool { return e.retryable }

type fakeAdapter struct {
	calls  int
	result ports.IntelligenceResult
	err    error
}

func (f *fakeAdapter) Generate(context.Context, ports.IntelligenceRequest) (ports.IntelligenceResult, error) {
	f.calls++
	return f.result, f.err
}

func TestGenerate_PrimarySucceeds_SecondaryNeverCalled(t *testing.T) {
	primary := &fakeAdapter{result: ports.IntelligenceResult{Text: "from primary", Provider: "primary"}}
	secondary := &fakeAdapter{result: ports.IntelligenceResult{Text: "from secondary", Provider: "secondary"}}
	a := New([]Provider{{Name: "primary", Adapter: primary}, {Name: "secondary", Adapter: secondary}}, time.Minute)

	res, err := a.Generate(context.Background(), ports.IntelligenceRequest{Prompt: "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Provider != "primary" {
		t.Errorf("provider = %q, want primary", res.Provider)
	}
	if secondary.calls != 0 {
		t.Errorf("secondary called %d times, want 0", secondary.calls)
	}
}

func TestGenerate_PrimaryRetryable_FallsBackToSecondary(t *testing.T) {
	primary := &fakeAdapter{err: &retryableErr{retryable: true}}
	secondary := &fakeAdapter{result: ports.IntelligenceResult{Text: "from secondary", Provider: "secondary"}}
	a := New([]Provider{{Name: "primary", Adapter: primary}, {Name: "secondary", Adapter: secondary}}, time.Minute)

	res, err := a.Generate(context.Background(), ports.IntelligenceRequest{Prompt: "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Provider != "secondary" {
		t.Errorf("provider = %q, want secondary", res.Provider)
	}
	if primary.calls != 1 {
		t.Errorf("primary called %d times, want 1", primary.calls)
	}
}

func TestGenerate_PrimaryTerminal_StopsImmediately(t *testing.T) {
	wantErr := &retryableErr{retryable: false}
	primary := &fakeAdapter{err: wantErr}
	secondary := &fakeAdapter{result: ports.IntelligenceResult{Text: "from secondary"}}
	a := New([]Provider{{Name: "primary", Adapter: primary}, {Name: "secondary", Adapter: secondary}}, time.Minute)

	_, err := a.Generate(context.Background(), ports.IntelligenceRequest{Prompt: "hi"})
	if err != error(wantErr) {
		t.Fatalf("error = %v, want the terminal error returned unchanged", err)
	}
	if secondary.calls != 0 {
		t.Errorf("secondary called %d times, want 0 (terminal error must not trigger fallback)", secondary.calls)
	}
}

func TestGenerate_AllProvidersRetryable_ReturnsLastError(t *testing.T) {
	primary := &fakeAdapter{err: &retryableErr{retryable: true}}
	secondary := &fakeAdapter{err: &retryableErr{retryable: true}}
	a := New([]Provider{{Name: "primary", Adapter: primary}, {Name: "secondary", Adapter: secondary}}, time.Minute)

	_, err := a.Generate(context.Background(), ports.IntelligenceRequest{Prompt: "hi"})
	if err == nil || !ports.IsRetryable(err) {
		t.Fatalf("error = %v, want a retryable error so Runtime's own backoff applies", err)
	}
	if primary.calls != 1 || secondary.calls != 1 {
		t.Errorf("calls = primary:%d secondary:%d, want 1 each", primary.calls, secondary.calls)
	}
}

func TestGenerate_CooldownSkipsRecentlyFailedProvider(t *testing.T) {
	primary := &fakeAdapter{err: &retryableErr{retryable: true}}
	secondary := &fakeAdapter{result: ports.IntelligenceResult{Provider: "secondary"}}
	a := New([]Provider{{Name: "primary", Adapter: primary}, {Name: "secondary", Adapter: secondary}}, time.Minute)

	clock := time.Now()
	a.now = func() time.Time { return clock }

	// First call: primary fails (retryable), falls back to secondary, primary enters cooldown.
	if _, err := a.Generate(context.Background(), ports.IntelligenceRequest{}); err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}
	if primary.calls != 1 {
		t.Fatalf("primary calls after first dispatch = %d, want 1", primary.calls)
	}

	// Second call, still within the cooldown window: primary must not be retried.
	if _, err := a.Generate(context.Background(), ports.IntelligenceRequest{}); err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}
	if primary.calls != 1 {
		t.Errorf("primary calls after second dispatch (still in cooldown) = %d, want still 1", primary.calls)
	}
	if secondary.calls != 2 {
		t.Errorf("secondary calls = %d, want 2", secondary.calls)
	}

	// Advance past the cooldown: primary should be tried again (it's first priority).
	clock = clock.Add(2 * time.Minute)
	if _, err := a.Generate(context.Background(), ports.IntelligenceRequest{}); err != nil {
		t.Fatalf("third call: unexpected error: %v", err)
	}
	if primary.calls != 2 {
		t.Errorf("primary calls after cooldown expired = %d, want 2 (should be retried)", primary.calls)
	}
}

func TestGenerate_NoProvidersConfigured_ReturnsError(t *testing.T) {
	a := New(nil, time.Minute)
	_, err := a.Generate(context.Background(), ports.IntelligenceRequest{})
	if err == nil {
		t.Fatal("expected an error with zero providers configured")
	}
}
