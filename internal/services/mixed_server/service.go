package mixed_server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
)

type Service struct {
	service.BaseService

	cfg Config

	server *mixedServer
}

func New(cfg Config) (*Service, error) {
	s := &Service{
		cfg: cfg,

		server: newMixedServer(cfg),
	}

	s.SetControl("Mixed Server", s)

	return s, nil
}

func (s *Service) ServiceStart() error {
	logger.InfoLn("Starting server on port", s.cfg.Port)

	if err := s.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Service) ServiceStop() error {
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := runningServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown mixed server: %v", err)
	}
	return nil
}

func (s *Service) UpdateConfig(cfg Config) {
	sitesChanged := false
	if len(cfg.InterceptedSites) != len(s.cfg.InterceptedSites) {
		sitesChanged = true
	} else {
		for i, site := range s.cfg.InterceptedSites {
			if cfg.InterceptedSites[i] != site {
				sitesChanged = true
				break
			}
		}
	}

	if cfg.EnableHttpsProxy != s.cfg.EnableHttpsProxy || sitesChanged {
		s.server.UpdateProxy(cfg)
	}
	if cfg.Port != s.cfg.Port {
		s.Stop()
		s.server.UpdatePort(cfg)
		s.Start()
	}
	s.cfg = cfg
}

func (s *Service) UpdatePort(port int) {
	newCfg := s.cfg
	newCfg.Port = port
	s.UpdateConfig(newCfg)
}

func (s *Service) UpdateEnableHttpsProxy(enableHttpsProxy bool) {
	newCfg := s.cfg
	newCfg.EnableHttpsProxy = enableHttpsProxy
	s.UpdateConfig(newCfg)
}

func (s *Service) GetConfig() Config {
	return s.cfg
}
