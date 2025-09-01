package live

import (
	"encoding/json"
	"log"
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
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func (ws *WsSession) Init() {

}

func (ws *WsSession) Close() {
	ws.conn.Close()
}

func (ws *WsSession) SendText(text []byte) {
	err := ws.conn.WriteMessage(websocket.TextMessage, text)
	if err != nil {
		log.Println("Error sending message to session:", err)
	}
}

func (s *Server) handleWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading to websocket:", err)
		return
	}

	session := &WsSession{
		conn: conn,
	}
	session.Init()
	s.newSession <- session
}

func (s *Server) Send(t string, payload interface{}) {
	m := Message{
		Type:    t,
		Payload: payload,
	}
	j, err := json.Marshal(m)
	if err != nil {
		log.Println("Error sending", t, ":", err)
		return
	}
	log.Println(string(j))

	for _, session := range s.sessions {
		session.SendText(j)
	}
}
