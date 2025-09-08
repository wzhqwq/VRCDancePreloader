package song

import (
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var retryScheduler = utils.NewScheduler(time.Second*3, time.Second)

func (sm *StateMachine) planNextRetry() {
	sm.CoolingDown = true
	go func() {
		<-time.After(retryScheduler.Reserve())
		sm.CoolingDown = false
		sm.StartDownload()
	}()
}
