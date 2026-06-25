package service

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type Control interface {
	ServiceStart() error
	ServiceStop() error
}

type BaseService struct {
	InteractiveService

	name   string
	logger *utils.CustomLogger

	running   bool
	lastError error
	em        *utils.EventManager[Status]

	mu       sync.Mutex
	shutdown atomic.Bool

	control Control
}

func (s *BaseService) Start() {
	if s.shutdown.Load() {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	if s.control == nil {
		panic(errors.New("control is nil"))
	}

	defer s.notify()

	err := s.control.ServiceStart()
	if err != nil {
		s.logger.Errorf("Failed to start %s service: %v", s.name, err)
		s.lastError = err
		return
	}

	s.running = true
	s.lastError = nil
}

func (s *BaseService) Shutdown() error {
	if !s.shutdown.CompareAndSwap(false, true) {
		return nil
	}

	if !s.running {
		return nil
	}

	if s.control == nil {
		return errors.New("control is nil")
	}
	return s.control.ServiceStop()
}

func (s *BaseService) Stop() {
	if s.shutdown.Load() {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.control == nil {
		return
	}

	defer s.notify()

	s.running = false
	err := s.control.ServiceStop()
	if err != nil {
		s.logger.Errorf("Failed to stop %s service: %v", s.name, err)
		s.lastError = err
	}
}

func (s *BaseService) SetControl(name string, control Control) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.name = name
	s.control = control
	s.logger = utils.NewLogger(name)
}

func (s *BaseService) L() *utils.CustomLogger {
	return s.logger
}
