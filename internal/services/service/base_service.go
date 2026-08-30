package service

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/wzhqwq/VRCDancePreloader/internal/stability"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type Control interface {
	ServiceStart() error
	ServiceStop() error
	Enabled() bool
}

type Service interface {
	Start()
	Shutdown() error
	Status() interactive.RunnerStatus
}

type BaseService[T any] struct {
	interactive.StatefulService
	Service

	name   string
	logger *utils.CustomLogger

	running   bool
	lastError error
	em        *utils.EventManager[interactive.RunnerStatus]

	Wg     sync.WaitGroup
	stopCh chan struct{}

	mu       sync.Mutex
	shutdown atomic.Bool

	control Control

	Cfg T
}

func ConstructBaseService[T any](cfg T) BaseService[T] {
	return BaseService[T]{
		Cfg: cfg,
	}
}

func (s *BaseService[T]) Start() {
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

	if !s.control.Enabled() {
		return
	}
	s.stopCh = make(chan struct{})
	err := s.control.ServiceStart()
	if err != nil {
		s.logger.ErrorLnf("Failed to start %s service: %v", s.name, err)
		s.lastError = err
		return
	}

	s.running = true
	s.lastError = nil
}

func (s *BaseService[T]) Shutdown() error {
	if !s.shutdown.CompareAndSwap(false, true) {
		return nil
	}

	if !s.running {
		return nil
	}

	if s.control == nil {
		return errors.New("control is nil")
	}

	cancel := stability.PanicIfTimeout(s.name + "_ShuttingDown")
	defer cancel()

	s.logger.InfoLn("Shutting down...")
	defer s.logger.InfoLn("Shut down")
	close(s.stopCh)
	return s.control.ServiceStop()
}

func (s *BaseService[T]) Stop() {
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
	close(s.stopCh)
	err := s.control.ServiceStop()
	if err != nil {
		s.logger.ErrorLnf("Failed to stop %s service: %v", s.name, err)
		s.lastError = err
	}
}

func (s *BaseService[T]) SetControl(name string, control Control) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.name = name
	s.control = control
	s.logger = utils.NewLogger(name)
	s.em = utils.NewEventManager[interactive.RunnerStatus]()
}

func (s *BaseService[T]) L() *utils.CustomLogger {
	return s.logger
}

func (s *BaseService[T]) Go(fn func(stopCh <-chan struct{})) {
	s.Wg.Go(func() {
		fn(s.stopCh)
	})
}

type ServerLike interface {
	Serve(l net.Listener) error
}

func (s *BaseService[T]) ServeAndTest(port int, server ServerLike) error {
	s.logger.InfoLn("Starting", s.name, "on port", port)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	s.logger.InfoLn(s.name, "listening on port", port)

	s.Wg.Go(func() {
		err := server.Serve(l)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.ErrorLnf("Service %s stopped because: %v", s.name, err)
			s.lastError = err
			s.Stop()
		}
	})

	resp, err := http.Get(fmt.Sprintf("http://0.0.0.0:%d/alive", port))
	if err != nil {
		s.lastError = fmt.Errorf("server started but test failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		s.lastError = fmt.Errorf("server started but test failed: unexpected status %s", resp.Status)
	}

	return nil
}
