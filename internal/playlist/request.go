package playlist

import (
	"io"

	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var asyncDownload = true

func SetAsyncDownload(async bool) {
	asyncDownload = async
}

func (pl *PlayList) FindPyPySong(id int) *song.PreloadedSong {
	items := pl.GetItemsSnapshot()
	for _, item := range items {
		if item.MatchWithPyPyId(id) {
			return item
		}
	}
	item := song.CreatePreloadedPyPySong(id)
	// TODO: add to temporary list
	return item
}

func (pl *PlayList) FindWannaSong(id int) *song.PreloadedSong {
	items := pl.GetItemsSnapshot()
	for _, item := range items {
		if item.MatchWithWannaId(id) {
			return item
		}
	}
	item := song.CreatePreloadedWannaSong(id)
	// TODO: add to temporary list
	return item
}

func (pl *PlayList) FindCustomSong(url string) *song.PreloadedSong {
	items := pl.GetItemsSnapshot()
	for _, item := range items {
		if item.MatchWithCustomUrl(url) {
			return item
		}
	}
	item := song.CreatePreloadedCustomSong(url)
	// TODO: add to temporary list
	return item
}

func RequestPyPySong(id int) (io.ReadSeekCloser, error) {
	item := currentPlaylist.FindPyPySong(id)
	if asyncDownload {
		return item.GetSongRSAsync()
	} else {
		return item.GetSongRSSync()
	}
}

func RequestWannaSong(id int) (io.ReadSeekCloser, error) {
	item := currentPlaylist.FindWannaSong(id)
	if asyncDownload {
		return item.GetSongRSAsync()
	} else {
		return item.GetSongRSSync()
	}
}

func RequestBiliSong(bvID string) (io.ReadSeekCloser, error) {
	item := currentPlaylist.FindCustomSong(utils.GetStandardBiliURL(bvID))
	if asyncDownload {
		return item.GetSongRSAsync()
	} else {
		return item.GetSongRSSync()
	}
}

// TODO

func RequestYoutubeSong(id string) (io.ReadSeekCloser, error) {
	item := currentPlaylist.FindCustomSong(utils.GetStandardYoutubeURL(id))
	return item.GetSongRSSync()
}
