package download

import (
	"time"
)

func (dm *downloadManager) slowDown() {
	dm.scheduler.Throttle()
	go func() {
		// PyPyDance seems to ban you for 3 minute if you have been requesting so fast
		<-time.After(time.Minute * 3)
		dm.scheduler.ReleaseOneThrottle()
	}()
}

func (dm *downloadManager) getDelay(retryDelay bool) (delay time.Duration) {
	// delay or cool down before we start downloading
	if retryDelay {
		delay = dm.scheduler.ReserveWithDelay()
	} else {
		delay = dm.scheduler.Reserve()
	}
	return
}

const tooSlowThreshold = time.Minute * 3
const restartMinInterval = time.Second * 30
const acceptableEta = time.Second * 10
const estimationBias = time.Second * 30

func (dm *downloadManager) restartIfNeeded(task *Task, endMoment time.Time) {
	// Do not try restart if throttle is applied, otherwise we will be blocked again
	if dm.scheduler.ThrottleApplied() {
		return
	}

	passed := task.eta.Passed()
	if passed < restartMinInterval {
		return
	}
	if passed > tooSlowThreshold {
		logger.InfoLn("Restart task", task.ID, "because it already spent too much time")
		task.restart()
		return
	}

	eta, valid := task.eta.QueryEta()
	if valid {
		// will be done in 10 seconds
		if eta.Sub(time.Now()) < acceptableEta {
			return
		}
		if endMoment.Sub(eta) < estimationBias {
			logger.InfoLn("Restart task", task.ID, "because it cannot be done before the song ends")
			task.restart()
		}
	}
}

func (dm *downloadManager) UpdateRequestEta(id string, eta time.Time, duration time.Duration) {
	dm.Lock()
	defer dm.Unlock()

	task, exists := dm.tasks[id]
	if !exists {
		return
	}

	// The task must be downloading
	if task.Done || task.Cooling || task.Pending {
		return
	}

	dm.restartIfNeeded(task, eta.Add(duration))
}
