package mixed_server

import (
	"net/http"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils/internal_id"
)

func (s *mixedServer) getHttpHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/download", s.handleDownloadRequest)
	mux.HandleFunc("/cached", s.handleCacheRequest)
	mux.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return mux
}

func (s *mixedServer) handleCacheRequest(w http.ResponseWriter, r *http.Request) {
	s.handleDirectVideoRequest(w, r, false)
}

func (s *mixedServer) handleDownloadRequest(w http.ResponseWriter, r *http.Request) {
	s.handleDirectVideoRequest(w, r, true)
}

func (s *mixedServer) handleDirectVideoRequest(w http.ResponseWriter, req *http.Request, download bool) {
	id := req.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	if download {
		w.Header().Set("Content-Disposition", `attachment; filename="`+id+`.mp4"`)
	}

	found := false

	wg := &sync.WaitGroup{}

	if platform := internal_id.GetPlatformNameByInternalId(id); platform != "" {
		found = s.handlePlatformVideoRequest(platform, id, w, req, wg)
	}

	if found {
		wg.Wait()
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
