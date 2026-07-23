package utils

import (
	"fmt"
	"sync"
	"time"
)

var pypyScheduler = NewScheduler(5*time.Second, time.Minute, time.Minute*3)
var videoScheduler = NewScheduler(3*time.Second, time.Minute, time.Minute*3)
var thumbnailScheduler = NewScheduler(500*time.Millisecond, 10*time.Second, time.Minute)

func PyPyVideoScheduler() *Scheduler {
	return pypyScheduler
}
func SharedVideoScheduler() *Scheduler {
	return videoScheduler
}
func SharedThumbnailScheduler() *Scheduler {
	return thumbnailScheduler
}

func NewBasicScheduler() *Scheduler {
	return NewScheduler(3*time.Second, time.Minute, time.Minute*3)
}

type ThrottledError struct {
	RetryAfter time.Duration
}

func (e *ThrottledError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("server throttled, try after %v", e.RetryAfter)
	}
	return fmt.Sprintf("server throttled")
}

func NewThrottledError(retryAfter time.Duration) *ThrottledError {
	return &ThrottledError{RetryAfter: retryAfter}
}

type Scheduler struct {
	mu sync.Mutex

	// Earliest unallocated sending slot.
	next time.Time

	// No request should be sent before this time.
	blockedUntil time.Time

	minInterval time.Duration
	maxInterval time.Duration
	interval    time.Duration

	recoveryEvery time.Duration
	nextRecovery  time.Time

	intervalEm *EventManager[time.Duration]
}

func NewScheduler(
	minInterval time.Duration,
	maxInterval time.Duration,
	recoveryEvery time.Duration,
) *Scheduler {
	if minInterval <= 0 {
		panic("minInterval must be positive")
	}
	if maxInterval < minInterval {
		panic("maxInterval must not be less than minInterval")
	}
	if recoveryEvery <= 0 {
		panic("recoveryEvery must be positive")
	}

	return &Scheduler{
		minInterval:   minInterval,
		maxInterval:   maxInterval,
		interval:      minInterval,
		recoveryEvery: recoveryEvery,

		intervalEm: NewEventManager[time.Duration](),
	}
}

// Reserve allocates the next site-level sending slot.
func (s *Scheduler) Reserve() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	originalInterval := s.interval
	now := time.Now()
	s.recoverLocked(now)
	if s.interval != originalInterval {
		s.intervalEm.NotifySubscribers(s.interval)
	}

	candidate := now

	if s.next.After(candidate) {
		candidate = s.next
	}
	if s.blockedUntil.After(candidate) {
		candidate = s.blockedUntil
	}

	// next represents the earliest slot that can be allocated afterwards.
	s.next = candidate.Add(s.interval)

	return candidate.Sub(now)
}

// Throttle handles a site-wide rate-limit signal.
//
// retryAfter should be zero when the server did not provide one.
func (s *Scheduler) Throttle(retryAfter time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	originalInterval := s.interval
	now := time.Now()
	s.recoverLocked(now)

	if s.interval >= s.maxInterval/2 {
		s.interval = s.maxInterval
	} else {
		s.interval *= 2
	}
	if s.interval != originalInterval {
		s.intervalEm.NotifySubscribers(s.interval)
	}

	// Without Retry-After, at least pause for one current interval.
	pause := s.interval
	if retryAfter > pause {
		pause = retryAfter
	}

	until := now.Add(pause)
	if until.After(s.blockedUntil) {
		s.blockedUntil = until
	}

	if s.blockedUntil.After(s.next) {
		s.next = s.blockedUntil
	}

	// A new 429 restarts the quiet recovery window.
	s.nextRecovery = now.Add(s.recoveryEvery)
}

// Pause temporarily blocks requests without changing the long-term interval.
// This is useful for 503 + Retry-After, maintenance windows, etc.
func (s *Scheduler) Pause(delay time.Duration) {
	if delay <= 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	until := time.Now().Add(delay)
	if until.After(s.blockedUntil) {
		s.blockedUntil = until
	}
	if s.blockedUntil.After(s.next) {
		s.next = s.blockedUntil
	}
}

func (s *Scheduler) recoverLocked(now time.Time) {
	for s.interval > s.minInterval && !s.nextRecovery.IsZero() && !now.Before(s.nextRecovery) {
		s.interval /= 2
		if s.interval < s.minInterval {
			s.interval = s.minInterval
		}

		s.nextRecovery = s.nextRecovery.Add(s.recoveryEvery)
	}

	if s.interval == s.minInterval {
		s.nextRecovery = time.Time{}
	}
}

func (s *Scheduler) ThrottleApplied() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.recoverLocked(now)

	return s.interval > s.minInterval || s.blockedUntil.After(now)
}

func (s *Scheduler) Interval() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.recoverLocked(time.Now())
	return s.interval
}

// ResetAdaptiveState clears all temporary scheduling effects caused by
// Pause and Throttle.
//
// It should be called when a network change makes the previous scheduling
// observations obsolete, such as switching networks or obtaining a new
// outbound IP address.
func (s *Scheduler) ResetAdaptiveState() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	s.blockedUntil = time.Time{}
	s.interval = s.minInterval
	s.nextRecovery = time.Time{}

	// Discard slots that were pushed into the future by Pause or Throttle.
	// The next reservation may start immediately.
	s.next = now
}

func (s *Scheduler) SubscribeIntervalEvent() *EventSubscriber[time.Duration] {
	return s.intervalEm.SubscribeEvent()
}
