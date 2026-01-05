package playlist

import (
	"context"
	"errors"
	"strconv"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
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
	return nil
}

func (pl *PlayList) FindWannaSong(id int) *song.PreloadedSong {
	items := pl.GetItemsSnapshot()
	for _, item := range items {
		if item.MatchWithWannaId(id) {
			return item
		}
	}
	return nil
}

func (pl *PlayList) FindDuDuSong(id int) *song.PreloadedSong {
	items := pl.GetItemsSnapshot()
	for _, item := range items {
		if item.MatchWithDuDuId(id) {
			return item
		}
	}
	return nil
}

func (pl *PlayList) FindCustomSong(url string) *song.PreloadedSong {
	items := pl.GetItemsSnapshot()
	for _, item := range items {
		if item.MatchWithCustomUrl(url) {
			return item
		}
	}
	return nil
}

func Request(platform, id string, ctx context.Context) (cache.Entry, error) {
	var url string

	switch platform {
	case "PyPyDance":
		numId, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}

		item := currentPlaylist.FindPyPySong(numId)
		if item == nil {
			item = song.GetTemporaryPyPySong(numId, ctx)
		}
		return item.DownloadInstantly(!asyncDownload, ctx)

	case "WannaDance":
		numId, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}

		item := currentPlaylist.FindWannaSong(numId)
		if item == nil {
			item = song.GetTemporaryWannaSong(numId, ctx)
		}
		return item.DownloadInstantly(!asyncDownload, ctx)

	case "DuDuFitDance":
		numId, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}

		item := currentPlaylist.FindDuDuSong(numId)
		if item == nil {
			item = song.GetTemporaryDuDuSong(numId, ctx)
		}
		return item.DownloadInstantly(!asyncDownload, ctx)

	case "BiliBili":
		url = utils.GetStandardBiliURL(id)
		// TODO youtube
	default:
		return nil, errors.New("invalid platform")
	}

	item := currentPlaylist.FindCustomSong(url)
	if item == nil {
		item = song.GetTemporaryCustomSong(url, ctx)
	}
	return item.DownloadInstantly(!asyncDownload, ctx)
}
