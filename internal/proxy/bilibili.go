package proxy

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"log"
	"net/http"
	"regexp"
	"time"
)

func handleBiliRequest(w http.ResponseWriter, req *http.Request) bool {
	if req.Host != "api.xin.moe" {
		return false
	}
	if regexp.MustCompile(`/BV[a-zA-Z0-9]+`).MatchString(req.URL.Path) {
		id := req.URL.Path[1:]
		rangeHeader := req.Header.Get("Range")
		if rangeHeader == "" {
			log.Printf("Intercepted BiliBili video %s full request", id)
		} else {
			log.Printf("Intercepted BiliBili video %s range: %s", id, rangeHeader)
		}
		reader, err := playlist.RequestBiliSong(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			log.Println("Failed to load BiliBili video:", err)
			return true
		}
		log.Printf("Requested BiliBili video %s is available", id)

		http.ServeContent(w, req, "video.mp4", time.Now(), reader)
		return true
	}

	return false
}
