package live

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type SongWatcher struct {
	s *Server

	song *song.PreloadedSong

	stopCh     chan struct{}
	songUpdate *utils.EventSubscriber[song.ChangeType]
}

func NewSongWatcher(s *Server, song *song.PreloadedSong) *SongWatcher {
	w := &SongWatcher{
		s:    s,
		song: song,

		stopCh:     make(chan struct{}),
		songUpdate: song.SubscribeEvent(true),
	}
	go w.Loop()

	return w
}

func (w *SongWatcher) Loop() {
	for {
		select {
		case <-w.stopCh:
			return
		case event := <-w.songUpdate.Channel:
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
	w.songUpdate.Close()
}
