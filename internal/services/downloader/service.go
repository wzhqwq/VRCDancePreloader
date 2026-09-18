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

	// Failed tasks are re-run from the manager's due table rather than by one
	// timer goroutine per failure, so something has to tick.
	//
	// Hosting the ticker on the service means it observes stopCh, which
	// BaseService.Shutdown closes before ServiceStop runs. Note that it is *not*
	// joined: BaseService.Wg is written by Go() but never waited on (see
	// notes/02-host-service-framework.md). So the ticker has usually returned by
	// the time Destroy runs, but nothing guarantees it — the same structural gap
	// that leaves the download goroutines untracked.
	s.Go(s.retryLoop)

	return nil
}

// retryTickInterval is how often due retries are looked for. With the 3s retry
// delay this lands a retry 3.0~3.5s after the failure, against the 3s the per
// task timer used to give.
const retryTickInterval = 500 * time.Millisecond

// retryLoop runs the scheduled retries of both managers until the service stops.
func (s *Service) retryLoop(stopCh <-chan struct{}) {
	ticker := time.NewTicker(retryTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			for _, dm := range s.managers {
				dm.runDueRetries()
			}
		}
	}
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

	// The same runner the scheduled retries use, so both paths refresh the queue
	// the same way.
	dm.startTask(t)

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
