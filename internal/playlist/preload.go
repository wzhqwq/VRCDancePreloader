package playlist

import (
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
)

func (pl *PlayList) CriticalUpdate() {
	select {
	case pl.criticalUpdateCh <- struct{}{}:
	default:
	}
}

func (pl *PlayList) loop() {
	pl.preload()
	for {
		select {
		case <-pl.criticalUpdateCh:
			if pl.stopped {
				return
			}
			pl.preload()
		case <-pl.songListUpdate.Channel:
			if pl.stopped {
				return
			}
			pl.refresh()
		}
	}
}

func (pl *PlayList) preload() {
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

func (pl *PlayList) refresh() {
	items := lo.Slice(pl.GetItemsSnapshot(), 0, pl.maxPreload+1)
	for _, item := range items {
		item.UpdateSong()
	}
}
