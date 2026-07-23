package cache_manager

import (
	"errors"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_fs"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager/cache_map"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager/entry"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var managerLogger = utils.NewLogger("Cache Manager")

type Service struct {
	service.BaseService[Config]

	videoMap   cache_map.CacheMap
	catalogMap cache_map.CacheMap

	logger utils.LoggerImpl

	cacheFs *cache_fs.CacheFS
}

func New(cfg Config) *Service {
	s := &Service{
		BaseService: service.ConstructBaseService(cfg),

		logger: utils.NewLogger("Cache Manager"),
	}

	s.SetControl("Cache Manager", s)

	return s
}

func (s *Service) ServiceStart() error {
	var err error
	s.cacheFs, err = cache_fs.New(s.Cfg.Path)
	if err != nil {
		return err
	}

	if persistence.IsScheduleDueReached("cache_sync_fs", time.Hour*24*7) {
		managerLogger.InfoLn("Doing scheduled filesystem-database synchronization")
		err = s.SyncWithFS()
		if err != nil {
			managerLogger.ErrorLn("Failed to sync with filesystem:", err)
		} else {
			persistence.UpdateSchedule("cache_sync_fs")
			persistence.WalCheckpoint()
			managerLogger.InfoLn("Synchronization done. We will do it again next week")
		}
	}

	s.videoMap = cache_map.NewBoundedCacheMap(
		"Video Cache",
		int64(s.Cfg.MaxVideoCache)*1024*1024,
		s.Cfg.KeepFavorites,
		s.newVideoEntry,
		s.removeVideoInCache,
	)
	s.catalogMap = cache_map.NewCacheMap(
		"Catalog Cache",
		s.newCatalogEntry,
		s.removeCatalogInCache,
	)

	return nil
}

func (s *Service) ServiceStop() error {
	s.videoMap.CloseAll()
	s.videoMap = nil
	return nil
}

func (s *Service) Enabled() bool {
	return true
}

func (s *Service) UpdateConfig(cfg Config, field string) error {
	if field == "" {
		if cfg.KeepFavorites != s.Cfg.KeepFavorites || cfg.MaxVideoCache != s.Cfg.MaxVideoCache {
			maxSize := int64(s.Cfg.MaxVideoCache) * 1024 * 1024
			s.videoMap.SetCleanup(cache_map.CleanupConfig{MaxSize: maxSize, KeepFavorites: cfg.KeepFavorites})
		}
	}

	switch field {
	case "path":
		// TODO migrate
	case "max-video-cache", "keep-favorites":
		maxSize := int64(s.Cfg.MaxVideoCache) * 1024 * 1024
		s.videoMap.SetCleanup(cache_map.CleanupConfig{MaxSize: maxSize, KeepFavorites: cfg.KeepFavorites})

		// VideoFileFormat and ForceExpiration don't need immediate update
	}

	s.Cfg = cfg
	return nil
}

func (s *Service) CacheSize() int64 {
	return int64(s.Cfg.MaxVideoCache) * 1024 * 1024
}

func (s *Service) newVideoEntry(id string) (entry.CDNEntry, error) {
	return entry.NewVideoEntry(id, s.Cfg.VideoFileFormat, s.cacheFs)
}

func (s *Service) newCatalogEntry(id string) (entry.CDNEntry, error) {
	return entry.NewCatalogEntry(id, s.cacheFs), nil
}

func (s *Service) removeVideoInCache(id string) error {
	persistence.RemoveCacheMetaIfExists(id, "video")

	err := s.cacheFs.DeleteWithoutExt("video$" + id)
	if err != nil {
		return err
	}

	err = s.cacheFs.DeleteWithoutExt("etag$" + id)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) removeCatalogInCache(id string) error {
	persistence.RemoveCacheMetaIfExists(id, "catalog")
	err := s.cacheFs.DeleteWithoutExt("catalog$" + id)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) CreateSession(cacheType string) (types.CDNFileSession, error) {
	switch cacheType {
	case "video":
		return newCdnFileSession(s.videoMap, s), nil
	case "catalog":
		return newCdnFileSession(s.catalogMap, s), nil
	default:
		return nil, errors.New("invalid cache type")
	}
}
