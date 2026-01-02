package live

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
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

	lastSettings string
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

func (ws *WsSession) SendText(text []byte) {
	err := ws.conn.WriteMessage(websocket.TextMessage, text)
	if err != nil {
		logger.ErrorLn("Error sending message to session:", err)
	}
}

func (s *Server) handleWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.ErrorLn("Error upgrading to websocket:", err)
		return
	}

	session := &WsSession{
		conn: conn,
	}
	conn.SetCloseHandler(func(code int, text string) error {
		logger.ErrorLn("WebSocket closed:", code, text)
		if s.running {
			s.closedSession <- session
		}
		return nil
	})
	go func() {
		defer session.Close()
		for {
			mType, data, err := conn.ReadMessage()
			if err != nil {
				logger.ErrorLn("Error reading message from session:", err)
				break
			}
			if mType == websocket.TextMessage {
				logger.DebugLn("Received message:", string(data))
				s.HandleClientMessage(session, data)
			}
		}
	}()
	s.newSession <- session
}

type SettingsChange struct {
	Settings  string
	Initiator *WsSession
}

func (s *Server) HandleClientMessage(session *WsSession, data []byte) {
	msg := Message{}
	err := json.Unmarshal(data, &msg)
	if err != nil {
		logger.ErrorLn("Error unmarshalling message:", err)
	}

	switch msg.Type {
	case "SETTINGS":
		if settings, ok := msg.Payload.(string); ok {
			s.settingsCh <- SettingsChange{Settings: settings, Initiator: session}
		}
	}
}

func toJsonMessage(t string, payload interface{}) ([]byte, bool) {
	m := Message{
		Type:    t,
		Payload: payload,
	}
	j, err := json.Marshal(m)
	if err != nil {
		logger.ErrorLn("Error sending", t, ":", err)
		return nil, false
	}
	return j, true
}

func (s *Server) Broadcast(t string, payload interface{}) {
	j, ok := toJsonMessage(t, payload)
	if ok {
		s.sendCh <- BroadcastMessage{Content: j}
	}
}

func (s *Server) ExclusiveBroadcast(t string, payload interface{}, except *WsSession) {
	j, ok := toJsonMessage(t, payload)
	if ok {
		s.sendCh <- BroadcastMessage{Content: j, Except: except}
	}
}
