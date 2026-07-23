package cache_manager

import (
	"errors"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
)

type LocalVideoInfo struct {
	Meta *persistence.CacheMeta

	id     string
	Active bool
}

func (i LocalVideoInfo) ID() string {
	return i.id
}

const localVideosPageSize = 20

func (s *Service) ListLocalVideos(offset int, sortColumn string, preservedOnly bool) []LocalVideoInfo {
	videos, err := persistence.ListCacheMeta("video", sortColumn, offset, localVideosPageSize, preservedOnly)
	if err != nil {
		s.logger.ErrorLn("Failed to list local videos:", err)
	}

	return lo.Map(videos, func(item *persistence.CacheMeta, _ int) LocalVideoInfo {
		return LocalVideoInfo{
			Meta:   item,
			id:     item.EntityID,
			Active: s.videoMap.IsActive(item.EntityID),
		}
	})
}

func (s *Service) GetLocalVideoInfo(id string) LocalVideoInfo {
	meta, ok := persistence.GetCacheMeta(id, "video")
	if ok {
		return LocalVideoInfo{
			Meta:   meta,
			id:     id,
			Active: s.videoMap.IsActive(id),
		}
	}

	return LocalVideoInfo{
		id: id,
	}
}

func (s *Service) RemoteCache(fileType, id string) error {
	switch fileType {
	case "video":
		return s.videoMap.Remove(id)
	default:
		return errors.New("file type not supported")
	}
}
