package download

import (
	"io"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type TaskChangeType string

const (
	Progress TaskChangeType = "progress"
	State    TaskChangeType = "state"
)

type Task struct {
	sync.Mutex
	io.Writer

	manager *downloadManager

	connected bool

	ID string

	TotalSize      int64
	DownloadedSize int64

	Requesting bool
	Done       bool
	Pending    bool
	Cooling    bool
	Error      error

	eta *etaCalculator
	em  *utils.EventManager[TaskChangeType]

	CancelCh   chan struct{}
	PriorityCh chan int
	RestartCh  chan struct{}
}

func newTask(manager *downloadManager, id string) *Task {
	return &Task{
		manager: manager,

		ID: id,

		em: utils.NewEventManager[TaskChangeType](),

		CancelCh:   make(chan struct{}),
		PriorityCh: make(chan int, 1),
		RestartCh:  make(chan struct{}, 1),
	}
}

func (t *Task) unlockAndNotifyStateChange() {
	t.Unlock()
	t.notifyStateChange()
}

func (t *Task) notifyStateChange() {
	t.em.NotifySubscribers(State)
}

// Non-blocking latest message, only be called by manager and must be protected by mutex

func (t *Task) sendPriority(priority int) {
	// flush first
	select {
	case <-t.PriorityCh:
	default:
	}
	t.PriorityCh <- priority
}

func (t *Task) restart() {
	// flush first
	select {
	case <-t.RestartCh:
	default:
	}
	t.RestartCh <- struct{}{}
}

// ETA

func (t *Task) resetEta() {
	t.eta = newEtaCalculator(t.TotalSize - t.DownloadedSize)
}

func (t *Task) addBytes(size int64) {
	t.DownloadedSize += size
	t.eta.Add(size)
	t.em.NotifySubscribers(Progress)
}

// Destroy

func (t *Task) Cancel() {
	close(t.CancelCh)
}

// Event

func (t *Task) SubscribeChanges() *utils.EventSubscriber[TaskChangeType] {
	return t.em.SubscribeEvent()
}
