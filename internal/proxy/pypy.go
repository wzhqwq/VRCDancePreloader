package proxy

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"log"
	"net/http"
	"regexp"
	"strconv"
)

func handlePypyRequest(w http.ResponseWriter, req *http.Request) bool {
	if !constants.IsPyPySite(req.Host) {
		return false
	}
	if matches := regexp.MustCompile(`/api/v1/videos/(\d+)\.mp4`).FindStringSubmatch(req.URL.Path); len(matches) > 1 {
		id, err := strconv.Atoi(matches[1])
		if err != nil {
			log.Println("Failed to parse video ID:", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return true
		}

		rangeHeader := req.Header.Get("Range")
		if rangeHeader == "" {
			log.Printf("Intercepted PyPyDance video %d full request", id)
		} else {
			log.Printf("Intercepted PyPyDance video %d range: %s", id, rangeHeader)
		}
		reader, modTime, err := playlist.RequestPyPySong(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			log.Println("Failed to load PyPyDance video:", err)
			return true
		}
		log.Printf("Requested PyPyDance video %d is available", id)

		http.ServeContent(w, req, "video.mp4", modTime, reader)
		return true
	}

	return false
}
