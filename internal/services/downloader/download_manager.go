package downloader

import (
	"sync"
	"time"

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

	// dueAt is when a failed task may be run again, keyed by task id. It is
	// filled by ScheduleRetry and drained by runDueRetries, which the service
	// ticker calls — one entry per pending retry, instead of one goroutine per
	// failed task.
	//
	// Guarded by the embedded Mutex.
	dueAt map[string]time.Time

	// retryDelay is how long a failed task waits before it is run again. It is a
	// per manager field rather than the package level variable this used to be
	// (ManagedTask.tempDelay), so tests can shorten it without affecting
	// anything else, and nothing can change it behind the manager's back.
	retryDelay time.Duration
}

// defaultRetryDelay is the production retry backoff. It is what the old
// ManagedTask.tempDelay was, so the behavior is unchanged.
const defaultRetryDelay = 3 * time.Second

func newDownloadManager(maxParallel int, scheduler *utils.Scheduler) *downloadManager {
	return &downloadManager{
		tasks:     make(map[string]*ManagedTask),
		queue:     make([]string, 0),
		scheduler: scheduler,
		em:        utils.NewEventManager[ManagerChangeType](),

		queueLogger: utils.NewUniqueLogger("Download Queue"),

		maxParallel: maxParallel,

		dueAt:      make(map[string]time.Time),
		retryDelay: defaultRetryDelay,
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

	// A fresh request supersedes a pending retry: the caller wants it now, and
	// Service.Download is about to start it anyway.
	delete(dm.dueAt, id)

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

		// A canceled task must not be resurrected by a scheduled retry.
		delete(dm.dueAt, id)
	}
}
func (dm *downloadManager) unlockAndUpdate() {
	dm.Unlock()
	dm.UpdatePriorities()
}

// startTask runs one download loop in the background and re-derives the queue
// when it returns.
//
// The goroutine is deliberately not tracked by the service lifecycle: that is
// the pre-existing shape of Service.Download, and retries reuse it instead of
// introducing a second, different runner.
func (dm *downloadManager) startTask(t *ManagedTask) {
	go func() {
		t.Download()
		dm.UpdatePriorities()
	}()
}

// ScheduleRetry records that a failed task may be run again after the retry
// delay, and returns that moment so the caller can show a countdown.
//
// This is deliberately a command rather than a query: the caller that decides
// "this failure is worth retrying" is the song state machine (it is where all
// the other error classification lives), and it needs the resulting time in the
// same breath. The old ManagedTask.Retry did the same thing, but as a side
// effect of a getter, and with one un-cancellable timer goroutine per failure.
func (dm *downloadManager) ScheduleRetry(id string) time.Time {
	dm.Lock()
	defer dm.Unlock()

	at := time.Now().Add(dm.retryDelay)
	dm.dueAt[id] = at

	return at
}

// runDueRetries starts every failed task whose retry delay has elapsed. It is
// called by the service ticker (Service.retryLoop).
//
// Note that a retry does not get any special permission: startTask runs the
// ordinary download loop, so the task re-enters the same gate and is judged
// against the current queue order and maxParallel like everybody else. There is
// deliberately no "put it back in the allowed prefix" step — the task never
// left the queue, and forcing it forward would be exactly the bypass the queue
// ordering exists to prevent.
func (dm *downloadManager) runDueRetries() {
	dm.Lock()

	now := time.Now()
	var due []*ManagedTask

	for id, at := range dm.dueAt {
		if now.Before(at) {
			continue
		}

		delete(dm.dueAt, id)

		// The task may have completed, or been canceled, while the retry was
		// pending.
		if t := dm.tasks[id]; t != nil && t.State() != task.TaskCompleted {
			due = append(due, t)
		}
	}

	dm.Unlock()

	// Outside the lock: startTask spawns a goroutine, and the download loop
	// takes the manager lock again as soon as it reaches the gate.
	for _, t := range due {
		dm.startTask(t)
	}
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
