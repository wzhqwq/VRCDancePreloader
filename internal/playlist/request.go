package playlist

import (
	"io"

	"github.com/wzhqwq/PyPyDancePreloader/internal/song"
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
)

func (pl *PlayList) findPyPySong(id int) *song.PreloadedSong {
	pl.Lock()
	defer pl.Unlock()
	for _, item := range pl.Items {
		if item.MatchWithPyPyId(id) {
			return item
		}
	}
	item := song.CreatePreloadedPyPySong(id)
	// TODO: add to temporary list
	return item
}

func (pl *PlayList) findCustomSong(url string) *song.PreloadedSong {
	pl.Lock()
	defer pl.Unlock()
	for _, item := range pl.Items {
		if item.MatchWithCustomUrl(url) {
			return item
		}
	}
	item := song.CreatePreloadedCustomSong("", url)
	// TODO: add to temporary list
	return item
}

func RequestPyPySong(id int) (io.ReadSeekCloser, error) {
	item := currentPlaylist.findPyPySong(id)
	return item.GetSongRSSync()
}

func RequestYoutubeSong(id string) (io.ReadSeekCloser, error) {
	item := currentPlaylist.findCustomSong(utils.GetStandardYoutubeURL(id))
	return item.GetSongRSSync()
}
