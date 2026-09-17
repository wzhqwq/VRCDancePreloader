package downloader

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader/task"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// traffic is the queue controlled TrafficControl implementation.
//
// The permit is *state* (position + allowed) rather than a stream of messages:
// WaitPending re-reads it on every gate entry, which is what makes "abort the
// attempt and re-check the queue" correct. The previous design published
// positions through a capacity-1 channel and treated "no pending message" as
// "still permitted" — that is only sound while a task stays inside one approved
// download, and the restart branches (ErrRestarted / ErrConnectionTimeoutClosed
// / ErrEndOfFragment / ...) make that assumption false: the task leaves the
// approved download and comes back to the gate.
type traffic struct {
	ID string

	manager *downloadManager

	// position is the task's index in the queue, or -1 when it has no position
	// yet (newly created, or re-requested after it completed).
	position atomic.Int64
	allowed  atomic.Bool

	// wake carries a token whenever the permit changes. Capacity 1 with a non
	// blocking send: a token that arrives while one is already pending is
	// redundant, because the waiter re-reads the state after every wakeup.
	wake chan struct{}

	// CancelCh is closed by Cancel, and is the only irreversible signal here.
	CancelCh chan struct{}
}

func (p *traffic) Cancel() {
	close(p.CancelCh)
}

// NotifyPermit publishes the current position and permission, then wakes a
// waiter if there is one. The stores happen before the wakeup so that a waiter
// released by the token always observes the value that triggered it.
func (p *traffic) NotifyPermit(position int, allowed bool) {
	p.position.Store(int64(position))
	p.allowed.Store(allowed)

	select {
	case p.wake <- struct{}{}:
	default:
	}
}

func (p *traffic) WaitPending(beforeWait func(eta time.Time)) error {
	if p.manager == nil {
		return nil
	}

	waited := false

	for !p.allowed.Load() {
		if !waited {
			waited = true
			beforeWait(p.manager.EstimatedToResume(p.ID))
			logger.InfoLnf("Paused download task %s, because its position is %d", p.ID, p.position.Load())
		}

		select {
		case <-p.CancelCh:
			return task.ErrCanceled
		case <-p.wake:
			// The permit may have changed; re-read it at the top of the loop.
		}
	}

	if waited {
		logger.InfoLn("Continue download task", p.ID)
	}

	return nil
}

func newTraffic(manager *downloadManager, id string) *traffic {
	return &traffic{
		ID: id,

		manager: manager,

		wake:     make(chan struct{}, 1),
		CancelCh: make(chan struct{}),
	}
}

type ManagedTask struct {
	task.Task

	manager *downloadManager
}

func newManagedTask(manager *downloadManager, id string, remote task.RemoteProvider, local task.LocalProvider) *ManagedTask {
	return &ManagedTask{
		Task: task.ConstructTask(id, newTraffic(manager, id), remote, local),

		manager: manager,
	}
}

// Download is wrapped with throttle handling
func (t *ManagedTask) Download() {
	t.Task.Download()

	var throttleErr *utils.ThrottledError
	if errors.As(t.Err(), &throttleErr) {
		t.manager.scheduler.Throttle(throttleErr.RetryAfter)
	}
}

// Cancel is redirected to manager canceling
func (t *ManagedTask) Cancel() {
	t.manager.CancelDownload(t.ID)
}

var tempDelay = time.Second * 3

func (t *ManagedTask) Retry() time.Time {
	go func() {
		<-time.After(tempDelay)
		t.Download()
		t.manager.UpdatePriorities()
	}()
	return time.Now().Add(tempDelay)
}
