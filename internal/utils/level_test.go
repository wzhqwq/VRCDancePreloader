package utils

import (
	"sync"
	"testing"
	"time"
)

func pendingWakeUps(level *Level[int]) int {
	woken := 0

drain:
	for {
		select {
		case <-level.Wakes():
			woken++
		default:
			break drain
		}
	}

	return woken
}

// TestLevelStoresBeforeWaking pins rule 1 of Level: a reader that reacts to a
// wake-up reads the value that caused it.
func TestLevelStoresBeforeWaking(t *testing.T) {
	level := NewLevel(0)

	level.Store(7)

	select {
	case <-level.Wakes():
	default:
		t.Fatal("Store did not wake the readers")
	}

	if got := level.Current(); got != 7 {
		t.Fatalf("Current() = %d after the wake-up, want 7", got)
	}
}

// TestLevelCoalescesWakeUps pins rule 2: one pending wake-up is enough, because
// the reader reads the value instead of receiving it.
func TestLevelCoalescesWakeUps(t *testing.T) {
	level := NewLevel(0)

	level.Store(1)
	level.Store(2)
	level.Store(3)

	if woken := pendingWakeUps(level); woken != 1 {
		t.Fatalf("got %d pending wake-ups after three stores, want 1", woken)
	}
	if got := level.Current(); got != 3 {
		t.Fatalf("Current() = %d, want the latest value 3", got)
	}

	level.Store(4)
	if woken := pendingWakeUps(level); woken != 1 {
		t.Fatalf("got %d wake-ups after the channel was drained, want 1", woken)
	}
}

// TestLevelWakeKeepsTheValue covers the "an external condition changed, the value
// did not" case (a replaced HTTP client).
func TestLevelWakeKeepsTheValue(t *testing.T) {
	level := NewLevel(5)

	level.Wake()

	if woken := pendingWakeUps(level); woken != 1 {
		t.Fatalf("got %d pending wake-ups after Wake(), want 1", woken)
	}
	if got := level.Current(); got != 5 {
		t.Fatalf("Wake() changed the value to %d, want it to stay 5", got)
	}
}

// TestLevelConcurrentReadersSeeTheLatestValue is the property the level is there
// for: whatever the interleaving, a reader that has been woken reads a value that
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

	go func() {
		defer close(done)

		wake := level.Wakes()
		for {
			select {
			case <-wake:
				mu.Lock()
				seen = append(seen, level.Current())
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
		t.Fatal("the reader was never woken")
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
