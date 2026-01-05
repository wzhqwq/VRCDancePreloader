package hijack

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var reqIncrement = 0

const reqIdMax = math.MaxInt32

func handlePlatformVideoRequest(platform, id string, w http.ResponseWriter, req *http.Request, wg *sync.WaitGroup) bool {
	handledCh := make(chan bool, 1)

	go func() {
		reqIncrement = (reqIncrement + 1) % reqIdMax
		reqId := reqIncrement
		requestLogger := utils.NewLogger(fmt.Sprintf("Request %d", reqId))

		rangeHeader := req.Header.Get("Range")
		if rangeHeader == "" {
			requestLogger.InfoLnf("Intercepted %s video %s full request", platform, id)
		} else {
			requestLogger.InfoLnf("Intercepted %s video %s range: %s", platform, id, rangeHeader)
		}
		defer requestLogger.InfoLn("Finished")

		defer wg.Done()
		ctx, cancel := context.WithCancel(
			context.WithValue(
				context.WithValue(
					context.Background(),
					"logger", requestLogger,
				),
				"trace_id", fmt.Sprintf("Request %d", reqId),
			),
		)
		// TODO consider using req.Context?
		defer cancel()

		entry, err := playlist.Request(platform, id, ctx)
		if err != nil {
			requestLogger.ErrorLnf("Failed to load %s video, reason: %v", platform, err)
			handledCh <- false
			return
		}

		rs, err := entry.GetReadSeeker(ctx)
		if err != nil {
			requestLogger.ErrorLnf("Failed to load %s video, reason: %v", platform, err)
			handledCh <- false
			return
		}

		contentLength, err := entry.TotalLen()
		if err != nil {
			requestLogger.ErrorLnf("Failed to load %s video, reason: %v", platform, err)
			handledCh <- false
			return
		}

		requestLogger.InfoLnf("Requested %s video %s is available", platform, id)
		handledCh <- true

		if rangeHeader != "" {
			entry.UpdateReqRangeStart(parseRange(rangeHeader, contentLength))
		}

		http.ServeContent(w, req, "video.mp4", entry.ModTime(), rs)
	}()

	return <-handledCh
}

func handlePypyRequest(w http.ResponseWriter, req *http.Request, wg *sync.WaitGroup) bool {
	if !constants.IsPyPySite(req.Host) {
		return false
	}
	if id, ok := utils.CheckPyPyRequest(req); ok {
		return handlePlatformVideoRequest("PyPyDance", id, w, req, wg)
	}
	return false
}

func handleWannaRequest(w http.ResponseWriter, req *http.Request, wg *sync.WaitGroup) bool {
	if !constants.IsWannaSite(req.Host) {
		return false
	}
	if id, ok := utils.CheckWannaRequest(req); ok {
		return handlePlatformVideoRequest("WannaDance", id, w, req, wg)
	}
	return false
}

func handleDuDuRequest(w http.ResponseWriter, req *http.Request, wg *sync.WaitGroup) bool {
	if !constants.IsDuDuSite(req.Host) {
		return false
	}
	if id, ok := utils.CheckDuDuRequest(req); ok {
		return handlePlatformVideoRequest("DuDuFitDance", id, w, req, wg)
	}
	return false
}

func handleBiliRequest(w http.ResponseWriter, req *http.Request, wg *sync.WaitGroup) bool {
	if !constants.IsBiliSite(req.Host) {
		return false
	}
	if id, ok := utils.CheckBiliRequest(req); ok {
		return handlePlatformVideoRequest("BiliBili", id, w, req, wg)
	}
	return false
}
