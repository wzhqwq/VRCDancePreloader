package downloader

import (
	"strings"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewLogger("Downloader")

type Service struct {
	service.BaseService[Config]

	managers map[string]*downloadManager
}

func New(cfg Config) *Service {
	s := &Service{
		BaseService: service.ConstructBaseService(cfg),

		managers: make(map[string]*downloadManager),
	}

	s.SetControl("Downloader", s)

	return s
}

func (s *Service) ServiceStart() error {
	s.managers["pypy"] = newDownloadManager(s.Cfg.MaxDownload, utils.PyPyVideoScheduler())
	s.managers["default"] = newDownloadManager(s.Cfg.MaxDownload, utils.SharedVideoScheduler())
	return nil
}

func (s *Service) ServiceStop() error {
	for _, dm := range s.managers {
		dm.Destroy()
	}
	return nil
}

func (s *Service) Enabled() bool {
	return true
}

func (s *Service) UpdateConfig(cfg Config, field string) error {
	if cfg.MaxDownload != s.Cfg.MaxDownload {
		for _, dm := range s.managers {
			dm.SetMaxParallel(cfg.MaxDownload)
		}
	}

	s.Cfg = cfg

	return nil
}

func (s *Service) Prioritize(ids ...string) {
	if len(ids) == 0 {
		return
	}
	for _, dm := range s.managers {
		dm.Prioritize(ids...)
	}
}

// WithFrozenQueue runs fn with permit publication suspended on both managers and
// then publishes the resulting queue order exactly once.
//
// Use it around any queue change that takes more than one step. The preloader is
// the reason this exists: it has to create a task per upcoming song — and
// creating a task appends it to the queue — but only the Prioritize call at the
// end puts the tasks in their real order (currently playing first). Without the
// freeze, every creation, and every unrelated queue change happening in between
// (a task finishing, a video request, a cancel), would publish permits for the
// half-built order, and a task granted that way starts downloading in a position
// it is about to lose.
//
// The freeze is reference counted and released even if fn panics.
func (s *Service) WithFrozenQueue(fn func()) {
	for _, dm := range s.managers {
		dm.freeze()
	}
	defer func() {
		for _, dm := range s.managers {
			dm.thaw()
		}
	}()

	fn()
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

func (s *Service) UpdateRequestEta(id string, eta time.Time, duration time.Duration) {
	s.findManager(id).UpdateRequestEta(id, eta, duration)
}

func (s *Service) findManager(id string) *downloadManager {
	if strings.HasPrefix(id, "pypy") {
		return s.managers["pypy"]
	}
	return s.managers["default"]
}
