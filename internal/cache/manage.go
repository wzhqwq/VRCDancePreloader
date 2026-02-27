package cache

import (
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache/cache_fs"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache/entry"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache/video_cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var cacheMap = video_cache.NewCacheMap()

var managerLogger = utils.NewLogger("Cache Manager")

func SetupCache(path string) {
	cache_fs.Setup(path)
	go cacheMap.EventLoop()
	if persistence.IsScheduleDueReached("cache_sync_fs", time.Hour*24*7) {
		managerLogger.InfoLn("Doing scheduled filesystem-database synchronization")
		err := SyncWithFS()
		if err != nil {
			managerLogger.ErrorLn("Failed to sync with filesystem:", err)
		} else {
			persistence.UpdateSchedule("cache_sync_fs")
			persistence.WalCheckpoint()
			managerLogger.InfoLn("Synchronization done. We will do it again next week")
		}
	}
}

func StopCache() {
	cacheMap.Stop()
}

func CleanUpCache() {
	cacheMap.CleanUp()
}

func GetLocalCacheInfos(page int, sortColumn string, preservedOnly bool) []video_cache.LocalVideoInfo {
	return cacheMap.ListLocalVideos(page, sortColumn, preservedOnly)
}

func GetLocalCacheInfo(id string) video_cache.LocalVideoInfo {
	return cacheMap.GetLocalVideoInfo(id)
}

func RemoveLocalCacheById(id string) error {
	return cacheMap.Remove(id)
}

func OpenCacheEntry(id string, logger utils.LoggerImpl) (entry.Entry, error) {
	logger.InfoLn("Open cache entry:", id)
	return cacheMap.Open(id)
}

func ReleaseCacheEntry(id string, logger utils.LoggerImpl) {
	logger.InfoLn("Release cache entry:", id)
	cacheMap.Release(id)
	go func() {
		<-time.After(time.Second)
		if cacheMap.CloseIfInactive(id) {
			managerLogger.InfoLn("Closed cache entry:", id)
		}
	}()
}
