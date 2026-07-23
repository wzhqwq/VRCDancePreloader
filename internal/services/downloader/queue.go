package downloader

import (
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader/task"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func (dm *downloadManager) Prioritize(ids ...string) {
	dm.Lock()
	defer dm.unlockAndUpdate()

	if utils.IsPrefixOf(dm.queue, ids) {
		return
	}

	dm.queue = append(
		lo.Filter(ids, func(id string, _ int) bool {
			return dm.tasks[id] != nil
		}),
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
		t, ok := dm.tasks[id]
		return ok && t.State != task.TaskCompleted
	})
	dm.queueLogger.InfoLn("tasks:", dm.queue)
	for i, id := range dm.queue {
		t := dm.tasks[id]
		if t != nil {
			t.Traffic.(*traffic).sendPriority(i)
		}
	}

	dm.em.NotifySubscribers(QueueChange)
}

func (dm *downloadManager) CanDownload(priority int) bool {
	return priority >= 0 && priority < dm.maxParallel
}

func (dm *downloadManager) allDownloadingEta() []int64 {
	return lo.FilterMap(dm.queue, func(id string, _ int) (int64, bool) {
		if t, ok := dm.tasks[id]; ok && t.Eta != nil {
			eta, valid := t.Eta.QueryEta()
			if valid {
				return eta.Unix(), true
			}
		}
		return 0, false
	})
}

func (dm *downloadManager) EstimatedToResume(id string) time.Time {
	dm.Lock()
	defer dm.Unlock()

	knownEta := dm.allDownloadingEta()
	slices.Sort(knownEta)

	inQueue := lo.FilterMap(dm.queue, func(id string, _ int) (string, bool) {
		if t, ok := dm.tasks[id]; ok && t.State == task.TaskPending {
			return id, true
		}
		return "", false
	})

	order := lo.IndexOf(inQueue, id)
	if order != -1 && order < len(knownEta) {
		return time.Unix(knownEta[order], 0)
	}

	return time.Time{}
}
