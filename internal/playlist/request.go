package playlist

import (
	"io"
	"time"

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

func request(item *song.PreloadedSong) (io.ReadSeekCloser, time.Time, error) {
	entry, err := item.DownloadInstantly(!asyncDownload)
	if err != nil {
		return nil, time.Time{}, err
	}
	return entry.GetReadSeekCloser(), entry.ModTime(), nil
}

func RequestPyPySong(id int) (io.ReadSeekCloser, time.Time, error) {
	return request(currentPlaylist.FindPyPySong(id))
}
func RequestWannaSong(id int) (io.ReadSeekCloser, time.Time, error) {
	return request(currentPlaylist.FindWannaSong(id))
}
func RequestBiliSong(bvID string) (io.ReadSeekCloser, time.Time, error) {
	return request(currentPlaylist.FindCustomSong(utils.GetStandardBiliURL(bvID)))
}

// TODO

func RequestYoutubeSong(id string) (io.ReadSeekCloser, error) {
	item := currentPlaylist.FindCustomSong(utils.GetStandardYoutubeURL(id))
	return item.GetSongRSSync()
}
