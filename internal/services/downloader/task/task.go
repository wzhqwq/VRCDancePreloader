package task

import (
	"io"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
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
	TaskResolving
	TaskResolvingFailed
	TaskRequested
	TaskDownloading
	TaskCompleted
)

type Task struct {
	io.Writer

	// run owns the re-entrancy guard, the permanent cancellation flag and the
	// cancellation scope of the attempt that is currently in flight.
	run runControl

	ID string

	TotalSize      int64
	DownloadedSize int64

	State TaskState
	Error error

	ResolverStatus interactive.RemoteStatus

	Eta     *etaCalculator
	Traffic TrafficControl
	Remote  RemoteProvider
	Local   LocalProvider
	em      *utils.EventManager[TaskChangeType]

	lastBodyRequest time.Time
}

func NewTask(id string, remote RemoteProvider, local LocalProvider) *Task {
	return &Task{
		ID: id,

		Traffic: &nopTrafficControl{},
		Remote:  remote,
		Local:   local,

		em: utils.NewEventManager[TaskChangeType](),

		Eta: newEtaCalculator(0),
	}
}

func ConstructTask(id string, traffic TrafficControl, remote RemoteProvider, local LocalProvider) Task {
	return Task{
		ID: id,

		Traffic: traffic,
		Remote:  remote,
		Local:   local,

		em: utils.NewEventManager[TaskChangeType](),

		Eta: newEtaCalculator(0),
	}
}

func (t *Task) setState(state TaskState) {
	t.State = state
	if state == TaskInitial || state == TaskCompleted {
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

// Attempt scoped cancellation.
//
// Restart and CloseConnection only abort the attempt that is currently in
// flight, and the download loop installs a fresh context with beginAttempt for
// the next one. Cancel is different: it is irreversible.
//
// All of this state lives in Task.run (see run_control.go); the methods below
// are the task level entry points.

func (t *Task) Restart() {
	if t.run.cancelled() {
		return
	}

	t.run.abortAttempt(ErrRestarted)
}

func (t *Task) CloseConnection() {
	if t.run.cancelled() {
		return
	}

	t.run.abortAttempt(ErrConnectionTimeoutClosed)
}

// ETA

// resetEta restarts the measurement window for the current attempt.
//
// Eta is allocated by the constructors so that it is never nil (Passed() used
// to panic when a resolving task was throttled before the first reset). The
// calculator itself is replaced rather than mutated in place: Eta is read from
// other goroutines (downloadManager.allDownloadingEta, restartIfNeeded), and a
// fresh value keeps those readers from observing a half-reset window.
func (t *Task) resetEta() {
	t.Eta = newEtaCalculator(t.TotalSize - t.DownloadedSize)
}

func (t *Task) addBytes(size int64) {
	t.DownloadedSize += size
	t.Eta.Add(size)
	t.em.NotifySubscribers(Progress)
}

func (t *Task) Speed() float64 {
	return t.Eta.QuerySpeed()
}

func (t *Task) RemainTime() time.Duration {
	return t.Eta.QueryRemainTime()
}

// Cancel permanently terminates the task. Unlike Restart and CloseConnection
// it does not only abort the current attempt: the done phase prevents any
// further attempt from starting, and beginAttempt also honours it for a scope
// that is being installed concurrently.
//
// Traffic is cancelled after the context, so that the attempt scope stays the
// authority even if a waiter happens to be released by the traffic control.
// Both side effects run exactly once, on the call that transitioned the task.
func (t *Task) Cancel() {
	if !t.run.cancelAll() {
		return
	}

	t.Traffic.Cancel()
}

// Event

func (t *Task) SubscribeChanges() *utils.EventSubscriber[TaskChangeType] {
	return t.em.SubscribeEvent()
}
