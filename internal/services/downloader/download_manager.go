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

	// batch counts the callers that are currently assembling a queue change.
	//
	// While it is non zero the queue order is not final, so no permit may be
	// published: a permit granted for an intermediate order is a permit granted
	// for a position the task is about to lose, and a granted task starts
	// downloading and does not re-read the permit until its attempt restarts.
	//
	// Guarded by the embedded Mutex.
	batch int
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

	// Keep the task paused until the queue has placed it.
	//
	// This is the primary reason for the -1: a task must not start before
	// Prioritize has given it its real position. The task is already appended to
	// dm.queue above, but only at the tail — the order that matters (currently
	// playing first) is established later, by the caller's Prioritize call. So
	// the permit has to start out denied, and -1 is deliberately outside any
	// allowed prefix.
	//
	// The second, narrower reason: a task that is re-requested after it
	// completed is still in dm.tasks but no longer in dm.queue (UpdatePriorities
	// filters completed tasks out), so the publish below cannot reach it. It
	// keeps whatever allowed value it had when it completed unless it is reset
	// here.
	//
	// Publishing during a batch that creates several tasks is suppressed by
	// Service.WithFrozenQueue, which is what keeps this -1 in force until the
	// batch has decided the final order.
	t.Traffic.NotifyPermit(-1, false)

	return t
}
func (dm *downloadManager) CancelDownload(ids ...string) {
	dm.Lock()
	defer dm.unlockAndUpdate()

	for _, id := range ids {
		if t, ok := dm.tasks[id]; ok {
			t.Task.Cancel()
			delete(dm.tasks, id)
		}
	}
}
func (dm *downloadManager) unlockAndUpdate() {
	dm.Unlock()
	dm.UpdatePriorities()
}

// freeze suspends permit publication until the matching thaw.
//
// A caller that changes the queue in more than one step (create the tasks, then
// order them) has to freeze around the whole assembly: publishing after each
// individual step would hand out permits for a queue order that is only
// half-built. Freezing is reference counted, so nested batches only publish
// once, when the outermost one ends.
//
// Deliberately not a lock: the caller runs its own code while frozen, and that
// code takes dm.Lock() through the normal entry points. Only the *publication*
// of permits is suspended, not the mutation of the queue.
func (dm *downloadManager) freeze() {
	dm.Lock()
	defer dm.Unlock()

	dm.batch++
}

// thaw ends one freeze and publishes the permits once the last one ends, so
// that the final queue order is what the tasks are judged against.
func (dm *downloadManager) thaw() {
	dm.Lock()
	if dm.batch > 0 {
		dm.batch--
	}
	final := dm.batch == 0
	dm.Unlock()

	if final {
		dm.UpdatePriorities()
	}
}

// SetMaxParallel changes the parallel limit and republishes the permits
// immediately, so a task that fell out of (or into) the allowed prefix learns
// about it without waiting for the next queue change.
func (dm *downloadManager) SetMaxParallel(max int) {
	dm.Lock()
	defer dm.Unlock()

	dm.maxParallel = max
	dm.publishPermitsLocked()
}
func (dm *downloadManager) Destroy() {
	dm.Lock()
	defer dm.Unlock()
	for _, t := range dm.tasks {
		t.Task.Cancel()
	}
	dm.em.NotifySubscribers(Stopped)
}

func (dm *downloadManager) Subscribe() *utils.EventSubscriber[ManagerChangeType] {
	return dm.em.SubscribeEvent()
}

func (dm *downloadManager) GetQueueSnapshot() []*ManagedTask {
	dm.Lock()
	defer dm.Unlock()

	return lo.FilterMap(dm.queue, func(id string, _ int) (*ManagedTask, bool) {
		t, ok := dm.tasks[id]
		return t, ok
	})
}
