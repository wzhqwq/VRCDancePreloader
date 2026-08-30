package preloader

import (
	"context"
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
)

func (s *Service) Request(id string, ctx context.Context) (types.CDNResource, error) {
	s.plMu.RLock()
	defer s.plMu.RUnlock()

	if s.pl == nil {
		return nil, errors.New("no valid playlist")
	}

	item := s.pl.SearchByInternalId(id)
	if item == nil {
		item = song.GetTemporarySongByInternalId(id, ctx)
	}
	s.makeSureDownloading(item)
	s.downloaderSvc.Prioritize(item.SongId())
	return s.getResource(item, ctx)
}
