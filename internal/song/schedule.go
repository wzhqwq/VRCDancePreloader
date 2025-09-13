package song

import (
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var downloadScheduler = utils.NewScheduler(time.Second*3, time.Second)

func (sm *StateMachine) planNextRetry(slowDown bool) {
	sm.CoolingDown = true
	if slowDown {
		downloadScheduler.SlowDown()
		go func() {
			<-time.After(time.Second * 30)
			downloadScheduler.Resume()
		}()
	}
	go func() {
		<-time.After(downloadScheduler.ReserveWithDelay())
		sm.CoolingDown = false
		sm.StartDownload()
	}()
}

func (sm *StateMachine) reserveForPreload() {
	<-time.After(downloadScheduler.Reserve())
}
