package interactive

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// stubAvailability is an AvailabilitySource whose level the test flips. Its
// wake-up channel has room for exactly one pending wake-up, like the real
// producers.
type stubAvailability struct {
	mu        sync.Mutex
	available bool
	wake      chan struct{}
}

func newStubAvailability() *stubAvailability {
	return &stubAvailability{wake: make(chan struct{}, 1)}
}

func (s *stubAvailability) Available() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.available
}

func (s *stubAvailability) Wakes() <-chan struct{} {
	return s.wake
}

func (s *stubAvailability) set(available bool) {
	s.mu.Lock()
	s.available = available
	s.mu.Unlock()

	if !available {
		return
	}

	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func waitFor(t *testing.T, what string, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", what)
}

// newFailingOnceManager builds a manager whose first fetch fails and whose
// second one succeeds, plus the handle that keeps the entry referenced. With the
// default retry policy (MaxRetries == 0) the first failure is terminal, so the
// only thing that can bring the entry back is an availability wake-up.
func newFailingOnceManager(t *testing.T) (*RemoteManager[int], *RemoteHandle[int], *atomic.Int64) {
	t.Helper()

	var calls atomic.Int64

	manager := NewRemoteManager[int](func(string, context.Context) (int, error) {
		if calls.Add(1) == 1 {
			return 0, errors.New("not available yet")
		}

		return 42, nil
	}, nil, 8, 1)

	handle := manager.Acquire("entry")
	t.Cleanup(func() {
		handle.Release()
		manager.Close()
	})

	return manager, handle, &calls
}

// TestAvailabilityWakeUpsRetryAFailedEntry is the review/09 §2.5 scenario: an
// entry that failed once is terminal, and the wake-up is what makes the manager
// look again.
func TestAvailabilityWakeUpsRetryAFailedEntry(t *testing.T) {
	manager, handle, calls := newFailingOnceManager(t)

	availability := newStubAvailability()
	manager.BindAvailability(availability)

	waitFor(t, "the first fetch to fail", func() bool {
		return handle.Snapshot().Status.Phase == RemoteError
	})

	availability.set(true)

	waitFor(t, "the entry to become ready", func() bool {
		return handle.Snapshot().Status.Phase == RemoteReady
	})

	if snapshot := handle.Snapshot(); snapshot.Data != 42 {
		t.Fatalf("got data %v, want 42", snapshot.Data)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("the getter ran %d times, want 2", got)
	}
}

// TestAvailabilityIsReadAsALevelAtBindTime covers the other half of the same
// hole: a level that is already true when the loop starts never sends a wake-up
// the loop could have subscribed to, so it has to be read once up front.
func TestAvailabilityIsReadAsALevelAtBindTime(t *testing.T) {
	manager, handle, calls := newFailingOnceManager(t)

	waitFor(t, "the first fetch to fail", func() bool {
		return handle.Snapshot().Status.Phase == RemoteError
	})

	// The level turns true before anything is bound to it: no wake-up is left
	// pending for the loop, only the level itself.
	availability := newStubAvailability()
	availability.set(true)
	<-availability.Wakes()

	manager.BindAvailability(availability)

	waitFor(t, "the entry to become ready", func() bool {
		return handle.Snapshot().Status.Phase == RemoteReady
	})

	if got := calls.Load(); got != 2 {
		t.Fatalf("the getter ran %d times, want 2", got)
	}
}

// TestAvailabilityWakeUpsAreIgnoredWhileUnavailable pins that a wake-up with a
// false level does not restart entries: the level, not the signal, is the
// decision.
func TestAvailabilityWakeUpsAreIgnoredWhileUnavailable(t *testing.T) {
	manager, handle, calls := newFailingOnceManager(t)

	availability := newStubAvailability()
	manager.BindAvailability(availability)

	waitFor(t, "the first fetch to fail", func() bool {
		return handle.Snapshot().Status.Phase == RemoteError
	})

	// A wake-up while the level is false: whatever raised it, the manager must
	// not retry.
	availability.set(false)
	select {
	case availability.wake <- struct{}{}:
	default:
	}

	time.Sleep(50 * time.Millisecond)

	if phase := handle.Snapshot().Status.Phase; phase != RemoteError {
		t.Fatalf("the entry moved to %v, want it to stay in RemoteError", phase)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("the getter ran %d times, want 1", got)
	}
}
