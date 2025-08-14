package live

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

var currentLiveServer *LiveServer

func StartLiveServer() {
	if currentLiveServer != nil {
		return
	}
	log.Println("Starting Live Server")
	currentLiveServer = NewLiveServer(5678)

	go func() {
		if err := currentLiveServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Println("Error starting Live Server:", err)
			currentLiveServer = nil
		}
	}()
}

func StopLiveServer() {
	if currentLiveServer != nil {
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownRelease()

		if err := currentLiveServer.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("HTTP shutdown error: %v", err)
		}
		currentLiveServer = nil
	}
}
