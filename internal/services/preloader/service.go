package preloader

import (
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
)

type Service struct {
	service.BaseService[Config]

	cacheSvc      *cache_manager.Service
	downloaderSvc *downloader.Service

	duduOriginalVideoInfoProvider *third_parties.DDFDOriginalVideoInfoProvider

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
	s.duduOriginalVideoInfoProvider = third_parties.NewDDFOriginalVideoInfoProvider()
	s.duduOriginalVideoInfoProvider.SetEnabled(s.Cfg.HighFramerateFallback)

	s.Go(s.loop)
	return nil
}

func (s *Service) ServiceStop() error {
	s.duduOriginalVideoInfoProvider.Close()
	s.duduOriginalVideoInfoProvider = nil

	return nil
}

func (s *Service) Enabled() bool {
	return true
}

func (s *Service) UpdateConfig(cfg Config, _ string) error {
	if cfg.HighFramerateFallback != s.Cfg.HighFramerateFallback {
		s.duduOriginalVideoInfoProvider.SetEnabled(cfg.HighFramerateFallback)
	}
	if cfg.LowSpeedFallback != s.Cfg.LowSpeedFallback {
		defer s.healthCheck()
	}

	s.Cfg = cfg
	select {
	case s.cfgCh <- struct{}{}:
	default:
	}
	return nil
}
