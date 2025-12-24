package live

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
)

type SongWatcher struct {
	s *Server

	song *song.PreloadedSong

	stopCh chan struct{}
}

func NewSongWatcher(s *Server, song *song.PreloadedSong) *SongWatcher {
	w := &SongWatcher{
		s:    s,
		song: song,

		stopCh: make(chan struct{}),
	}
	go w.Loop()

	return w
}

func (w *SongWatcher) Loop() {
	ch := w.song.SubscribeEvent(true)
	defer ch.Close()

	for {
		select {
		case <-w.stopCh:
			return
		case event := <-ch.Channel:
			switch event {
			case song.StatusChange:
				status := w.song.GetPreloadStatus()
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
}
