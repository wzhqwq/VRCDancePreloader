package downloader

import (
	"strings"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewLogger("Downloader")

var fallbackNames []string
var fallbackNamesEm = utils.NewEventManager[[]string]()

func SubscribeFallbackNames() *utils.EventSubscriber[[]string] {
	return fallbackNamesEm.SubscribeEvent()
}

func FallbackNames() []string {
	return fallbackNames
}

type Service struct {
	service.BaseService[Config]

	managers map[string]*downloadManager
}

func New(cfg Config) *Service {
	s := &Service{
		BaseService: service.ConstructBaseService(cfg),

		managers: make(map[string]*downloadManager),
	}

	fallbackNames = cfg.UseYoutubeFallback

	s.SetControl("Downloader", s)

	return s
}

func (s *Service) ServiceStart() error {
	s.managers["pypy"] = newDownloadManager(s.Cfg.MaxDownload, utils.PyPyVideoScheduler())
	s.managers["default"] = newDownloadManager(s.Cfg.MaxDownload, utils.SharedVideoScheduler())
	return nil
}

func (s *Service) ServiceStop() error {
	s.managers["pypy"].Destroy()
	s.managers["default"].Destroy()
	return nil
}

func (s *Service) Enabled() bool {
	return true
}

func (s *Service) UpdateConfig(cfg Config, field string) error {
	if cfg.MaxDownload != s.Cfg.MaxDownload {
		s.managers["pypy"].SetMaxParallel(cfg.MaxDownload)
		s.managers["default"].SetMaxParallel(cfg.MaxDownload)
	}

	s.Cfg = cfg

	return nil
}

func (s *Service) Prioritize(ids ...string) {
	if len(ids) == 0 {
		return
	}
	s.managers["pypy"].Prioritize(ids...)
	s.managers["default"].Prioritize(ids...)
}

func (s *Service) QueueTransaction() func() {
	cancelPyPy := s.managers["pypy"].QueueTransaction()
	cancelDefault := s.managers["default"].QueueTransaction()
	return func() {
		cancelPyPy()
		cancelDefault()
	}
}

func (s *Service) Download(id string, remoteFn RemoteProviderFn, localFn LocalProviderFn) *ManagedTask {
	dm := s.findManager(id)
	t := dm.CreateOrGetPausedTask(id, remoteFn, localFn)
	if t == nil {
		return nil
	}
	go func() {
		t.Download()
		// re-calculate priorities after download
		dm.UpdatePriorities()
	}()

	return t
}

func (s *Service) GetScheduler(id string) *utils.Scheduler {
	return s.findManager(id).scheduler
}

func (s *Service) UpdateRequestEta(id string, eta time.Time, duration time.Duration) {
	s.findManager(id).UpdateRequestEta(id, eta, duration)
}

func (s *Service) findManager(id string) *downloadManager {
	if strings.HasPrefix(id, "pypy") {
		return s.managers["pypy"]
	}
	return s.managers["default"]
}

func (s *Service) SubscribeCoolDownInterval(name string) *utils.EventSubscriber[time.Duration] {
	dm, ok := s.managers[name]
	if !ok {
		return nil
	}
	return dm.scheduler.SubscribeIntervalEvent()
}
