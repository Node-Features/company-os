package concurrency

import "sync"

// Event is a minimal, generic publish/subscribe message. Deliberately not
// internal/domain/event.DomainEvent: Bus is not the CompanyOS event
// mechanism (see below), so it has no OrganizationID, CausationID, or any
// of the fields that make a DomainEvent an organizational fact rather than
// an in-memory notification.
type Event struct {
	Topic   string
	Payload any
}

// subscriberBufferSize bounds each subscriber's queue. Chosen to match
// internal/application.Application.Notify's existing buffered-channel size
// (make(chan uuid.UUID, 16) in cmd/companyd/main.go) for consistency with
// the one other bounded in-process notification channel in this codebase,
// not because 16 is derived from any load analysis.
const subscriberBufferSize = 16

// Bus is an in-memory, topic-based, concurrency-safe publish/subscribe
// primitive. Subscribe and Publish may be called from any number of
// goroutines concurrently without external synchronization.
//
// This is explicitly not CompanyOS's event mechanism. docs/architecture/
// events.md requires publication to begin "only from a persisted Event,"
// after commit, through an outbox-equivalent mechanism — internal/ports.
// Publisher (added for ADR-0006) is that real port; nothing implements it
// yet, and this Bus does not either. docs/references/feature-provenance.md
// is explicit about why: "reject... in-process bus delivery... as
// durable organizational records." Bus has no notion of durability,
// replay, or organizational identity — a subscriber that isn't listening
// when Publish is called simply never sees that event, permanently. Use it
// for the same kind of thing internal/application.Application.Notify
// already is: a live, best-effort, in-process hint — never for anything
// that must survive a crash or be observed exactly once.
type Bus struct {
	mu     sync.Mutex
	subs   map[string]map[int]chan Event
	nextID int
}

// NewBus returns an empty Bus, ready to use.
func NewBus() *Bus {
	return &Bus{subs: make(map[string]map[int]chan Event)}
}

// Subscription is a live registration returned by Subscribe. Read from
// C() to receive events; call Unsubscribe when done. A Subscription must
// not be copied after first use (it embeds no lock itself — synchronization
// lives on the Bus that created it).
type Subscription struct {
	bus   *Bus
	topic string
	id    int
	ch    chan Event
}

// C returns the channel this subscription receives on. It is closed when
// Unsubscribe is called — a receiver should use the two-value form
// (v, ok := <-sub.C()) to detect that, since reading from a closed channel
// is always safe in Go and never panics; only sending to (or closing) an
// already-closed channel does, which is exactly what this type's locking
// is designed to prevent Publish from ever doing.
func (s *Subscription) C() <-chan Event { return s.ch }

// Subscribe registers interest in topic and returns a Subscription whose
// channel receives every Event subsequently Published to that topic, from
// the moment Subscribe returns onward — there is no replay of anything
// published before this call.
func (b *Bus) Subscribe(topic string) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subs[topic] == nil {
		b.subs[topic] = make(map[int]chan Event)
	}
	id := b.nextID
	b.nextID++
	ch := make(chan Event, subscriberBufferSize)
	b.subs[topic][id] = ch
	return &Subscription{bus: b, topic: topic, id: id, ch: ch}
}

// Unsubscribe removes this subscription and closes its channel. It is safe
// to call from any goroutine, safe to call concurrently with Publish (see
// the package-internal correctness argument below), and safe to call more
// than once — the second and later calls are no-ops, not panics.
//
// Why this can never panic on a closed channel, by construction rather than
// by recover(): Publish holds b.mu for the entire time it is sending to
// subscriber channels for a topic. Unsubscribe holds the same b.mu for the
// entire time it removes a subscriber from the map and closes its channel.
// Because both critical sections are guarded by the identical mutex, it is
// impossible for Publish to be sending to (or about to send to) a channel
// that Unsubscribe has already closed, or for Unsubscribe to close a
// channel while Publish is mid-send to it — one of the two always
// completes first, and Unsubscribe's map deletion happens inside the same
// locked section as the close, so a Publish that acquires the lock after
// an Unsubscribe will not find that channel in the map at all. Double
// Unsubscribe is handled the same way: the second call acquires the lock,
// finds the ID already absent from the map, and returns before reaching
// close.
func (s *Subscription) Unsubscribe() {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()

	byID := s.bus.subs[s.topic]
	if byID == nil {
		return
	}
	if _, ok := byID[s.id]; !ok {
		return
	}
	delete(byID, s.id)
	if len(byID) == 0 {
		delete(s.bus.subs, s.topic)
	}
	close(s.ch)
}

// Publish delivers event to every current subscriber of event.Topic and
// never blocks, regardless of how slow or absent a subscriber's reader is.
//
// Buffer-full policy: drop-oldest, not block, and not drop-newest.
// internal/application.Application.notify — the one other bounded
// in-process notification channel in this codebase — already answers this
// exact question, but with drop-newest: "select { case a.Notify <-
// intentID: default: }". That is correct there because every queued value
// is an interchangeable wake-up hint — Runtime's poll sweep re-checks all
// due work regardless of which specific ID triggered it, so discarding a
// redundant new hint loses nothing. Bus does not have that property: each
// Event carries a distinct Payload a subscriber actually needs, so
// dropping the new one while keeping arbitrarily stale queued ones is
// backwards — a subscriber that eventually catches up should see the
// freshest state, not ancient history. Blocking was rejected because a
// single slow or stuck subscriber must never be able to stall every other
// subscriber's delivery, or the publisher itself, which would turn a
// notification primitive into an accidental synchronization point between
// unrelated goroutines.
func (b *Bus) Publish(event Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, ch := range b.subs[event.Topic] {
		select {
		case ch <- event:
			continue
		default:
		}
		// Buffer full: drop the oldest queued event, then enqueue the new
		// one. Both inner sends/receives are non-blocking (select+default)
		// even though, given Publish is the only sender and holds b.mu for
		// its whole body, a concurrent receiver can only ever free up room,
		// never remove it — the defensive default branches mean Publish
		// still cannot block even if that reasoning is ever wrong.
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- event:
		default:
		}
	}
}
