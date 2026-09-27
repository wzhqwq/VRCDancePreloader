package utils

import (
	"sync"
	"sync/atomic"
)

// LevelSource is a value that can be read at any time, together with a wake-up
// that says "read it again".
//
// The wake-up deliberately carries no value: a reader re-reads Current when it is
// woken, so a dropped, duplicated or late wake-up cannot make it believe a state
// that no longer holds — which is the failure mode of an edge-triggered signal
// that carries the new value (review/09 §2.5 and §8.5).
//
// It is the counterpart of EventManager, not a replacement: EventManager
// delivers payloads and every event matters (deltas, one-shot notifications,
// protocol messages), while LevelSource is for *state* — "the configuration
// changed", "this resource is usable now", "this entry has a new snapshot".
type LevelSource[T any] interface {
	// Current returns the value as of now.
	Current() T
	// Wakes returns the wake-up channel: a receive means "Current may have
	// changed". The values it carries are never read.
	Wakes() <-chan struct{}
}

// Level is the producer side of a state: it holds a value and wakes its watchers
// when the value is stored.
//
// Two rules make it work, and both are load-bearing:
//
//  1. the value is stored *before* the wake-ups are sent, so a watcher that
//     reacts to a wake-up always sees the value that caused it;
//  2. wake-ups are coalesced (one pending per watcher is enough), because the
//     watcher reads the value instead of receiving it — so dropping one loses
//     nothing.
//
// A wake-up means "read it again", not "the value became true": storing a value
// that turns something off wakes the watchers too, and they see the new value.
//
// Every consumer gets its own watcher from Subscribe. Do not hand one watcher to
// two consumers: a wake-up is delivered to a single watcher, so the other
// consumer would keep sleeping (measured: with one shared wake slot only 1 of 8
// readers was woken per store).
type Level[T any] struct {
	mu       sync.RWMutex
	value    T
	watchers []*LevelWatcher[T]
}

func NewLevel[T any](initial T) *Level[T] {
	return &Level[T]{value: initial}
}

// Current returns the value as of now.
func (l *Level[T]) Current() T {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.value
}

// Store publishes a new value and wakes every watcher.
func (l *Level[T]) Store(value T) {
	l.mu.Lock()
	l.value = value
	l.wakeLocked()
	l.mu.Unlock()
}

// Wake tells the watchers to read Current again without changing it: for callers
// that know an external condition changed while the value itself did not (a
// replaced HTTP client, for example, which makes previously failed work worth
// another try).
func (l *Level[T]) Wake() {
	l.mu.RLock()
	l.wakeLocked()
	l.mu.RUnlock()
}

func (l *Level[T]) wakeLocked() {
	for _, watcher := range l.watchers {
		watcher.wakeUp()
	}
}

// Subscribe returns a watcher for one consumer. Close it when the consumer stops,
// so that it is no longer woken (the Level keeps its watchers forever otherwise).
func (l *Level[T]) Subscribe() *LevelWatcher[T] {
	watcher := &LevelWatcher[T]{
		level: l,
		wake:  make(chan struct{}, 1),
	}

	l.mu.Lock()
	l.watchers = append(l.watchers, watcher)
	l.mu.Unlock()

	return watcher
}

// LevelWatcher is one consumer's view of a Level: it is a LevelSource itself, and
// it is safe for use from a single consumer goroutine (the same discipline as an
// EventSubscriber).
type LevelWatcher[T any] struct {
	level *Level[T]
	wake  chan struct{}

	closed atomic.Bool
}

// Current implements LevelSource: the watcher reads through to the level, so it
// always sees the latest value, not the one that was current when it was woken.
func (w *LevelWatcher[T]) Current() T {
	return w.level.Current()
}

// Wakes implements LevelSource.
func (w *LevelWatcher[T]) Wakes() <-chan struct{} {
	return w.wake
}

// Close removes the watcher from its level. The wake-up channel is deliberately
// *not* closed: a wake-up that is already pending stays harmless, and nothing can
// ever send on a closed channel.
func (w *LevelWatcher[T]) Close() {
	if w.closed.CompareAndSwap(false, true) {
		w.level.unsubscribe(w)
	}
}

func (l *Level[T]) unsubscribe(watcher *LevelWatcher[T]) {
	l.mu.Lock()
	defer l.mu.Unlock()

	kept := l.watchers[:0]
	for _, candidate := range l.watchers {
		if candidate != watcher {
			kept = append(kept, candidate)
		}
	}
	l.watchers = kept
}

func (w *LevelWatcher[T]) wakeUp() {
	if w.closed.Load() {
		return
	}

	select {
	case w.wake <- struct{}{}:
	default:
		// A wake-up is already pending and this watcher will read the value that
		// is set right now, so this one adds nothing.
	}
}
