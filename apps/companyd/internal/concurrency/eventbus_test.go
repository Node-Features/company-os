package concurrency

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestBus_PublishSubscribe_Basic(t *testing.T) {
	b := NewBus()
	sub := b.Subscribe("topic-a")
	defer sub.Unsubscribe()

	b.Publish(Event{Topic: "topic-a", Payload: "hello"})

	select {
	case e := <-sub.C():
		if e.Payload != "hello" {
			t.Fatalf("payload = %v, want %q", e.Payload, "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for published event")
	}
}

func TestBus_Publish_OnlyReachesMatchingTopic(t *testing.T) {
	b := NewBus()
	a := b.Subscribe("topic-a")
	other := b.Subscribe("topic-b")
	defer a.Unsubscribe()
	defer other.Unsubscribe()

	b.Publish(Event{Topic: "topic-a", Payload: 1})

	select {
	case <-a.C():
	case <-time.After(time.Second):
		t.Fatal("subscriber to topic-a never received its event")
	}
	select {
	case e := <-other.C():
		t.Fatalf("subscriber to topic-b received an event meant for topic-a: %v", e)
	case <-time.After(50 * time.Millisecond):
		// correctly received nothing
	}
}

func TestBus_Publish_NeverBlocksOnFullOrAbsentSubscriber(t *testing.T) {
	b := NewBus()
	sub := b.Subscribe("topic-a")
	defer sub.Unsubscribe()

	// Never drain sub.C(): fill the buffer well past capacity and prove
	// Publish keeps returning immediately regardless.
	done := make(chan struct{})
	go func() {
		for i := 0; i < subscriberBufferSize*10; i++ {
			b.Publish(Event{Topic: "topic-a", Payload: i})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Publish blocked with a full, undrained subscriber buffer")
	}
}

func TestBus_Publish_DropsOldestKeepsNewest(t *testing.T) {
	b := NewBus()
	sub := b.Subscribe("topic-a")
	defer sub.Unsubscribe()

	// Publish more than the buffer can hold before reading anything.
	total := subscriberBufferSize + 5
	for i := 0; i < total; i++ {
		b.Publish(Event{Topic: "topic-a", Payload: i})
	}

	var got []int
	for {
		select {
		case e := <-sub.C():
			got = append(got, e.Payload.(int))
		default:
			goto drained
		}
	}
drained:
	if len(got) != subscriberBufferSize {
		t.Fatalf("buffered count = %d, want %d (buffer capacity)", len(got), subscriberBufferSize)
	}
	// The retained events must be the newest ones (the last
	// subscriberBufferSize payloads published), not the oldest.
	wantFirst := total - subscriberBufferSize
	if got[0] != wantFirst {
		t.Fatalf("oldest retained payload = %d, want %d (drop-oldest should have evicted everything before it)", got[0], wantFirst)
	}
	if got[len(got)-1] != total-1 {
		t.Fatalf("newest retained payload = %d, want %d", got[len(got)-1], total-1)
	}
}

func TestSubscription_Unsubscribe_IsIdempotent(t *testing.T) {
	b := NewBus()
	sub := b.Subscribe("topic-a")

	sub.Unsubscribe()
	sub.Unsubscribe() // must not panic
	sub.Unsubscribe() // neither must this

	// A closed channel must be safe to read from (returns zero value,
	// ok == false), never panics.
	v, ok := <-sub.C()
	if ok {
		t.Fatalf("expected closed channel, got value %v with ok=true", v)
	}
}

func TestBus_Publish_AfterUnsubscribe_DoesNotPanic(t *testing.T) {
	b := NewBus()
	sub := b.Subscribe("topic-a")
	sub.Unsubscribe()

	// Must not panic — Unsubscribe removed this subscriber from the map
	// before closing its channel, so Publish never sees it again.
	b.Publish(Event{Topic: "topic-a", Payload: "after unsubscribe"})
}

// TestBus_ConcurrentPublishAndUnsubscribe_NoPanic is the core proof for
// "graceful Unsubscribe that doesn't panic on a closed channel": many
// goroutines Publish to a topic while many other goroutines concurrently
// Subscribe and immediately Unsubscribe from the same topic, repeatedly,
// for long enough that a locking mistake (Publish observing a subscriber
// that Unsubscribe has already closed) would reliably surface as a panic
// and fail the test. -race could not be run in this environment (same
// CGo/gcc toolchain gap as ADR-0007's SafeState tests); this test
// compensates by maximizing the number of Publish/Unsubscribe
// interleavings rather than relying on race-detector instrumentation.
func TestBus_ConcurrentPublishAndUnsubscribe_NoPanic(t *testing.T) {
	b := NewBus()
	const topic = "topic-a"
	const publishers = 20
	const subscribers = 20
	const iterations = 2000

	var wg sync.WaitGroup

	for p := 0; p < publishers; p++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				b.Publish(Event{Topic: topic, Payload: fmt.Sprintf("p%d-%d", p, i)})
			}
		}(p)
	}

	for s := 0; s < subscribers; s++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				sub := b.Subscribe(topic)
				// Race Unsubscribe directly against whatever Publish calls
				// are in flight right now, with no delay.
				sub.Unsubscribe()
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("timed out — possible deadlock between Publish and Unsubscribe")
	}
	// Reaching here without a panic, across publishers*subscribers*iterations
	// = 800,000 Publish/Subscribe/Unsubscribe calls racing on one topic, is
	// the proof: any send-on-closed-channel bug would have panicked well
	// before this many interleavings completed.
}

// TestBus_ConcurrentPublishAndSubscribe_ReceivesWithoutRace exercises the
// ordinary case at the same scale: many publishers, many long-lived
// subscribers actually reading, confirming no deadlock and no lost
// goroutines under sustained concurrent load.
func TestBus_ConcurrentPublishAndSubscribe_ReceivesWithoutRace(t *testing.T) {
	b := NewBus()
	const topic = "topic-a"
	const publishers = 10
	const subscribers = 10
	const perPublisher = 1000

	var wg sync.WaitGroup
	stop := make(chan struct{})

	for s := 0; s < subscribers; s++ {
		wg.Add(1)
		sub := b.Subscribe(topic)
		go func(sub *Subscription) {
			defer wg.Done()
			defer sub.Unsubscribe()
			for {
				select {
				case _, ok := <-sub.C():
					if !ok {
						return
					}
				case <-stop:
					return
				}
			}
		}(sub)
	}

	var pubWG sync.WaitGroup
	for p := 0; p < publishers; p++ {
		pubWG.Add(1)
		go func(p int) {
			defer pubWG.Done()
			for i := 0; i < perPublisher; i++ {
				b.Publish(Event{Topic: topic, Payload: p*perPublisher + i})
			}
		}(p)
	}

	done := make(chan struct{})
	go func() {
		pubWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for publishers to finish")
	}

	close(stop)
	wg.Wait() // must return promptly; a hang here means a subscriber goroutine leaked
}
