package utils

import (
	"log"
	"testing"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func TestConcurrentRequests(t *testing.T) {
	scheduler := utils.NewScheduler(time.Second*3, time.Second)
	for i := 0; i < 3; i++ {
		log.Println(scheduler.ReserveWithDelay())
	}
	<-time.After(time.Second * 2)
	for i := 0; i < 3; i++ {
		log.Println(scheduler.ReserveWithDelay())
	}
}
