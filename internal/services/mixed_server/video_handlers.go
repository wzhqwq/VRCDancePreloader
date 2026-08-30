package mixed_server

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/internal_id"
)

var reqIncrement = 0

const reqIdMax = math.MaxInt32

func (s *mixedServer) handlePlatformVideoRequest(platform, id string, w http.ResponseWriter, req *http.Request, wg *sync.WaitGroup) bool {
	handledCh := make(chan bool, 1)

	wg.Go(func() {
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

		f, err := s.svc.preloaderSvc.Request(id, ctx)
		if err != nil {
			requestLogger.ErrorLnf("Failed to request %s video, reason: %v", platform, err)
			handledCh <- false
			return
		}

		rs, contentLength, modeTime, err := f.GetResource(ctx)
		if err != nil {
			requestLogger.ErrorLnf("Failed to load %s video, reason: %v", platform, err)
			handledCh <- false
			return
		}

		requestLogger.InfoLnf("Requested %s video %s is available", platform, id)
		handledCh <- true

		if rangeHeader != "" {
			f.UpdateReqRangeStart(parseRange(rangeHeader, contentLength))
		}

		http.ServeContent(w, req, "video.mp4", modeTime, rs)
	})

	return <-handledCh
}

func (s *mixedServer) handlePypyRequest(w http.ResponseWriter, req *http.Request, wg *sync.WaitGroup) bool {
	if !constants.IsPyPySite(req.Host) {
		return false
	}
	if id, ok := internal_id.CheckPyPyRequest(req); ok {
		return s.handlePlatformVideoRequest("PyPyDance", id, w, req, wg)
	}
	return false
}

func (s *mixedServer) handleWannaRequest(w http.ResponseWriter, req *http.Request, wg *sync.WaitGroup) bool {
	if !constants.IsWannaSite(req.Host) {
		return false
	}
	if id, ok := internal_id.CheckWannaRequest(req); ok {
		return s.handlePlatformVideoRequest("WannaDance", id, w, req, wg)
	}
	return false
}

func (s *mixedServer) handleDuDuRequest(w http.ResponseWriter, req *http.Request, wg *sync.WaitGroup) bool {
	if !constants.IsDuDuSite(req.Host) {
		return false
	}
	if id, ok := internal_id.CheckDuDuRequest(req); ok {
		return s.handlePlatformVideoRequest("DuDuFitDance", id, w, req, wg)
	}
	return false
}

func (s *mixedServer) handleBiliRequest(w http.ResponseWriter, req *http.Request, wg *sync.WaitGroup) bool {
	if !constants.IsBiliSite(req.Host) {
		return false
	}
	if id, ok := internal_id.CheckBiliRequest(req); ok {
		return s.handlePlatformVideoRequest("BiliBili", id, w, req, wg)
	}
	return false
}

func (s *mixedServer) handleYouTubeRequest(w http.ResponseWriter, req *http.Request, wg *sync.WaitGroup) bool {
	if !constants.IsYouTubeSite(req.Host) {
		return false
	}
	if id, ok := CheckYouTubeWebPageRequest(req); ok {
		return handleYouTubeWebpage(w, id, wg)
	}
	if id, ok := CheckInnerTubeApiRequest(req); ok {
		return handleInnerTubeApi(w, id, wg)
	}
	if id, ok := CheckYouTubeLocalRequest(req); ok {
		return s.handlePlatformVideoRequest("YouTube", id, w, req, wg)
	}
	return false
}
