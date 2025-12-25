package download

import (
	"strings"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var managers = make(map[string]*downloadManager)

func InitDownloadManager(maxParallel int) {
	managers["pypy"] = newDownloadManager(maxParallel, time.Second*5)
	managers["default"] = newDownloadManager(maxParallel, time.Second*2)
}

func findManager(id string) *downloadManager {
	if strings.HasPrefix(id, "pypy") {
		return managers["pypy"]
	}
	return managers["default"]
}

func SubscribeCoolDownInterval(name string) *utils.EventSubscriber[time.Duration] {
	dm, ok := managers[name]
	if !ok {
		return nil
	}
	return dm.scheduler.SubscribeIntervalEvent()
}

// For future use

func ListManagers() []string {
	return []string{
		"pypy",
		"default",
	}
}

func SubscribeManager(name string) *utils.EventSubscriber[ManagerChangeType] {
	dm, ok := managers[name]
	if !ok {
		return nil
	}
	return dm.em.SubscribeEvent()
}

func GetQueue(name string) []*Task {
	dm, ok := managers[name]
	if !ok {
		return nil
	}
	return dm.GetQueueSnapshot()
}
