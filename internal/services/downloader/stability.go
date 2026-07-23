package downloader

import (
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader/task"
)

const tooSlowThreshold = time.Minute * 3
const restartMinInterval = time.Second * 30
const acceptableEta = time.Second * 10
const estimationBias = time.Second * 30

func (dm *downloadManager) restartIfNeeded(task *ManagedTask, endMoment time.Time) {
	// Do not try restart if throttle is applied, otherwise we will be blocked again
	if dm.scheduler.ThrottleApplied() {
		return
	}

	passed := task.Eta.Passed()
	if passed < restartMinInterval {
		return
	}
	if passed > tooSlowThreshold {
		logger.InfoLn("Restart task", task.ID, "because it already spent too much time")
		task.Restart()
		return
	}

	eta, valid := task.Eta.QueryEta()
	if valid {
		// will be done in 10 seconds
		if eta.Sub(time.Now()) < acceptableEta {
			return
		}
		if endMoment.Sub(eta) < estimationBias {
			logger.InfoLn("Restart task", task.ID, "because it cannot be done before the song ends")
			task.Restart()
		}
	}
}

func (dm *downloadManager) UpdateRequestEta(id string, eta time.Time, duration time.Duration) {
	dm.Lock()
	defer dm.Unlock()

	t, exists := dm.tasks[id]
	if !exists {
		return
	}

	// The task must be downloading
	if t.State == task.TaskDownloading || t.State == task.TaskRequested || t.State == task.TaskResolving {
		dm.restartIfNeeded(t, eta.Add(duration))
	}
}
