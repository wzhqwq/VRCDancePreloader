package live

import (
	"encoding/json"
	"net/http"
	"slices"

	"github.com/gorilla/websocket"
	playlist2 "github.com/wzhqwq/VRCDancePreloader/internal/tools/playlist"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WsSession struct {
	conn *websocket.Conn

	version string
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type BroadcastMessage struct {
	Content []byte
	Except  *WsSession
}

func (ws *WsSession) Close() {
	ws.conn.Close()
}

func (ws *WsSession) SendText(text []byte) error {
	return ws.conn.WriteMessage(websocket.TextMessage, text)
}

type WsService struct {
	s *Server

	sessions []*WsSession

	newSessionCh    chan *WsSession
	closedSessionCh chan *WsSession

	sendCh     chan BroadcastMessage
	settingsCh chan SettingsChange

	running bool
}

func NewWsService(s *Server) *WsService {
	return &WsService{
		s: s,

		newSessionCh:    make(chan *WsSession, 10),
		closedSessionCh: make(chan *WsSession, 10),

		sendCh:     make(chan BroadcastMessage, 50),
		settingsCh: make(chan SettingsChange, 50),
	}
}

func (s *WsService) Loop(stopCh <-chan struct{}) {
	listCh := playlist2.SubscribeNewListEvent()
	defer listCh.Close()

	s.running = true
	watcher := NewPlaylistWatcher(s, playlist2.GetCurrentPlaylist())

	defer func() {
		watcher.Stop()
		s.running = false
		for _, session := range s.sessions {
			session.Close()
		}
	}()

	for {
		select {
		case <-stopCh:
			return
		case pl := <-listCh.Channel:
			if watcher != nil {
				watcher.Stop()
			}
			watcher = NewPlaylistWatcher(s, pl)
			s.Broadcast("PL_NEW", "")
		case session := <-s.newSessionCh:
			s.sessions = append(s.sessions, session)
		case session := <-s.closedSessionCh:
			for i, ss := range s.sessions {
				if ss == session {
					s.sessions = slices.Delete(s.sessions, i, i+1)
					break
				}
			}
		case msg := <-s.sendCh:
			for _, session := range s.sessions {
				if msg.Except != session {
					err := session.SendText(msg.Content)
					if err != nil {
						s.s.svc.L().ErrorLn("Error sending message to session:", err)
					}
				}
			}
		case settings := <-s.settingsCh:
			s.s.setSetting(settings.Initiator.version, settings.Settings)
			s.ExclusiveBroadcast("SETTINGS", settings.Settings, settings.Initiator)
		}
	}
}

func (s *WsService) HandleWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.s.svc.L().ErrorLn("Error upgrading to websocket:", err)
		return
	}

	session := &WsSession{
		conn: conn,
	}
	conn.SetCloseHandler(func(code int, text string) error {
		s.s.svc.L().ErrorLn("WebSocket closed:", code, text)

		if s.running {
			s.closedSessionCh <- session
		}
		return nil
	})
	go func() {
		defer session.Close()
		for {
			mType, data, err := conn.ReadMessage()
			if err != nil {
				s.s.svc.L().ErrorLn("Error reading message from session:", err)
				break
			}
			if mType == websocket.TextMessage {
				s.s.svc.L().DebugLn("Received message:", string(data))
				s.HandleClientMessage(session, data)
			}
		}
	}()
	s.newSessionCh <- session
}

type SettingsChange struct {
	Settings  string
	Initiator *WsSession
}

func (s *WsService) HandleClientMessage(session *WsSession, data []byte) {
	msg := Message{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		s.s.svc.L().ErrorLn("Error unmarshalling message:", err)
	}

	switch msg.Type {
	case "VERSION":
		session.version = msg.Payload.(string)
	case "SETTINGS":
		if settings, ok := msg.Payload.(string); ok {
			s.settingsCh <- SettingsChange{Settings: settings, Initiator: session}
		}
	}
}

func (s *WsService) toJsonMessage(t string, payload interface{}) ([]byte, bool) {
	m := Message{
		Type:    t,
		Payload: payload,
	}
	j, err := json.Marshal(m)
	if err != nil {
		s.s.svc.L().ErrorLn("Error sending", t, ":", err)
		return nil, false
	}
	return j, true
}

func (s *WsService) Broadcast(t string, payload interface{}) {
	j, ok := s.toJsonMessage(t, payload)
	if ok {
		s.sendCh <- BroadcastMessage{Content: j}
	}
}

func (s *WsService) ExclusiveBroadcast(t string, payload interface{}, except *WsSession) {
	j, ok := s.toJsonMessage(t, payload)
	if ok {
		s.sendCh <- BroadcastMessage{Content: j, Except: except}
	}
}
