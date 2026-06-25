package service

import (
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type Status struct {
	Running bool
	Error   error
}

type InteractiveService interface {
	SubscribeStatus() *utils.EventSubscriber[Status]
	Status() Status
	Restart()
}

func (s *BaseService) SubscribeStatus() *utils.EventSubscriber[Status] {
	if s.em == nil {
		panic(errors.New("not a interactive service"))
	}
	return s.em.SubscribeEvent()
}

func (s *BaseService) Status() Status {
	if s.running {
		return Status{
			Running: true,
		}
	}

	return Status{
		Running: false,
		Error:   s.lastError,
	}
}

func (s *BaseService) Restart() {
	s.Stop()
	s.Start()
}

func (s *BaseService) notify() {
	if s.em != nil {
		s.em.NotifySubscribers(s.Status())
	}
}
