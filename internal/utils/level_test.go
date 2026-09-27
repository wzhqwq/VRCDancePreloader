package utils

import (
	"sync"
	"testing"
	"time"
)

func pendingWakeUps(watcher *LevelWatcher[int]) int {
	woken := 0

drain:
	for {
		select {
		case <-watcher.Wakes():
			woken++
		default:
			break drain
		}
	}

	return woken
}

// TestLevelStoresBeforeWaking pins rule 1 of Level: a watcher that reacts to a
// wake-up reads the value that caused it.
func TestLevelStoresBeforeWaking(t *testing.T) {
	level := NewLevel(0)
	watcher := level.Subscribe()
	defer watcher.Close()

	level.Store(7)

	select {
	case <-watcher.Wakes():
	default:
		t.Fatal("Store did not wake the watcher")
	}

	if got := watcher.Current(); got != 7 {
		t.Fatalf("Current() = %d after the wake-up, want 7", got)
	}
}

// TestLevelCoalescesWakeUps pins rule 2: one pending wake-up is enough, because
// the watcher reads the value instead of receiving it.
func TestLevelCoalescesWakeUps(t *testing.T) {
	level := NewLevel(0)
	watcher := level.Subscribe()
	defer watcher.Close()

	level.Store(1)
	level.Store(2)
	level.Store(3)

	if woken := pendingWakeUps(watcher); woken != 1 {
		t.Fatalf("got %d pending wake-ups after three stores, want 1", woken)
	}
	if got := watcher.Current(); got != 3 {
		t.Fatalf("Current() = %d, want the latest value 3", got)
	}

	level.Store(4)
	if woken := pendingWakeUps(watcher); woken != 1 {
		t.Fatalf("got %d wake-ups after the channel was drained, want 1", woken)
	}
}

// TestLevelWakeKeepsTheValue covers the "an external condition changed, the value
// did not" case (a replaced HTTP client).
func TestLevelWakeKeepsTheValue(t *testing.T) {
	level := NewLevel(5)
	watcher := level.Subscribe()
	defer watcher.Close()

	level.Wake()

	if woken := pendingWakeUps(watcher); woken != 1 {
		t.Fatalf("got %d pending wake-ups after Wake(), want 1", woken)
	}
	if got := watcher.Current(); got != 5 {
		t.Fatalf("Wake() changed the value to %d, want it to stay 5", got)
	}
}

// TestLevelWakesEveryWatcher is why watchers exist instead of one shared wake-up
// channel: each consumer has its own slot, so one store wakes all of them.
func TestLevelWakesEveryWatcher(t *testing.T) {
	level := NewLevel(0)

	watchers := make([]*LevelWatcher[int], 0, 4)
	for i := 0; i < 4; i++ {
		watcher := level.Subscribe()
		defer watcher.Close()
		watchers = append(watchers, watcher)
	}

	level.Store(9)

	for i, watcher := range watchers {
		if woken := pendingWakeUps(watcher); woken != 1 {
			t.Fatalf("watcher %d got %d wake-ups, want 1", i, woken)
		}
		if got := watcher.Current(); got != 9 {
			t.Fatalf("watcher %d read %d, want 9", i, got)
		}
	}
}

// TestLevelClosedWatcherStopsBeingWoken covers the lifecycle: a consumer that is
// gone must not be woken (nor block the producer) forever.
func TestLevelClosedWatcherStopsBeingWoken(t *testing.T) {
	level := NewLevel(0)
	watcher := level.Subscribe()
	other := level.Subscribe()
	defer other.Close()

	level.Store(1) // sanity: the watcher is live
	if woken := pendingWakeUps(watcher); woken != 1 {
		t.Fatalf("live watcher got %d wake-ups, want 1", woken)
	}

	watcher.Close()
	level.Store(2)

	if woken := pendingWakeUps(watcher); woken != 0 {
		t.Fatalf("closed watcher got %d wake-ups, want 0", woken)
	}
	// The other watcher is woken, but its wake-up is coalesced: Store(1) already
	// left one pending, so the slot holds one wake-up, not two.
	if woken := pendingWakeUps(other); woken != 1 {
		t.Fatalf("the other watcher got %d wake-ups, want 1 (coalesced)", woken)
	}
	if got := other.Current(); got != 2 {
		t.Fatalf("the other watcher read %d, want 2", got)
	}

	watcher.Close() // idempotent
}

// TestLevelConcurrentReadersSeeTheLatestValue is the property the level is there
// for: whatever the interleaving, a watcher that has been woken reads a value that
// is at least as new as the one that woke it. Values are produced by a counter
// incremented in the same critical section as the store, like a real producer.
func TestLevelConcurrentReadersSeeTheLatestValue(t *testing.T) {
	level := NewLevel(int64(0))

	var (
		value int64
		mu    sync.Mutex
		seen  []int64
		stop  = make(chan struct{})
		done  = make(chan struct{})
	)

	watcher := level.Subscribe()
	defer watcher.Close()

	go func() {
		defer close(done)

		for {
			select {
			case <-watcher.Wakes():
				mu.Lock()
				seen = append(seen, watcher.Current())
				mu.Unlock()
			case <-stop:
				return
			}
		}
	}()

	var producers sync.WaitGroup
	for producer := 0; producer < 4; producer++ {
		producers.Add(1)
		go func() {
			defer producers.Done()

			for i := 0; i < 50; i++ {
				// The counter and the store share one critical section, exactly
				// like a real producer (which writes its state and publishes it
				// while holding its own lock): without that, two producers can
				// store their values out of counter order and the level really
				// does move backwards.
				mu.Lock()
				value++
				level.Store(value)
				mu.Unlock()
			}
		}()
	}

	producers.Wait()
	time.Sleep(10 * time.Millisecond)
	close(stop)
	<-done

	mu.Lock()
	defer mu.Unlock()

	if len(seen) == 0 {
		t.Fatal("the watcher was never woken")
	}
	for i := 1; i < len(seen); i++ {
		if seen[i] < seen[i-1] {
			t.Fatalf("read %v went backwards", seen[i-1:i+1])
		}
	}
	if last := seen[len(seen)-1]; last > level.Current() {
		t.Fatalf("the last read (%d) is newer than the level (%d)", last, level.Current())
	}
}
