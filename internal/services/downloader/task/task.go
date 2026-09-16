package task

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
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

	downloading atomic.Bool
	canceled    atomic.Bool

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

	// cancelFn cancels the attempt that is currently running. It is replaced by
	// beginAttempt at the start of every attempt and cleared by endAttempt, so
	// it must always be accessed under cancelMu.
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
// flight; the download loop is expected to install a fresh context and try
// again. A single task wide context cannot express that: once it is cancelled
// it stays cancelled, so every following attempt would fail immediately
// without performing any work (and the retry loop would spin at full speed).
//
// Cancel is different: it sets the canceled flag, which permanently terminates
// the task no matter how many attempts are left.

// beginAttempt installs a fresh cancellation scope for one attempt and returns
// it together with its cancel function. The caller must pass the returned
// cancel function to endAttempt.
//
// Cancel and beginAttempt are serialised by cancelMu, and beginAttempt
// re-checks the canceled flag while still holding it. Without that, a Cancel
// landing between the loop's canceled check and the installation of the new
// cancel function would find cancelFn pointing at the previous (already
// finished) attempt, and the fresh attempt would run to completion before the
// loop noticed the cancellation.
func (t *Task) beginAttempt() (context.Context, context.CancelCauseFunc) {
	ctx, cancel := context.WithCancelCause(context.Background())

	t.cancelMu.Lock()
	t.cancelFn = cancel
	alreadyCanceled := t.canceled.Load()
	t.cancelMu.Unlock()

	if alreadyCanceled {
		cancel(ErrCanceled)
	}

	return ctx, cancel
}

// endAttempt releases the scope installed by beginAttempt.
//
// The cancel function is cleared first: a Restart or CloseConnection racing
// with the end of an attempt then observes a nil cancel function and does
// nothing, which is correct because the attempt it wanted to abort has already
// finished (and the loop is about to decide what to do next anyway).
func (t *Task) endAttempt(cancel context.CancelCauseFunc) {
	t.cancelMu.Lock()
	t.cancelFn = nil
	t.cancelMu.Unlock()

	cancel(nil)
}

// cancelCurrentAttempt aborts the attempt that is currently in flight, if any.
func (t *Task) cancelCurrentAttempt(cause error) {
	t.cancelMu.Lock()
	cancel := t.cancelFn
	t.cancelMu.Unlock()

	if cancel != nil {
		cancel(cause)
	}
}

func (t *Task) Restart() {
	if t.canceled.Load() {
		return
	}

	t.cancelCurrentAttempt(ErrRestarted)
}

func (t *Task) CloseConnection() {
	if t.canceled.Load() {
		return
	}

	t.cancelCurrentAttempt(ErrConnectionTimeoutClosed)
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
// it does not only abort the current attempt: the canceled flag prevents any
// further attempt from starting, and beginAttempt also honours it for a scope
// that is being installed concurrently.
func (t *Task) Cancel() {
	t.cancelMu.Lock()

	if !t.canceled.CompareAndSwap(false, true) {
		t.cancelMu.Unlock()
		return
	}

	cancel := t.cancelFn
	t.cancelMu.Unlock()

	t.Traffic.Cancel()
	if cancel != nil {
		cancel(ErrCanceled)
	}
}

// Event

func (t *Task) SubscribeChanges() *utils.EventSubscriber[TaskChangeType] {
	return t.em.SubscribeEvent()
}
