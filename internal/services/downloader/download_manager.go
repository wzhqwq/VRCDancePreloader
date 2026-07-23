package downloader

import (
	"sync"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader/task"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type ManagerChangeType string

const (
	QueueChange ManagerChangeType = "queue"
	Stopped     ManagerChangeType = "stopped"
)

type downloadManager struct {
	sync.Mutex
	//utils.LoggingMutex

	tasks     map[string]*ManagedTask
	queue     []string
	scheduler *utils.Scheduler
	em        *utils.EventManager[ManagerChangeType]

	queueLogger *utils.UniqueLogger

	maxParallel int

	inTransaction bool
}

func newDownloadManager(maxParallel int, scheduler *utils.Scheduler) *downloadManager {
	return &downloadManager{
		tasks:     make(map[string]*ManagedTask),
		queue:     make([]string, 0),
		scheduler: scheduler,
		em:        utils.NewEventManager[ManagerChangeType](),

		queueLogger: utils.NewUniqueLogger("Download Queue"),

		maxParallel: maxParallel,
	}
}

type RemoteProviderFn func() task.RemoteProvider
type LocalProviderFn func(impl utils.LoggerImpl) task.LocalProvider

func (dm *downloadManager) CreateOrGetPausedTask(id string, remoteFn RemoteProviderFn, localFn LocalProviderFn) *ManagedTask {
	dm.Lock()
	defer dm.unlockAndUpdate()

	t, exists := dm.tasks[id]
	if !exists {
		local := localFn(logger)
		if local == nil {
			return nil
		}
		t = newManagedTask(dm, id, remoteFn(), local)
		dm.tasks[id] = t
		dm.queue = append(dm.queue, id)
	}

	t.Traffic.(*traffic).sendPriority(-1)

	return t
}
func (dm *downloadManager) CancelDownload(ids ...string) {
	dm.Lock()
	defer dm.unlockAndUpdate()

	for _, id := range ids {
		if t, ok := dm.tasks[id]; ok {
			t.Cancel()
			delete(dm.tasks, id)
		}
	}
}
func (dm *downloadManager) unlockAndUpdate() {
	dm.Unlock()
	dm.UpdatePriorities()
}
func (dm *downloadManager) SetMaxParallel(max int) {
	dm.maxParallel = max
	dm.UpdatePriorities()
}
func (dm *downloadManager) Destroy() {
	dm.Lock()
	defer dm.Unlock()
	for _, t := range dm.tasks {
		t.Cancel()
	}
	dm.em.NotifySubscribers(Stopped)
}

func (dm *downloadManager) Subscribe() *utils.EventSubscriber[ManagerChangeType] {
	return dm.em.SubscribeEvent()
}

func (dm *downloadManager) GetQueueSnapshot() []*ManagedTask {
	dm.Lock()
	defer dm.Unlock()

	return lo.Map(dm.queue, func(id string, _ int) *ManagedTask {
		return dm.tasks[id]
	})
}
