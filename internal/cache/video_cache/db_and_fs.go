package video_cache

import (
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache/cache_fs"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func (cm *CacheMap) Remove(id string) error {
	cm.Lock()
	defer cm.Unlock()

	return cm.remove(id)
}

func (cm *CacheMap) remove(id string) error {
	if cm.isActive(id) {
		return nil
	}

	persistence.RemoveCacheMetaIfExists(id, "video")

	return cm.removeInFs(id)
}

func (cm *CacheMap) removeInFs(id string) error {
	err := cache_fs.DeleteWithoutExt("video$" + id)
	if err != nil {
		return err
	}

	err = cache_fs.DeleteWithoutExt("etag$" + id)
	if err != nil {
		return err
	}

	return nil
}

func (cm *CacheMap) cleanUp() {
	cm.Lock()
	defer cm.Unlock()

	tx, err := persistence.BeginCacheCleanupTx()
	if err != nil {
		logger.ErrorLn("Failed to launch cleanup:", err)
		return
	}
	// Commit instead of rolling back because files marked as removed are already removed from disk, it cannot be atomic
	defer tx.Finish()

	totalSize, err := tx.Summarize("video")
	if err != nil {
		logger.ErrorLn("Failed to calculate cache size:", err)
		return
	}

	size := totalSize
	if size <= maxSize {
		return
	}

	logger.InfoLn("Cleaning up cache ...")
	for size > maxSize {
		records, err := tx.ListCandidates("video")
		if err != nil {
			logger.ErrorLn("Failed to list cache meta:", err)
			return
		}

		for _, record := range records {
			if cm.isActive(record.ID) {
				continue
			}
			if keepFavorites && persistence.IsFavorite(record.ID) {
				continue
			}

			err := cm.removeInFs(record.ID)
			if err != nil {
				logger.ErrorLn("Failed to remove cache:", err)
				return
			}

			tx.MarkRemoved(record.ID, "video")

			size -= record.Size
			if size <= maxSize {
				break
			}
		}
	}
	logger.InfoLn("Cleaned up cache,", utils.PrettyByteSize(totalSize), "->", utils.PrettyByteSize(size))
}

type LocalVideoInfo struct {
	Meta *persistence.CacheMeta

	ID     string
	Active bool
}

const localVideosPageSize = 50

func (cm *CacheMap) ListLocalVideos(page int, sortColumn string, preservedOnly bool) []LocalVideoInfo {
	videos, err := persistence.ListCacheMeta("video", sortColumn, page, localVideosPageSize, preservedOnly)
	if err != nil {
		logger.ErrorLn("Failed to list local videos:", err)
	}

	return lo.Map(videos, func(item *persistence.CacheMeta, _ int) LocalVideoInfo {
		return LocalVideoInfo{
			Meta:   item,
			ID:     item.EntityID,
			Active: cm.isActive(item.EntityID),
		}
	})
}

func (cm *CacheMap) GetLocalVideoInfo(id string) LocalVideoInfo {
	meta, ok := persistence.GetCacheMeta(id, "video")
	if ok {
		return LocalVideoInfo{
			Meta:   meta,
			ID:     id,
			Active: cm.IsActive(id),
		}
	}

	return LocalVideoInfo{
		ID: id,
	}
}
