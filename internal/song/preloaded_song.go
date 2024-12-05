package song

import (
	"fmt"
	"io"

	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
	"github.com/wzhqwq/PyPyDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
)

type PreloadedSong struct {
	sm         SongStateMachine
	PyPySong   *raw_song.PyPyDanceSong
	CustomSong *raw_song.CustomSong

	// constant
	Adder string
	Order int

	// play progress states
	Duration   float64
	TimePassed float64

	// download progress states
	TotalSize      int64
	DownloadedSize int64

	// diagnostic states
	PreloadError error

	// event
	em EventManager
}

// constructors
func CreatePreloadedPyPySong(id int) *PreloadedSong {
	song, ok := raw_song.FindPyPySong(id)
	if !ok {
		return nil
	}
	ret := &PreloadedSong{
		sm: SongStateMachine{
			DownloadStatus: Initial,
		},
		PyPySong: song,
	}
	ret.sm.PreloadedSong = ret
	return ret
}

func CreatePreloadedCustomSong(title, url string) *PreloadedSong {
	ret := &PreloadedSong{
		sm: SongStateMachine{
			DownloadStatus: Initial,
		},
		CustomSong: raw_song.FindOrCreateCustomSong(title, url),
	}
	ret.sm.PreloadedSong = ret
	return ret
}

// getters
func (ps *PreloadedSong) GetOriginalUrl() string {
	if ps.PyPySong != nil && len(ps.PyPySong.OriginalURL) > 0 {
		return ps.PyPySong.OriginalURL[0]
	}
	if ps.CustomSong != nil {
		return ps.CustomSong.Url
	}
	return ""
}
func (ps *PreloadedSong) GetDownloadUrl() string {
	if ps.PyPySong != nil {
		return utils.GetPyPyVideoUrl(ps.PyPySong.ID)
	}
	if ps.CustomSong != nil {
		return ps.CustomSong.Url
	}
	return ""
}
func (ps *PreloadedSong) GetThumbnailUrl() string {
	if ps.PyPySong != nil {
		return utils.GetPyPyThumbnailUrl(ps.PyPySong.ID)
	}
	if ps.CustomSong != nil {
		return ps.CustomSong.ThumbnailUrl
	}
	return ""
}
func (ps *PreloadedSong) GetId() string {
	if ps.PyPySong != nil {
		return fmt.Sprintf("pypy_%d", ps.PyPySong.ID)
	}
	if ps.CustomSong != nil {
		return ps.CustomSong.UniqueId
	}
	return ""
}
func (ps *PreloadedSong) GetSongRSSync() (io.ReadSeekCloser, error) {
	err := ps.sm.WaitForCompleteSong()
	if err != nil {
		return nil, err
	}
	return cache.OpenCache(ps.GetId()), nil
}
