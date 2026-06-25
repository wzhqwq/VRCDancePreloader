package mixed_server

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func handleCacheRequest(w http.ResponseWriter, r *http.Request) {
	handleDirectVideoRequest(w, r, false)
}

func handleDownloadRequest(w http.ResponseWriter, r *http.Request) {
	handleDirectVideoRequest(w, r, true)
}

func handleDirectVideoRequest(w http.ResponseWriter, req *http.Request, download bool) {
	id := req.URL.Query().Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
	}

	if download {
		w.Header().Set("Content-Disposition", `attachment; filename="`+id+`.mp4"`)
	}

	found := false

	wg := &sync.WaitGroup{}
	wg.Add(1)

	if raw, ok := utils.CheckIdIsPyPy(id); ok {
		found = handlePlatformVideoRequest("PyPyDance", strconv.Itoa(raw), w, req, wg)
	}
	if raw, ok := utils.CheckIdIsWanna(id); ok {
		found = handlePlatformVideoRequest("WannaDance", strconv.Itoa(raw), w, req, wg)
	}
	if raw, ok := utils.CheckIdIsDuDu(id); ok {
		found = handlePlatformVideoRequest("DuDuFitDance", strconv.Itoa(raw), w, req, wg)
	}
	if raw, ok := utils.CheckIdIsBili(id); ok {
		found = handlePlatformVideoRequest("BiliBili", raw, w, req, wg)
	}
	if raw, ok := utils.CheckIdIsYoutube(id); ok {
		found = handlePlatformVideoRequest("YouTube", raw, w, req, wg)
	}

	if found {
		wg.Wait()
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
