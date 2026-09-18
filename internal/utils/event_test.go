package utils

import (
	"sync"
	"testing"
	"time"
)

// A3 — "this event does not match my filter" is not "unsubscribe me". The sink
// returning false on no match used to make NotifySubscribers drop the whole pipe
// on the first event the filter rejected.
func TestMappedEventSinkSurvivesANonMatchingEvent(t *testing.T) {
	em := NewEventManager[int]()

	// forward only even numbers
	ch := PipeEvent(em, func(in int) (int, bool) {
		return in, in%2 == 0
	})
	defer ch.Close()

	em.NotifySubscribers(1)
	em.NotifySubscribers(2)

	select {
	case got := <-ch.Channel:
		if got != 2 {
			t.Fatalf("received %d, want 2", got)
		}
	case <-time.After(time.Second):
		t.Fatal("the pipe stopped receiving after a non matching event: it unsubscribed itself")
	}
}

// A3 — a plain subscriber is dropped exactly when it is closed, and notifying a
// closed one must not panic.
func TestNotifySubscribersDropsClosedSubscribers(t *testing.T) {
	em := NewEventManager[int]()
	ch := em.SubscribeEvent()
	ch.Close()

	em.NotifySubscribers(1)

	em.subMu.Lock()
	remaining := len(em.subscribers)
	em.subMu.Unlock()

	if remaining != 0 {
		t.Fatalf("%d subscribers left after Close, want none", remaining)
	}
}

// A4 — registering a pipe while events are being delivered used to write
// em.subscribers without the lock, which raced with NotifySubscribers' lo.Filter
// reassignment and could silently drop the new pipe.
//
// Run with -race for this to mean anything.
func TestPipeEventIsSafeToRegisterWhileNotifying(t *testing.T) {
	em := NewEventManager[int]()

	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-stop:
				return
			default:
				em.NotifySubscribers(1)
			}
		}
	}()

	subs := make([]*EventSubscriber[int], 0, 64)
	for i := 0; i < 64; i++ {
		subs = append(subs, PipeEvent(em, func(in int) (int, bool) {
			return in, true
		}))
	}

	close(stop)
	wg.Wait()

	// Every registered pipe has to have survived the concurrent notifications.
	em.subMu.Lock()
	registered := len(em.subscribers)
	em.subMu.Unlock()

	if registered != len(subs) {
		t.Fatalf("%d pipes registered, want %d: some were dropped or duplicated", registered, len(subs))
	}

	for _, s := range subs {
		s.Close()
	}
}
