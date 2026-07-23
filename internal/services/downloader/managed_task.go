package downloader

import (
	"errors"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader/task"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type traffic struct {
	ID string

	manager *downloadManager

	PriorityCh chan int
	CancelCh   chan struct{}
}

func (p *traffic) Cancel() {
	close(p.CancelCh)
}

func (p *traffic) WaitPending(beforeWait func(eta time.Time)) error {
	if p.manager == nil {
		return nil
	}

	var priority int
	select {
	case priority = <-p.PriorityCh:
		// continue checking
	default:
		// This means the priority have not been changed since the previous pending check
		// which approved the downloading task to continue
		return nil
	}

	waited := false

	for {
		if p.manager.CanDownload(priority) {
			if waited {
				logger.InfoLn("Continue download task", p.ID)
			}
			return nil
		}

		if !waited {
			waited = true
			beforeWait(p.manager.EstimatedToResume(p.ID))
			logger.InfoLnf("Paused download task %s, because its priority is %d", p.ID, priority)
		}

		select {
		case <-p.CancelCh:
			return task.ErrCanceled
		case priority = <-p.PriorityCh:
			// continue checking
		}
	}
}

func (p *traffic) WaitScheduled(beforeWait func()) error {
	delay := p.manager.scheduler.Reserve()
	if delay > 0 {
		beforeWait()

		select {
		case <-p.CancelCh:
			return task.ErrCanceled
		case <-time.After(delay):
		}
	}

	return nil
}

func (p *traffic) sendPriority(priority int) {
	// flush first
	select {
	case <-p.PriorityCh:
	default:
	}
	p.PriorityCh <- priority
}

func newTraffic(manager *downloadManager, id string) *traffic {
	return &traffic{
		ID:         id,
		manager:    manager,
		PriorityCh: make(chan int, 1),
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

func (t *ManagedTask) Download() {
	t.Task.Download()

	var throttleErr *utils.ThrottledError
	if errors.As(t.Error, &throttleErr) {
		t.manager.scheduler.Throttle(throttleErr.RetryAfter)
	}
}

var tempDelay = time.Second * 3

func (t *ManagedTask) Retry() {
	go func() {
		<-time.After(tempDelay)
		t.Download()
		t.manager.UpdatePriorities()
	}()
}
