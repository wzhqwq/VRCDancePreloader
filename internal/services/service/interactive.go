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
	if s.running {
		return interactive.RunnerStatus{
			Running: true,
		}
	}

	return interactive.RunnerStatus{
		Running: false,
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
