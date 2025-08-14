package live

import (
	"fmt"
	"net/http"

	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"golang.org/x/net/websocket"
)

type LiveServer struct {
	http.Server

	wsServer *websocket.Server
	sessions []*WsSession

	currentPlaylist *playlist.PlayList

	listUpdate *utils.EventSubscriber[*playlist.PlayList]
	stopCh     chan struct{}
}

func NewLiveServer(port int) *LiveServer {
	mux := http.NewServeMux()

	s := &LiveServer{
		Server: http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}

	s.RegisterHandlers(mux)

	return s
}

func (s *LiveServer) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/check_live", s.handleCheckLive)
	mux.HandleFunc("/thumbnail/{id}", s.handleThumbnail)
}
