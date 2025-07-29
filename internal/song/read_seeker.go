package song

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"io"
)

func (ps *PreloadedSong) GetSongRSSync() (io.ReadSeekCloser, error) {
	err := ps.sm.DownloadInstantly(true)
	if err != nil {
		return nil, err
	}
	entry, err := cache.OpenCacheEntry(ps.GetSongId())
	if err != nil {
		return nil, err
	}
	return entry.GetReadSeekCloser(), nil
}

func (ps *PreloadedSong) GetSongRSAsync() (io.ReadSeekCloser, error) {
	err := ps.sm.DownloadInstantly(false)
	if err != nil {
		return nil, err
	}
	entry, err := cache.OpenCacheEntry(ps.GetSongId())
	if err != nil {
		return nil, err
	}
	return entry.GetReadSeekCloser(), nil
}
