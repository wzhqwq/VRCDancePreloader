package mixed_server

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/preloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/internal_id"
)

var reqIncrement = 0

const reqIdMax = math.MaxInt32

func (s *mixedServer) handlePlatformVideoRequest(platform, id string, w http.ResponseWriter, req *http.Request, wg *sync.WaitGroup) bool {
	result := make(chan bool, 1)

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

		result <- s.servePlatformVideo(platform, id, w, req, ctx, rangeHeader, requestLogger)
	})

	return <-result
}

// servePlatformVideo answers one video request, either from the cache or from
// the origin, and reports what it delivered.
//
// Returning true means "this response is the client's"; returning false means
// the handler wrote nothing and the request has to fall back to a plain proxy
// forward (or, on the local endpoints, to a 404). That is why a real failure is
// distinguished from the deliberate pass-through: the first falls back, the
// second is a response we are serving ourselves.
func (s *mixedServer) servePlatformVideo(platform, id string, w http.ResponseWriter, req *http.Request, ctx context.Context, rangeHeader string, requestLogger utils.LoggerImpl) bool {
	f, err := s.svc.preloaderSvc.Request(id, ctx)
	if errors.Is(err, preloader.ErrPassThrough) {
		// Not in the playlist, and not worth waiting for: the download was only
		// created at the tail of the queue, so the response comes from the origin
		// and is cut short as soon as the cache entry can take over.
		if !canPassThrough(req) {
			requestLogger.ErrorLnf("Cannot pass %s video %s through: this request has no upstream destination", platform, id)
			return false
		}

		return s.handlePassThrough(w, req, id, s.svc.preloaderSvc.WatchEntryReady(id, ctx))
	}

	if err != nil {
		requestLogger.ErrorLnf("Failed to request %s video, reason: %v", platform, err)
		return false
	}

	rs, contentLength, modeTime, err := f.GetResource(ctx)
	if err != nil {
		requestLogger.ErrorLnf("Failed to load %s video, reason: %v", platform, err)
		return false
	}

	requestLogger.InfoLnf("Requested %s video %s is available", platform, id)

	if rangeHeader != "" {
		f.UpdateReqRangeStart(parseRange(rangeHeader, contentLength))
	}

	http.ServeContent(w, req, "video.mp4", modeTime, rs)

	return true
}

// platformVideoId resolves which platform video a request is asking for.
//
// It is the mapping the per-platform handlers used to spell out one by one. It
// exists as one function so that the mapping can be asserted directly, without a
// proxy, a preloader or a cache behind it.
func platformVideoId(req *http.Request) (platform string, id string, ok bool) {
	switch {
	case constants.IsPyPySite(req.Host):
		id, ok = internal_id.CheckPyPyRequest(req)
		return "PyPyDance", id, ok

	case constants.IsWannaSite(req.Host):
		id, ok = internal_id.CheckWannaRequest(req)
		return "WannaDance", id, ok

	case constants.IsDuDuSite(req.Host):
		id, ok = internal_id.CheckDuDuRequest(req)
		return "DuDuFitDance", id, ok

	case constants.IsBiliSite(req.Host):
		id, ok = internal_id.CheckBiliRequest(req)
		return "BiliBili", id, ok

	case constants.IsYouTubeSite(req.Host):
		id, ok = CheckYouTubeLocalRequest(req)
		return "YouTube", id, ok
	}

	return "", "", false
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
