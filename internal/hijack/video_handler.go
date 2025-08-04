package hijack

import (
	"context"
	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"log"
	"net/http"
	"regexp"
)

var (
	numericIdRegex     = regexp.MustCompile("[0-9]+")
	pypyVideoPathRegex = regexp.MustCompile(`/api/v1/videos/(\d+)\.mp4`)
	biliVideoPathRegex = regexp.MustCompile(`/(BV[a-zA-Z0-9]+)`)
)

func handlePlatformVideoRequest(platform, id string, w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	entry, err := playlist.Request(platform, id, ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		log.Printf("Failed to load %s video: %v", platform, err)
		return
	}
	log.Printf("Requested %s video %s is available", platform, id)

	rangeHeader := req.Header.Get("Range")
	if rangeHeader == "" {
		log.Printf("Intercepted %s video %s full request", platform, id)
	} else {
		log.Printf("Intercepted %s video %s range: %s", platform, id, rangeHeader)
		playlist.DownloadSuffix(platform, id, parseRange(rangeHeader, entry.TotalLen()))
	}

	http.ServeContent(w, req, "video.mp4", entry.ModTime(), entry.GetReadSeeker(ctx))
}

func handlePypyRequest(w http.ResponseWriter, req *http.Request) bool {
	if !constants.IsPyPySite(req.Host) {
		return false
	}
	if matches := pypyVideoPathRegex.FindStringSubmatch(req.URL.Path); len(matches) > 1 {
		id := matches[1]

		handlePlatformVideoRequest("PyPyDance", id, w, req)
		return true
	}
	return false
}

func handleWannaRequest(w http.ResponseWriter, req *http.Request) bool {
	if !constants.IsWannaSite(req.Host) {
		return false
	}
	if req.URL.Path == "/Api/Songs/play" {
		q := req.URL.Query()
		id := q.Get("id")
		if !numericIdRegex.MatchString(id) {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return false
		}

		handlePlatformVideoRequest("WannaDance", id, w, req)
		return true
	}
	return false
}

func handleBiliRequest(w http.ResponseWriter, req *http.Request) bool {
	if !constants.IsBiliSite(req.Host) {
		return false
	}
	if matches := biliVideoPathRegex.FindStringSubmatch(req.URL.Path); len(matches) > 1 {
		id := matches[1]

		handlePlatformVideoRequest("BiliBili", id, w, req)
		return true
	}
	return false
}
