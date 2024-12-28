package song

import (
	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
	"io"
)

func (ps *PreloadedSong) GetSongRSSync() (io.ReadSeekCloser, error) {
	err := ps.sm.WaitForCompleteSong()
	if err != nil {
		return nil, err
	}
	entry, err := cache.OpenCacheEntry(ps.GetId())
	if err != nil {
		return nil, err
	}
	return entry.GetReadSeekCloser(), nil
}

func (ps *PreloadedSong) GetSongRSAsync() (io.ReadSeekCloser, error) {
	err := ps.sm.WaitForSong()
	if err != nil {
		return nil, err
	}
	entry, err := cache.OpenCacheEntry(ps.GetId())
	if err != nil {
		return nil, err
	}
	return entry.GetReadSeekCloser(), nil
}
