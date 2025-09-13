package utils

import (
	"sync"
	"time"
)

type Scheduler struct {
	mu          sync.Mutex
	next        time.Time
	minDelay    time.Duration
	minInterval time.Duration

	delay    time.Duration
	interval time.Duration
}

func NewScheduler(minDelay, minInterval time.Duration) *Scheduler {
	return &Scheduler{
		minDelay:    minDelay,
		minInterval: minInterval,
		delay:       minDelay,
		interval:    minInterval,
	}
}

func (s *Scheduler) ReserveWithDelay() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	earliest := now.Add(s.delay)
	candidate := s.next.Add(s.interval)
	if earliest.After(candidate) {
		candidate = earliest
	}
	s.next = candidate

	wait := candidate.Sub(now)
	if wait < 0 {
		wait = 0
	}

	return wait
}

func (s *Scheduler) Reserve() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	candidate := s.next.Add(s.interval)
	wait := candidate.Sub(now)

	if wait < 0 {
		wait = 0
		s.next = now
	} else {
		s.next = candidate
	}

	return wait
}

func (s *Scheduler) SlowDown() {
	s.mu.Lock()
	s.interval += time.Second
	s.mu.Unlock()
}

func (s *Scheduler) Resume() {
	s.mu.Lock()
	s.interval = max(s.minInterval, s.interval-time.Second)
	s.mu.Unlock()
}
