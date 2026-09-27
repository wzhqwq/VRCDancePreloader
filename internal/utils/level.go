package utils

import "sync"

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

// Level is the reusable producer side of LevelSource.
//
// Two rules make it work, and both are load-bearing:
//
//  1. the value is stored *before* the wake-up, so a reader that reacts to a
//     wake-up always sees the value that caused it;
//  2. wake-ups are coalesced (one pending is enough), because the reader reads
//     the value instead of receiving it — so dropping a wake-up loses nothing.
//
// A wake-up means "read it again", not "the value became true": storing a value
// that turns something off wakes the readers too, and they see the new value.
type Level[T any] struct {
	mu    sync.RWMutex
	value T
	wake  chan struct{}
}

func NewLevel[T any](initial T) *Level[T] {
	return &Level[T]{
		value: initial,
		wake:  make(chan struct{}, 1),
	}
}

// Current implements LevelSource.
func (l *Level[T]) Current() T {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.value
}

// Wakes implements LevelSource.
func (l *Level[T]) Wakes() <-chan struct{} {
	return l.wake
}

// Store publishes a new value and wakes the readers.
func (l *Level[T]) Store(value T) {
	l.mu.Lock()
	l.value = value
	l.mu.Unlock()

	l.Wake()
}

// Wake tells the readers to read Current again without changing it: for callers
// that know an external condition changed while the value itself did not (a
// replaced HTTP client, for example, which makes previously failed work worth
// another try).
func (l *Level[T]) Wake() {
	select {
	case l.wake <- struct{}{}:
	default:
		// A wake-up is already pending and the reader will read the value that is
		// set right now, so this one adds nothing.
	}
}
