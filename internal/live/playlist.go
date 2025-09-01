package live

import (
	"net/http"
	"weak"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func (s *Server) handlePlaylist(w http.ResponseWriter, r *http.Request) {
	writeOk(w, s.watcher.FullInfo())
}

type PlaylistWatcher struct {
	s *Server

	pl *playlist.PlayList

	songMap map[int64]weak.Pointer[SongWatcher]

	stopCh     chan struct{}
	listUpdate *utils.EventSubscriber[playlist.ChangeType]
}

func NewPlaylistWatcher(s *Server, pl *playlist.PlayList) *PlaylistWatcher {
	w := &PlaylistWatcher{
		s:  s,
		pl: pl,

		songMap: make(map[int64]weak.Pointer[SongWatcher]),

		stopCh:     make(chan struct{}),
		listUpdate: pl.SubscribeChangeEvent(),
	}
	go w.Loop()

	return w
}

type ShortSongInfo struct {
	ID int64 `json:"id"`
}

type PlaylistUpdateMessage struct {
	Current      []ShortSongInfo     `json:"current"`
	NewSongs     []song.LiveFullInfo `json:"newSongs"`
	RemovedSongs []ShortSongInfo     `json:"removedSongs"`
}

func (w *PlaylistWatcher) Loop() {
	w.UpdateItems()
	for {
		select {
		case <-w.stopCh:
			return
		case change := <-w.listUpdate.Channel:
			switch change {
			case playlist.ItemsChange:
				w.UpdateItems()
			case playlist.RoomChange:
			}
		}
	}
}

func (w *PlaylistWatcher) UpdateItems() {
	items := w.pl.GetItemsSnapshot()
	songs := lo.Map(items, func(item *song.PreloadedSong, index int) *SongWatcher {
		if s, ok := w.songMap[item.ID]; ok {
			if value := s.Value(); value != nil {
				return value
			}
		}
		return nil
	})

	newSongs := []song.LiveFullInfo{}
	for i, item := range items {
		if songs[i] == nil {
			s := NewSongWatcher(w.s, item)
			w.songMap[item.ID] = weak.Make(s)
			songs[i] = s
			newSongs = append(newSongs, s.song.LiveFullInfo())
		}
	}

	removedSongs := []ShortSongInfo{}
	for id, s := range w.songMap {
		if value := s.Value(); value == nil {
			removedSongs = append(removedSongs, ShortSongInfo{ID: id})
			delete(w.songMap, id)
		}
	}

	w.s.Send("PL_UPDATE", PlaylistUpdateMessage{
		Current: lo.Map(songs, func(s *SongWatcher, _ int) ShortSongInfo {
			return ShortSongInfo{ID: s.song.ID}
		}),

		NewSongs:     newSongs,
		RemovedSongs: removedSongs,
	})
}

func (w *PlaylistWatcher) FullInfo() []song.LiveFullInfo {
	return lo.Map(w.pl.GetItemsSnapshot(), func(item *song.PreloadedSong, _ int) song.LiveFullInfo {
		return item.LiveFullInfo()
	})
}

func (w *PlaylistWatcher) Stop() {
	close(w.stopCh)
	w.listUpdate.Close()
}
