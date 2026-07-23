package preloader

import (
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/playlist"
)

type Service struct {
	service.BaseService[Config]

	cacheSvc      *cache_manager.Service
	downloaderSvc *downloader.Service

	pl *playlist.PlayList

	cfgCh chan struct{}

	plMu sync.RWMutex
}

func New(cfg Config, cacheSvc *cache_manager.Service, downloaderSvc *downloader.Service) *Service {
	s := &Service{
		BaseService: service.ConstructBaseService(cfg),

		cacheSvc:      cacheSvc,
		downloaderSvc: downloaderSvc,

		cfgCh: make(chan struct{}),
	}

	s.SetControl("Preloader", s)

	return s
}

func (s *Service) ServiceStart() error {
	s.Go(s.loop)
	return nil
}

func (s *Service) ServiceStop() error {
	return nil
}

func (s *Service) Enabled() bool {
	return true
}

func (s *Service) UpdateConfig(cfg Config, _ string) error {
	s.Cfg = cfg
	select {
	case s.cfgCh <- struct{}{}:
	default:
	}
	return nil
}
