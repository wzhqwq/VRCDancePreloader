package service

import (
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

// WatchStatus hands the caller its own watcher of the service status level: wait
// on Wakes, then read Status (or the watcher's Current) for the value.
func (s *BaseService[T]) WatchStatus() *utils.LevelWatcher[interactive.RunnerStatus] {
	if s.status == nil {
		panic(errors.New("SetControl not called yet"))
	}
	return s.status.Subscribe()
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
	if s.status != nil {
		s.status.Store(s.Status())
	}
}
