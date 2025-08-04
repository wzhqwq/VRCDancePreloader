package playlist

import (
	"context"
	"errors"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"strconv"
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

func (pl *PlayList) FindCustomSong(url string) *song.PreloadedSong {
	items := pl.GetItemsSnapshot()
	for _, item := range items {
		if item.MatchWithCustomUrl(url) {
			return item
		}
	}
	return nil
}

func (pl *PlayList) FindPyPySongOrCreate(id int) *song.PreloadedSong {
	if item := pl.FindPyPySong(id); item != nil {
		return item
	}

	item := song.CreatePreloadedPyPySong(id)
	// TODO: add to temporary list
	return item
}

func (pl *PlayList) FindWannaSongOrCreate(id int) *song.PreloadedSong {
	if item := pl.FindWannaSong(id); item != nil {
		return item
	}

	item := song.CreatePreloadedWannaSong(id)
	// TODO: add to temporary list
	return item
}

func (pl *PlayList) FindCustomSongOrCreate(url string) *song.PreloadedSong {
	if item := pl.FindCustomSong(url); item != nil {
		return item
	}

	item := song.CreatePreloadedCustomSong(url)
	// TODO: add to temporary list
	return item
}

func request(item *song.PreloadedSong, ctx context.Context) (cache.Entry, error) {
	entry, err := item.DownloadInstantly(!asyncDownload, ctx)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func Request(platform, id string, ctx context.Context) (cache.Entry, error) {
	switch platform {
	case "PyPyDance":
		numId, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}
		return request(currentPlaylist.FindPyPySongOrCreate(numId), ctx)
	case "WannaDance":
		numId, err := strconv.Atoi(id)
		if err != nil {
			return nil, err
		}
		return request(currentPlaylist.FindWannaSongOrCreate(numId), ctx)
	case "BiliBili":
		return request(currentPlaylist.FindCustomSongOrCreate(utils.GetStandardBiliURL(id)), ctx)
		// TODO
	default:
		return nil, errors.New("invalid platform")
	}
}

func DownloadSuffix(platform, id string, start int64) {
	switch platform {
	case "PyPyDance":
		numId, err := strconv.Atoi(id)
		if err != nil {
			return
		}
		currentPlaylist.FindPyPySongOrCreate(numId).DownloadSuffix(start)
	case "WannaDance":
		numId, err := strconv.Atoi(id)
		if err != nil {
			return
		}
		currentPlaylist.FindWannaSongOrCreate(numId).DownloadSuffix(start)
	case "BiliBili":
		currentPlaylist.FindCustomSongOrCreate(utils.GetStandardBiliURL(id)).DownloadSuffix(start)
		// TODO
	}
}
