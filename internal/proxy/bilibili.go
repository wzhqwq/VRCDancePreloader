package proxy

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"log"
	"net/http"
	"regexp"
)

func handleBiliRequest(w http.ResponseWriter, req *http.Request) bool {
	if !constants.IsBiliSite(req.Host) {
		return false
	}
	if matches := regexp.MustCompile(`/(BV[a-zA-Z0-9]+)`).FindStringSubmatch(req.URL.Path); len(matches) > 1 {
		id := matches[1]
		rangeHeader := req.Header.Get("Range")
		if rangeHeader == "" {
			log.Printf("Intercepted BiliBili video %s full request", id)
		} else {
			log.Printf("Intercepted BiliBili video %s range: %s", id, rangeHeader)
		}
		reader, modTime, err := playlist.RequestBiliSong(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			log.Println("Failed to load BiliBili video:", err)
			return true
		}
		log.Printf("Requested BiliBili video %s is available", id)

		http.ServeContent(w, req, "video.mp4", modTime, reader)
		return true
	}

	return false
}
