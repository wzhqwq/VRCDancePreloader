package preloader

import (
	"time"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/playlist"
)

func (s *Service) loop(stopCh <-chan struct{}) {
	playlistCh := playlist.SubscribeNewListEvent()
	defer playlistCh.Close()
	pl := playlist.GetCurrentPlaylist()

	s.plMu.Lock()
	s.pl = pl
	s.plMu.Unlock()

	itemsChangeCh := pl.SubscribeChangeEvent()
	defer itemsChangeCh.Close()
	s.preload()

	for {
		select {
		case <-stopCh:
			return
		case change := <-itemsChangeCh.Channel:
			if change == playlist.ItemsChange {
				s.preload()
				s.healthCheck()
			}
		case <-s.cfgCh:
			s.preload()
		case pl = <-playlistCh.Channel:
			s.plMu.Lock()
			s.pl = pl
			s.plMu.Unlock()

			itemsChangeCh.Close()
			itemsChangeCh = pl.SubscribeChangeEvent()
			s.preload()
		case <-time.After(time.Minute):
			s.healthCheck()
		}
	}
}

func (s *Service) preload() {
	done := s.downloaderSvc.QueueTransaction()
	defer done()

	items := lo.Slice(s.pl.GetItemsSnapshot(), 0, s.Cfg.MaxPreload+1)
	for _, item := range items {
		s.makeSureDownloading(item)
	}
	// force prioritize currently playing video and the next one
	s.downloaderSvc.Prioritize(
		lo.FilterMap(
			lo.Slice(items, 0, 2),
			func(item *song.StatefulSong, index int) (string, bool) {
				if !item.StateMachine().IsDownloadLoopStarted() {
					return "", false
				}
				return item.SongId(), true
			},
		)...,
	)
}

func (s *Service) healthCheck() {
	items := lo.Slice(s.pl.GetItemsSnapshot(), 0, s.Cfg.MaxPreload+1)
	if len(items) < 2 {
		return
	}

	eta := time.Second
	if items[0].TimePassed != 0 {
		eta += max(0, items[0].EstDuration()-items[0].TimePassed)
	}

	for _, item := range items[1:] {
		duration := item.EstDuration()
		if item.StateMachine().DownloadStatus == song.Downloading {
			s.downloaderSvc.UpdateRequestEta(item.SongId(), time.Now().Add(eta), duration)
		}
		eta += duration + time.Second
	}
}
