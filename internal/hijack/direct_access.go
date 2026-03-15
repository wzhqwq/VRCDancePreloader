package hijack

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func handleCacheRequest(w http.ResponseWriter, req *http.Request, wg *sync.WaitGroup) bool {
	if req.URL.Path == "/cached" || req.URL.Path == "/download" {
		id := req.URL.Query().Get("id")
		if id != "" {
			if req.URL.Path == "/download" {
				w.Header().Set("Content-Disposition", `attachment; filename="`+id+`.mp4"`)
			}
			if raw, ok := utils.CheckIdIsPyPy(id); ok {
				return handlePlatformVideoRequest("PyPyDance", strconv.Itoa(raw), w, req, wg)
			}
			if raw, ok := utils.CheckIdIsWanna(id); ok {
				return handlePlatformVideoRequest("WannaDance", strconv.Itoa(raw), w, req, wg)
			}
			if raw, ok := utils.CheckIdIsDuDu(id); ok {
				return handlePlatformVideoRequest("DuDuFitDance", strconv.Itoa(raw), w, req, wg)
			}
			if raw, ok := utils.CheckIdIsBili(id); ok {
				return handlePlatformVideoRequest("BiliBili", raw, w, req, wg)
			}
			if raw, ok := utils.CheckIdIsYoutube(id); ok {
				return handlePlatformVideoRequest("YouTube", raw, w, req, wg)
			}
		}

		w.WriteHeader(http.StatusBadRequest)
		wg.Done()
		return true
	}
	return false
}
