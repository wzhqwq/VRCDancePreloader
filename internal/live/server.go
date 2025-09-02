package live

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type Server struct {
	http.Server

	sessions []*WsSession

	watcher *PlaylistWatcher

	listUpdate *utils.EventSubscriber[*playlist.PlayList]

	newSession    chan *WsSession
	closedSession chan *WsSession

	stopCh chan struct{}
	sendCh chan BroadcastMessage

	settingsCh chan SettingsChange

	running bool
}

func NewLiveServer(port int) *Server {
	mux := http.NewServeMux()

	s := &Server{
		Server: http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},

		listUpdate: playlist.SubscribeNewListEvent(),

		newSession:    make(chan *WsSession, 10),
		closedSession: make(chan *WsSession, 10),

		stopCh: make(chan struct{}),
		sendCh: make(chan BroadcastMessage, 50),

		settingsCh: make(chan SettingsChange, 50),
	}
	s.watcher = NewPlaylistWatcher(s, playlist.GetCurrentPlaylist())

	s.RegisterHandlers(mux)

	return s
}

func (s *Server) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/check_live", s.handleCheckLive)
	mux.HandleFunc("/thumbnail/{id}", s.handleThumbnail)
	mux.HandleFunc("/playlist", s.handlePlaylist)
	mux.HandleFunc("/ws", s.handleWs)
	mux.HandleFunc("/settings", s.handleSettings)
	// static
	mux.Handle("/", http.FileServerFS(staticFS{}))
}

func (s *Server) Loop() {
	for {
		select {
		case <-s.stopCh:
			return
		case pl := <-s.listUpdate.Channel:
			if s.watcher != nil {
				s.watcher.Stop()
			}
			s.watcher = NewPlaylistWatcher(s, pl)
			s.Broadcast("PL_NEW", "")
		case session := <-s.newSession:
			s.sessions = append(s.sessions, session)
		case session := <-s.closedSession:
			for i, ss := range s.sessions {
				if ss == session {
					s.sessions = slices.Delete(s.sessions, i, i+1)
					break
				}
			}
		case msg := <-s.sendCh:
			for _, session := range s.sessions {
				if msg.Except != session {
					session.SendText(msg.Content)
				}
			}
		case settings := <-s.settingsCh:
			if OnSettingsChanged != nil {
				OnSettingsChanged(settings.Settings)
			}
			s.ExclusiveBroadcast("SETTINGS", settings.Settings, settings.Initiator)
		}
	}
}

func (s *Server) Start() {
	s.running = true
	go s.Loop()
	go func() {
		if err := s.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Println("Error starting Live Server:", err)
			s.running = false
		}
	}()
}

func (s *Server) Stop() {
	if !s.running {
		return
	}
	s.running = false

	close(s.stopCh)
	if s.watcher != nil {
		s.watcher.Stop()
		s.watcher = nil
	}
	for _, session := range s.sessions {
		session.Close()
	}

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := s.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
}
