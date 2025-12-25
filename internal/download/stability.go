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

const forceRestartThreshold = time.Minute
const tooSlowThreshold = time.Minute * 3
const restartMinInterval = time.Second * 10
const acceptableEta = time.Second * 10
const estimationBias = time.Second * 30

func (dm *downloadManager) RestartTaskIfTooSlow(id string, timeRemaining time.Duration) {
	dm.Lock()
	defer dm.Unlock()

	task, exists := dm.tasks[id]
	if !exists {
		return
	}

	if task.Done || task.Cooling || task.Pending || task.eta == nil {
		return
	}

	passed := task.eta.Passed()
	if passed < restartMinInterval {
		return
	}
	if passed > tooSlowThreshold {
		logger.InfoLn("Restart task", id, "because it already spent too much time")
		task.restart()
		return
	}

	eta, valid := task.eta.QueryEta()
	if valid {
		if eta > acceptableEta && eta+estimationBias > timeRemaining {
			logger.InfoLn("Restart task", id, "because it cannot be done before the song plays")
			task.restart()
		}
		return
	}

	// invalid eta
	if timeRemaining < forceRestartThreshold {
		logger.InfoLn("Restart task", id, "because we cannot guess when it will be done")
		task.restart()
	}
}
