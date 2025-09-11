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
}

func NewScheduler(minDelay, minInterval time.Duration) *Scheduler {
	return &Scheduler{
		minDelay:    minDelay,
		minInterval: minInterval,
	}
}

func (s *Scheduler) Reserve() time.Duration {
	now := time.Now()

	s.mu.Lock()
	earliest := now.Add(s.minDelay)
	candidate := s.next.Add(s.minInterval)
	if earliest.After(candidate) {
		candidate = earliest
	}
	s.next = candidate
	s.mu.Unlock()

	wait := candidate.Sub(now)
	if wait < 0 {
		wait = 0
	}
	return wait
}

func (s *Scheduler) SlowDown() {
	s.mu.Lock()
	s.minInterval += time.Second
	s.mu.Unlock()
}
