package utils

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// The two primitives are compared where they differ: the cost of publishing a
// change, with the readers either draining (so nothing is dropped/coalesced) or
// absent (the burst case). Run with:
//
//	go test ./internal/utils/ -run '^$' -bench 'Notify|Level' -benchmem

func drainEventSubscribers(em *EventManager[int], count int) (stop func(), woken *atomic.Int64) {
	stopCh := make(chan struct{})
	woken = &atomic.Int64{}

	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		sub := em.SubscribeEvent()
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-sub.Channel:
					woken.Add(1)
				case <-stopCh:
					return
				}
			}
		}()
	}

	return func() {
		close(stopCh)
		wg.Wait()
	}, woken
}

func drainLevel(level *Level[int], count int) (stop func(), woken *atomic.Int64) {
	stopCh := make(chan struct{})
	woken = &atomic.Int64{}

	var wg sync.WaitGroup
	watchers := make([]*LevelWatcher[int], 0, count)
	for i := 0; i < count; i++ {
		watcher := level.Subscribe()
		watchers = append(watchers, watcher)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-watcher.Wakes():
					_ = watcher.Current()
					woken.Add(1)
				case <-stopCh:
					return
				}
			}
		}()
	}

	return func() {
		close(stopCh)
		wg.Wait()
		for _, watcher := range watchers {
			watcher.Close()
		}
	}, woken
}

func BenchmarkEventManagerNotify(b *testing.B) {
	for _, subscribers := range []int{0, 1, 8} {
		b.Run(fmt.Sprintf("subscribers=%d", subscribers), func(b *testing.B) {
			em := NewEventManager[int]()
			stop, woken := drainEventSubscribers(em, subscribers)
			defer stop()

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				em.NotifySubscribers(i)
			}
			b.StopTimer()

			b.ReportMetric(float64(woken.Load())/float64(b.N), "woken/op")
		})
	}
}

func BenchmarkLevelStore(b *testing.B) {
	for _, readers := range []int{0, 1, 8} {
		b.Run(fmt.Sprintf("readers=%d", readers), func(b *testing.B) {
			level := NewLevel(0)
			stop, woken := drainLevel(level, readers)
			defer stop()

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				level.Store(i)
			}
			b.StopTimer()

			b.ReportMetric(float64(woken.Load())/float64(b.N), "woken/op")
		})
	}
}

// BenchmarkLevelStoreCoalesced isolates the cost of waking a *parked* consumer
// from the cost of the store itself: the watchers exist but never read, so every
// wake-up after the first falls into the capacity-1 slot and is dropped.
func BenchmarkLevelStoreCoalesced(b *testing.B) {
	for _, watchers := range []int{1, 8} {
		b.Run(fmt.Sprintf("watchers=%d", watchers), func(b *testing.B) {
			level := NewLevel(0)
			for i := 0; i < watchers; i++ {
				watcher := level.Subscribe()
				defer watcher.Close()
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				level.Store(i)
			}
		})
	}
}

// BenchmarkLevelCurrent isolates the read side, which is what a consumer pays
// instead of receiving a payload.
func BenchmarkLevelCurrent(b *testing.B) {
	level := NewLevel(42)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if level.Current() != 42 {
			b.Fatal("read the wrong value")
		}
	}
}
