package song

import (
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var retryScheduler = utils.NewScheduler(time.Second*3, time.Second)

func (sm *StateMachine) planNextRetry(slowDown bool) {
	sm.CoolingDown = true
	if slowDown {
		retryScheduler.SlowDown()
	}
	go func() {
		<-time.After(retryScheduler.Reserve())
		sm.CoolingDown = false
		sm.StartDownload()
	}()
}
