package live

import (
	"log"
)

var currentLiveServer *Server

func StartLiveServer() {
	if currentLiveServer != nil {
		currentLiveServer.Stop()
	}
	log.Println("Starting Live Server")
	currentLiveServer = NewLiveServer(7652)
	currentLiveServer.Start()
}

func StopLiveServer() {
	if currentLiveServer != nil {
		currentLiveServer.Stop()
		currentLiveServer = nil
	}
}
