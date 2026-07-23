package live

import (
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/song"
)

type SongWatcher struct {
	s *WsService

	song *song.StatefulSong

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewSongWatcher(s *WsService, song *song.StatefulSong) *SongWatcher {
	w := &SongWatcher{
		s:    s,
		song: song,

		stopCh: make(chan struct{}),
	}
	w.wg.Go(w.loop)

	return w
}

func (w *SongWatcher) loop() {
	ch := w.song.SubscribeEvent(true)
	defer ch.Close()

	for {
		select {
		case <-w.stopCh:
			return
		case event := <-ch.Channel:
			switch event {
			case song.StatusChange:
				status := w.song.PreloadStatus()
				if status == song.Removed {
					w.Stop()
				} else {
					w.s.Broadcast("SONG_UPDATE", w.song.LiveStatusChange())
				}
			case song.ProgressChange:
				w.s.Broadcast("SONG_UPDATE", w.song.LiveProgressChange())
			case song.TimeChange:
				w.s.Broadcast("SONG_UPDATE", w.song.LivePlayStatusChange())
			case song.BasicInfoChange:
				w.s.Broadcast("SONG_UPDATE", w.song.LiveBasicInfoChange())
			}
		}
	}
}

func (w *SongWatcher) Stop() {
	close(w.stopCh)
	w.wg.Wait()
}
