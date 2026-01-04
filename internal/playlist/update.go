package playlist

import (
	"time"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
)

func (pl *PlayList) CriticalUpdate() {
	select {
	case pl.criticalUpdateCh <- struct{}{}:
	default:
	}
}

func (pl *PlayList) loop() {
	listCh := raw_song.SubscribeSongListChange()
	defer listCh.Close()

	pl.preload()
	for {
		select {
		case <-pl.stopCh:
			return
		case <-pl.criticalUpdateCh:
			pl.preload()
			pl.healthCheck()
		case <-listCh.Channel:
			pl.refresh()
		case <-time.After(time.Minute):
			pl.healthCheck()
		}
	}
}

func (pl *PlayList) preload() {
	done := download.QueueTransaction()
	defer done()

	items := lo.Slice(pl.GetItemsSnapshot(), 0, pl.maxPreload+1)
	for _, item := range items {
		item.PreloadSong()
	}
	// force prioritize currently playing video and the next one
	download.Prioritize(
		lo.FilterMap(
			lo.Slice(items, 0, 2),
			func(item *song.PreloadedSong, index int) (string, bool) {
				if !item.InDownloadQueue() {
					return "", false
				}
				return item.GetSongId(), true
			},
		)...,
	)
}

func (pl *PlayList) healthCheck() {
	items := lo.Slice(pl.GetItemsSnapshot(), 0, pl.maxPreload+1)
	if len(items) < 2 {
		return
	}

	eta := time.Second
	if items[0].TimePassed != 0 && items[0].Duration != 0 {
		eta += items[0].Duration - items[0].TimePassed
	}

	for _, item := range items[1:] {
		item.UpdateStartPlayingEta(eta)
		eta += item.Duration + time.Second
	}
}

func (pl *PlayList) refresh() {
	items := lo.Slice(pl.GetItemsSnapshot(), 0, pl.maxPreload+1)
	for _, item := range items {
		item.UpdateSong()
	}
}

func (pl *PlayList) UpdateBySongID(ID string) {
	items := lo.Slice(pl.GetItemsSnapshot(), 0, pl.maxPreload+1)
	for _, item := range items {
		if item.GetSongId() == ID {
			item.UpdateSong()
		}
	}
}
