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

func (dm *downloadManager) UpdatePriorities() {
	dm.Lock()
	defer dm.Unlock()

	if len(dm.queue) == 0 {
		return
	}

	// A frozen queue is mid-assembly: the order below is not the final one, so
	// neither the permits nor the queue event may be published from it. The
	// freeze's own thaw publishes, once, over the order the caller intended.
	if dm.batch > 0 {
		return
	}

	dm.queue = lo.Filter(dm.queue, func(id string, _ int) bool {
		t, ok := dm.tasks[id]
		return ok && t.State != task.TaskCompleted
	})
	dm.queueLogger.InfoLn("tasks:", dm.queue)

	dm.publishPermitsLocked()

	dm.em.NotifySubscribers(QueueChange)
}

// publishPermitsLocked publishes the queue position and the download permission
// to every queued task.
//
// The permission is derived from the position here and delivered as state, so
// there is exactly one place that decides "who may download": the queue order
// and maxParallel. Tasks never compare anything themselves.
//
// This is also the single place that *grants* a permit, so it is where the
// freeze is enforced: while a caller is still assembling the order, a grant
// would be a grant for a position the task is about to lose, and a granted task
// only re-reads its permit when its attempt restarts.
//
// Caller must hold dm.Lock().
func (dm *downloadManager) publishPermitsLocked() {
	if dm.batch > 0 {
		return
	}

	for i, id := range dm.queue {
		if t := dm.tasks[id]; t != nil {
			t.Traffic.NotifyPermit(i, i < dm.maxParallel)
		}
	}
}

func (dm *downloadManager) allDownloadingEta() []int64 {
	return lo.FilterMap(dm.queue, func(id string, _ int) (int64, bool) {
		if t, ok := dm.tasks[id]; ok {
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
