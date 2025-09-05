package live

import (
	"log"
)

var currentLiveServer *Server

var OnSettingsChanged func(settings string)
var GetSettings func() string

func StartLiveServer(port int) error {
	if currentLiveServer != nil {
		currentLiveServer.Stop()
	}
	log.Println("Starting Live Server")
	currentLiveServer = NewLiveServer(port)
	return currentLiveServer.Start()
}

func StopLiveServer() {
	if currentLiveServer != nil {
		currentLiveServer.Stop()
		currentLiveServer = nil
	}
}
