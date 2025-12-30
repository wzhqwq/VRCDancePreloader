package download

import (
	"sync"
	"time"

	"github.com/samber/lo"
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

	tasks     map[string]*Task
	queue     []string
	scheduler *utils.Scheduler
	em        *utils.EventManager[ManagerChangeType]

	maxParallel int

	inTransaction bool
}

func newDownloadManager(maxParallel int, minInterval time.Duration) *downloadManager {
	return &downloadManager{
		tasks:     make(map[string]*Task),
		queue:     make([]string, 0),
		scheduler: utils.NewScheduler(time.Second*3, minInterval),
		em:        utils.NewEventManager[ManagerChangeType](),

		maxParallel: maxParallel,
	}
}
func (dm *downloadManager) CreateOrGetPausedTask(id string) *Task {
	dm.Lock()
	defer dm.unlockAndUpdate()

	task, exists := dm.tasks[id]
	if !exists {
		task = newTask(dm, id)
		dm.tasks[id] = task
		dm.queue = append(dm.queue, id)
	}

	task.sendPriority(-1)

	return task
}
func (dm *downloadManager) CancelDownload(ids ...string) {
	dm.Lock()
	defer dm.unlockAndUpdate()

	for _, id := range ids {
		if task, ok := dm.tasks[id]; ok {
			task.Cancel()
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
	for _, task := range dm.tasks {
		task.Cancel()
		task.Lock()
		task.Unlock()
	}
	dm.em.NotifySubscribers(Stopped)
}

func (dm *downloadManager) GetQueueSnapshot() []*Task {
	dm.Lock()
	defer dm.Unlock()

	return lo.Map(dm.queue, func(id string, _ int) *Task {
		return dm.tasks[id]
	})
}
