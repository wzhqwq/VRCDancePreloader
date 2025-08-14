package live

import (
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WsSession struct {
	conn *websocket.Conn
}

func (ws *WsSession) Init() {

}

//func (s *LiveServer) handleWs(w http.ResponseWriter, r *http.Request) {
//	conn, err := upgrader.Upgrade(w, r, nil)
//	if err != nil {
//		w.WriteHeader(http.StatusBadRequest)
//		return
//	}
//
//	session := &WsSession{
//		conn: conn,
//	}
//}
