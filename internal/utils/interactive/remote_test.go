package interactive

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// --- delivery order (review/09 §8.5) -----------------------------------------

func newTestEntry[T any](initial T) *remoteEntry[T] {
	entry := &remoteEntry[T]{
		id: "test",
		em: utils.NewEventManager[RemoteSnapshot[T]](),
	}
	entry.data, entry.hasData = initial, true

	return entry
}

// bumpWithData models one write followed by the snapshot that announces it.
func bumpWithData[T any](entry *remoteEntry[T], data T) (RemoteSnapshot[T], uint64) {
	entry.mu.Lock()
	defer entry.mu.Unlock()

	entry.data, entry.hasData = data, true

	return entry.bumpSnapshotLocked()
}

func drainSnapshots[T any](ch *utils.EventSubscriber[RemoteSnapshot[T]]) []RemoteSnapshot[T] {
	var got []RemoteSnapshot[T]

drain:
	for {
		select {
		case snapshot := <-ch.Channel:
			got = append(got, snapshot)
		default:
			break drain
		}
	}

	return got
}

func snapshotData(t *testing.T, snapshots []RemoteSnapshot[int]) []int {
	t.Helper()

	data := make([]int, 0, len(snapshots))
	for _, snapshot := range snapshots {
		data = append(data, snapshot.Data)
	}

	return data
}

// TestPublishDropsSupersededSnapshots is the review/09 §8.5 case: the snapshot of
// an older state is announced after a newer one was already announced. It used to
// reach the subscriber and make it publish a value that had been replaced.
func TestPublishDropsSupersededSnapshots(t *testing.T) {
	entry := newTestEntry(0)
	ch := entry.em.SubscribeEvent()
	defer ch.Close()

	first, firstSeq := bumpWithData(entry, 1)
	second, secondSeq := bumpWithData(entry, 2)

	entry.publish(second, secondSeq)
	entry.publish(first, firstSeq)

	third, thirdSeq := bumpWithData(entry, 3)
	entry.publish(third, thirdSeq)

	got := snapshotData(t, drainSnapshots(ch))
	if len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Fatalf("delivered %v, want [2 3] (the stale 1 must be dropped)", got)
	}
}

func TestPublishKeepsInOrderDeliveries(t *testing.T) {
	entry := newTestEntry(0)
	ch := entry.em.SubscribeEvent()
	defer ch.Close()

	first, firstSeq := bumpWithData(entry, 1)
	entry.publish(first, firstSeq)
	second, secondSeq := bumpWithData(entry, 2)
	entry.publish(second, secondSeq)

	got := snapshotData(t, drainSnapshots(ch))
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("delivered %v, want [1 2]", got)
	}
}

// TestPublishDoesNotRepeatAVersion pins that re-announcing the same version is a
// no-op: the retry loop re-publishes its snapshot on the failure path, and a
// duplicate carries nothing new.
func TestPublishDoesNotRepeatAVersion(t *testing.T) {
	entry := newTestEntry(0)
	ch := entry.em.SubscribeEvent()
	defer ch.Close()

	snapshot, seq := bumpWithData(entry, 1)
	entry.publish(snapshot, seq)
	entry.publish(snapshot, seq)

	if got := snapshotData(t, drainSnapshots(ch)); len(got) != 1 || got[0] != 1 {
		t.Fatalf("delivered %v, want [1]", got)
	}
}

// TestPublishNeverRegressesUnderConcurrency checks the property the guard exists
// for: whatever the interleaving, a subscriber never sees a state older than one
// it has already been given.
//
// The counter is bumped inside the same critical section as the write, exactly
// like a real producer (which writes its state and bumps the version under the
// entry lock), so a delivery that is not strictly greater than the previous one
// can only mean the guard let an older state through.
func TestPublishNeverRegressesUnderConcurrency(t *testing.T) {
	entry := newTestEntry(int64(0))

	var counter int64
	bumpCounter := func() (RemoteSnapshot[int64], uint64) {
		entry.mu.Lock()
		defer entry.mu.Unlock()

		counter++
		entry.data, entry.hasData = counter, true

		return entry.bumpSnapshotLocked()
	}

	ch := entry.em.SubscribeEvent()
	defer ch.Close()

	var (
		mu       sync.Mutex
		observed []int64
		stop     = make(chan struct{})
		done     = make(chan struct{})
	)

	go func() {
		defer close(done)

		collect := func(snapshot RemoteSnapshot[int64]) {
			mu.Lock()
			observed = append(observed, snapshot.Data)
			mu.Unlock()
		}

		for {
			select {
			case snapshot := <-ch.Channel:
				collect(snapshot)
			case <-stop:
				for {
					select {
					case snapshot := <-ch.Channel:
						collect(snapshot)
					default:
						return
					}
				}
			}
		}
	}()

	var publishers sync.WaitGroup
	for publisher := 0; publisher < 4; publisher++ {
		publishers.Add(1)
		go func() {
			defer publishers.Done()

			for i := 0; i < 50; i++ {
				snapshot, seq := bumpCounter()
				entry.publish(snapshot, seq)
			}
		}()
	}

	publishers.Wait()
	close(stop)
	<-done

	mu.Lock()
	defer mu.Unlock()

	if len(observed) == 0 {
		t.Fatal("no snapshot was delivered at all")
	}

	for i := 1; i < len(observed); i++ {
		if observed[i] <= observed[i-1] {
			t.Fatalf("delivery %d went backwards: %v", i, observed[i-1:i+1])
		}
	}

	if last := observed[len(observed)-1]; last > entry.snapshot().Data {
		t.Fatalf("the last delivery (%d) is newer than the entry (%d)", last, entry.snapshot().Data)
	}
}

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

func (s *stubAvailability) Current() bool {
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
