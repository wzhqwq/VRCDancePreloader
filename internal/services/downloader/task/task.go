package task

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type TaskChangeType string

const (
	Progress TaskChangeType = "progress"
	State    TaskChangeType = "state"
)

type TaskState int

const (
	TaskInitial = iota
	TaskPending
	TaskWaitScheduled
	TaskResolving
	TaskRequested
	TaskDownloading
	TaskCompleted
)

type Task struct {
	io.Writer

	downloading atomic.Bool
	canceled    atomic.Bool

	connected bool

	ID string

	TotalSize      int64
	DownloadedSize int64

	State TaskState
	Error error

	Eta     *etaCalculator
	Traffic TrafficControl
	Remote  RemoteProvider
	Local   LocalProvider
	em      *utils.EventManager[TaskChangeType]

	cancelFn context.CancelCauseFunc
	cancelMu sync.Mutex
}

func NewTask(id string, remote RemoteProvider, local LocalProvider) *Task {
	return &Task{
		ID: id,

		Traffic: &nopTrafficControl{},
		Remote:  remote,
		Local:   local,

		em: utils.NewEventManager[TaskChangeType](),
	}
}

func ConstructTask(id string, traffic TrafficControl, remote RemoteProvider, local LocalProvider) Task {
	return Task{
		ID: id,

		Traffic: traffic,
		Remote:  remote,
		Local:   local,

		em: utils.NewEventManager[TaskChangeType](),
	}
}

func (t *Task) setState(state TaskState) {
	t.State = state
	if state == TaskInitial || state == TaskCompleted || state == TaskWaitScheduled {
		t.Error = nil
	}
	t.em.NotifySubscribers(State)
}

func (t *Task) setError(err error) {
	t.Error = err
	t.notifyStateChange()
}

func (t *Task) notifyStateChange() {
	t.em.NotifySubscribers(State)
}

func (t *Task) Restart() {
	if t.canceled.Load() {
		return
	}

	if t.cancelFn != nil {
		t.cancelFn(ErrRestarted)
	}
}

// ETA

func (t *Task) resetEta() {
	t.Eta = newEtaCalculator(t.TotalSize - t.DownloadedSize)
}

func (t *Task) addBytes(size int64) {
	t.DownloadedSize += size
	t.Eta.Add(size)
	t.em.NotifySubscribers(Progress)
}

func (t *Task) Speed() float64 {
	if t.Eta == nil {
		return 0
	}
	return t.Eta.QuerySpeed()
}

func (t *Task) RemainTime() time.Duration {
	if t.Eta == nil {
		return -1
	}
	return t.Eta.QueryRemainTime()
}

// Destroy

func (t *Task) Cancel() {
	if t.canceled.CompareAndSwap(false, true) {
		t.Traffic.Cancel()
		if t.cancelFn != nil {
			t.cancelFn(ErrCanceled)
		}
	}
}

// Event

func (t *Task) SubscribeChanges() *utils.EventSubscriber[TaskChangeType] {
	return t.em.SubscribeEvent()
}
