package service

import (
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (s *BaseService[T]) SubscribeStatus() *utils.EventSubscriber[interactive.RunnerStatus] {
	if s.em == nil {
		panic(errors.New("SetControl not called yet"))
	}
	return s.em.SubscribeEvent()
}

func (s *BaseService[T]) Status() interactive.RunnerStatus {
	// Running and Error are independent: a service can be up while something it
	// depends on failed its self check. Reporting the error next to Running
	// instead of dropping it is what makes that visible; consumers that only
	// care about liveness keep reading Running (host/runtime.go records the
	// error and still treats the service as running).
	//
	// Deliberately lock free: notify() calls this while holding s.mu (Start and
	// Stop both defer notify()), so taking the lock here would self deadlock.
	return interactive.RunnerStatus{
		Running: s.running,
		Error:   s.lastError,
	}
}

func (s *BaseService[T]) Restart() {
	s.Stop()
	s.Start()
}

func (s *BaseService[T]) notify() {
	if s.em != nil {
		s.em.NotifySubscribers(s.Status())
	}
}
