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
			switch change {
			case playlist.ItemsChange:
				s.preload()
				s.healthCheck()
			case playlist.RoomChange:
				s.preload()
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
	if !lo.Contains(s.Cfg.EnabledRooms, s.pl.RoomBrand) {
		return
	}

	items := lo.Slice(s.pl.GetItemsSnapshot(), 0, s.Cfg.MaxPreload+1)

	// Creating a task appends it to the download queue, but only the Prioritize
	// call below puts it in its real position (currently playing first). The
	// whole assembly therefore runs with the queue frozen: publishing permits in
	// between — once per creation, or from an unrelated queue change such as a
	// task finishing or a video request — would grant a task a position it is
	// about to lose, and a granted task starts downloading and does not re-read
	// its permit until its attempt restarts.
	s.downloaderSvc.WithFrozenQueue(func() {
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
	})
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
