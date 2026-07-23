package mixed_server

import (
	"context"
	"fmt"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/preloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
)

type Service struct {
	service.BaseService[Config]

	server *mixedServer

	preloaderSvc *preloader.Service
}

func New(cfg Config, preloaderSvc *preloader.Service) *Service {
	s := &Service{
		BaseService: service.ConstructBaseService(cfg),

		preloaderSvc: preloaderSvc,
	}

	s.SetControl("Mixed Server", s)

	return s
}

func (s *Service) ServiceStart() error {
	s.server = newMixedServer(s.Cfg, s)
	return s.ServeAndTest(s.Cfg.Port, s.server)
}

func (s *Service) ServiceStop() error {
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownRelease()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown mixed server: %v", err)
	}
	return nil
}

func (s *Service) Enabled() bool {
	return true
}

func (s *Service) UpdateConfig(cfg Config) error {
	sitesChanged := false
	if len(cfg.InterceptedSites) != len(s.Cfg.InterceptedSites) {
		sitesChanged = true
	} else {
		for i, site := range s.Cfg.InterceptedSites {
			if cfg.InterceptedSites[i] != site {
				sitesChanged = true
				break
			}
		}
	}

	if cfg.EnableHttpsProxy != s.Cfg.EnableHttpsProxy || sitesChanged {
		s.server.UpdateProxy(cfg)
	}
	if cfg.Port != s.Cfg.Port {
		s.Stop()
		s.Start()
	}
	s.Cfg = cfg

	return nil
}
