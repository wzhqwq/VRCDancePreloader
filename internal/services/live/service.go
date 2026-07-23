package live

import (
	"context"
	"fmt"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
)

type Service struct {
	service.BaseService[Config]

	server *Server
}

func New(cfg Config) *Service {
	s := &Service{
		BaseService: service.ConstructBaseService(cfg),
	}

	s.SetControl("Live Server", s)

	return s
}

func (s *Service) ServiceStart() error {
	s.server = NewLiveServer(s)
	err := s.ServeAndTest(s.Cfg.Port, s.server)
	if err != nil {
		return err
	}

	s.Go(s.server.ws.Loop)
	return nil
}

func (s *Service) ServiceStop() error {
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), time.Second)
	defer shutdownRelease()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown live server: %v", err)
	}
	return nil
}

func (s *Service) Enabled() bool {
	return s.Cfg.Enabled
}

func (s *Service) UpdateConfig(cfg Config) error {
	if cfg.Enabled != s.Cfg.Enabled {
		if cfg.Enabled {
			s.Start()
		} else {
			s.Stop()
		}
	}
	if cfg.Port != s.Cfg.Port {
		s.Stop()
		s.Start()
	}
	s.Cfg = cfg

	return nil
}
