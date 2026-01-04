package download

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func (dm *downloadManager) Prioritize(ids ...string) {
	dm.Lock()
	defer dm.unlockAndUpdate()

	if utils.IsPrefixOf(dm.queue, ids) {
		return
	}

	dm.queue = append(
		ids,
		lo.Reject(dm.queue, func(id string, _ int) bool {
			return lo.Contains(ids, id)
		})...,
	)
}

func (dm *downloadManager) QueueTransaction() func() {
	dm.inTransaction = true
	return func() {
		dm.inTransaction = false
		dm.UpdatePriorities()
	}
}

func (dm *downloadManager) UpdatePriorities() {
	dm.Lock()
	defer dm.Unlock()
	if len(dm.queue) == 0 || dm.inTransaction {
		return
	}

	dm.queue = lo.Filter(dm.queue, func(id string, _ int) bool {
		task, ok := dm.tasks[id]
		return ok && !task.Done
	})
	logger.InfoLn("Download queue:", dm.queue)
	for i, id := range dm.queue {
		task := dm.tasks[id]
		if task != nil {
			task.sendPriority(i)
		}
	}

	dm.em.NotifySubscribers(QueueChange)
}

func (dm *downloadManager) CanDownload(priority int) bool {
	return priority >= 0 && priority < dm.maxParallel
}

func (dm *downloadManager) allDownloadingEta() []int64 {
	return lo.FilterMap(dm.queue, func(id string, _ int) (int64, bool) {
		if task, ok := dm.tasks[id]; ok && task.eta != nil {
			eta, valid := task.eta.QueryEta()
			if valid {
				return eta.Unix(), true
			}
		}
		return 0, false
	})
}

func (dm *downloadManager) EstimatedToResume(id string) (time.Time, bool) {
	dm.Lock()
	defer dm.Unlock()

	knownEta := dm.allDownloadingEta()
	slices.Sort(knownEta)

	inQueue := lo.FilterMap(dm.queue, func(id string, _ int) (string, bool) {
		if task, ok := dm.tasks[id]; ok && task.Pending {
			return id, true
		}
		return "", false
	})

	order := lo.IndexOf(inQueue, id)
	if order != -1 && order < len(knownEta) {
		return time.Unix(knownEta[order], 0), true
	}

	return time.Time{}, false
}

const hangingConnectionTimeout = time.Second * 30

// BlockIfPending keeps blocked until this task is able to continue or is canceled (returning error)
func (t *Task) BlockIfPending() error {
	var priority int
	select {
	case priority = <-t.PriorityCh:
		// continue checking
	default:
		// This means the priority have not been changed since the previous pending check
		// which approved the downloading task to continue
		return nil
	}

	// scooped timeout
	ctx, cancel := context.WithTimeout(context.Background(), hangingConnectionTimeout)
	defer cancel()

	for {
		if t.manager.CanDownload(priority) {
			if t.Pending {
				t.Pending = false
				t.notifyStateChange()
				logger.InfoLn("Continue download task", t.ID)
			}
			return nil
		}

		if !t.Pending {
			t.Pending = true
			t.notifyStateChange()
			logger.InfoLnf("Paused download task %s, because its priority is %d", t.ID, priority)
			// clear ETA counter
			t.resetEta()

			if t.connected {
				eta, valid := t.manager.EstimatedToResume(t.ID)
				if valid && eta.Sub(time.Now()) > hangingConnectionTimeout {
					// close download stream if it won't resume in 30s
					t.restart()
				} else {
					// close download stream after 30s
					go func() {
						<-ctx.Done()
						if errors.Is(ctx.Err(), context.DeadlineExceeded) {
							t.restart()
						}
					}()
				}
			}
		}
		select {
		case <-t.CancelCh:
			return ErrCanceled
		case <-t.RestartCh:
			return ErrRestarted
		case priority = <-t.PriorityCh:
			// continue checking
		}
	}
}
