package live

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type Server struct {
	http.Server

	sessions []*WsSession

	watcher *PlaylistWatcher

	listUpdate *utils.EventSubscriber[*playlist.PlayList]
	newSession chan *WsSession
	stopCh     chan struct{}

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
		newSession: make(chan *WsSession),
		stopCh:     make(chan struct{}),
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
			s.Send("PL_NEW", "")
		case session := <-s.newSession:
			s.sessions = append(s.sessions, session)
		}
	}
}

func (s *Server) Start() {
	go s.Loop()
	go func() {
		s.running = true
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

	close(s.stopCh)
	if s.watcher != nil {
		s.watcher.Stop()
		s.watcher = nil
	}

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := s.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
	s.running = false
}
