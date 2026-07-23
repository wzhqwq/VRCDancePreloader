package live

import (
	"net/http"
)

type Server struct {
	http.Server

	frontSettingsByVersion map[string]string

	svc *Service

	ws *WsService
}

func NewLiveServer(svc *Service) *Server {
	mux := http.NewServeMux()

	s := &Server{
		Server: http.Server{
			Handler: mux,
		},
		svc: svc,
	}

	s.ws = NewWsService(s)

	mux.HandleFunc("/check_live", s.handleCheckLive)
	mux.HandleFunc("/thumbnail/{id}", s.handleThumbnail)
	mux.HandleFunc("/playlist", s.handlePlaylist)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		s.ws.HandleWs(w, r)
	})
	mux.HandleFunc("/settings", s.handleSettings)
	mux.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// static
	mux.Handle("/", http.FileServerFS(staticFS{}))

	return s
}
