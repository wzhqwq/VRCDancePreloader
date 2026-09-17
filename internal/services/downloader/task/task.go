package task

import (
	"io"
	"sync"
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

// Task is the reusable download engine. It must not be copied after first use:
// it contains mutexes.
type Task struct {
	io.Writer

	// run owns the re-entrancy guard, the permanent cancellation flag and the
	// cancellation scope of the attempt that is currently in flight.
	run runControl

	ID string

	// Traffic, Remote, Local and em are fixed at construction time, so they can
	// be read without synchronisation.
	Traffic TrafficControl
	Remote  RemoteProvider
	Local   LocalProvider
	em      *utils.EventManager[TaskChangeType]

	// mu guards every field below it.
	//
	// The download goroutine is the only writer; the queue manager, song and the
	// GUI read the same values from other goroutines. That is why the fields are
	// unexported and only reachable through the accessors further down — reading
	// them as plain fields was a data race (see
	// .agent-docs/review/06-task-state-cross-goroutine-race.md).
	//
	// Rule for every setter: compute the new values inside the critical section,
	// then publish the event with em.NotifySubscribers *after* releasing mu.
	// Subscribers run in this goroutine, so notifying while holding the lock
	// would deadlock any subscriber that reads the task back.
	mu sync.Mutex

	state          TaskState
	err            error
	totalSize      int64
	downloadedSize int64
	resolverStatus interactive.RemoteStatus

	// eta is replaced rather than mutated in place, and only ever read or
	// written under mu (its own internal window is not safe for concurrent
	// access either).
	eta *etaCalculator

	// lastBodyRequest is only touched by the download goroutine (Download is
	// serialized by run), so it needs no lock.
	lastBodyRequest time.Time
}

func NewTask(id string, remote RemoteProvider, local LocalProvider) *Task {
	return &Task{
		ID: id,

		Traffic: &nopTrafficControl{},
		Remote:  remote,
		Local:   local,

		em: utils.NewEventManager[TaskChangeType](),

		eta: newEtaCalculator(0),
	}
}

func ConstructTask(id string, traffic TrafficControl, remote RemoteProvider, local LocalProvider) Task {
	return Task{
		ID: id,

		Traffic: traffic,
		Remote:  remote,
		Local:   local,

		em: utils.NewEventManager[TaskChangeType](),

		eta: newEtaCalculator(0),
	}
}

// Accessors.
//
// Everything below is the only way to read the state of a task from another
// goroutine. They all share Task.mu with the setters, so a reader can never see
// a half written value.

// State returns the state the task is in right now.
func (t *Task) State() TaskState {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.state
}

// Err returns the error that stopped the task, or nil while it is still going.
func (t *Task) Err() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.err
}

// StateAndError returns the state together with the error in one critical
// section. Callers that branch on "did it fail, and if not, which state is it
// in" must use this instead of calling State and Err separately: two calls can
// land on either side of a transition and mix two different states.
func (t *Task) StateAndError() (TaskState, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.state, t.err
}

// TotalSize is the size the resolver reported, or 0 while it is unknown.
func (t *Task) TotalSize() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.totalSize
}

// DownloadedSize is how much of the file has been written so far.
func (t *Task) DownloadedSize() int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.downloadedSize
}

// Resolver is the last status the remote resolver reported.
func (t *Task) Resolver() interactive.RemoteStatus {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.resolverStatus
}

// QueryEta estimates when the download will finish. The boolean is false while
// the speed or the total size is still unknown.
func (t *Task) QueryEta() (time.Time, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.eta.QueryEta()
}

// Passed is how long the current attempt has been running.
func (t *Task) Passed() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.eta.Passed()
}

// Setters. Every one of them publishes its event after releasing the lock.

func (t *Task) setState(state TaskState) {
	t.mu.Lock()
	t.state = state
	if state == TaskInitial || state == TaskCompleted {
		t.err = nil
	}
	t.mu.Unlock()

	t.em.NotifySubscribers(State)
}

func (t *Task) setError(err error) {
	t.mu.Lock()
	t.err = err
	t.mu.Unlock()

	t.notifyStateChange()
}

func (t *Task) notifyStateChange() {
	t.em.NotifySubscribers(State)
}

func (t *Task) setTotalSize(size int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalSize = size
}

func (t *Task) setDownloadedSize(size int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.downloadedSize = size
}

func (t *Task) setResolverStatus(status interactive.RemoteStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.resolverStatus = status
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
// eta is allocated by the constructors so that it is never nil (Passed() used
// to panic when a resolving task was throttled before the first reset). The
// calculator itself is replaced rather than mutated in place, so a reader that
// already holds a reference keeps seeing a consistent window.
func (t *Task) resetEta() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.eta = newEtaCalculator(t.totalSize - t.downloadedSize)
}

func (t *Task) addBytes(size int64) {
	// eta's own window is not safe for concurrent access either, so the whole
	// update happens under the same lock the readers take.
	t.mu.Lock()
	t.downloadedSize += size
	t.eta.Add(size)
	t.mu.Unlock()

	t.em.NotifySubscribers(Progress)
}

// Speed is the measured download speed in bytes per second.
func (t *Task) Speed() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.eta.QuerySpeed()
}

// RemainTime estimates how long the download still needs, or -1 when unknown.
func (t *Task) RemainTime() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.eta.QueryRemainTime()
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
