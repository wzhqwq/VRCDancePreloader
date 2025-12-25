package download

import (
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
	return priority < dm.maxParallel
}

// BlockIfPending keeps blocked until this task is able to continue or is canceled (returning false)
func (t *Task) BlockIfPending() bool {
	var priority int
	select {
	case priority = <-t.PriorityCh:
		// continue checking
	default:
		// This means the priority have not been changed since the previous pending check
		// which approved the downloading task to continue
		return true
	}

	for {
		if t.manager.CanDownload(priority) {
			if t.Pending {
				t.Pending = false
				t.notifyStateChange()
				logger.InfoLn("Continue download task", t.ID)
			}
			return true
		}

		if !t.Pending {
			t.Pending = true
			t.notifyStateChange()
			logger.InfoLnf("Paused download task %s, because its priority is %d", t.ID, priority)
			// clear ETA counter
			t.resetEta()
		}
		select {
		case <-t.CancelCh:
			return false
		case priority = <-t.PriorityCh:
			// continue checking
		}
	}
}
