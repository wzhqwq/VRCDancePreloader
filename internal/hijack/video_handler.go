package hijack

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var reqIncrement = 0

const reqIdMax = math.MaxInt32

func handlePlatformVideoRequest(platform, id string, w http.ResponseWriter, req *http.Request) (bool, *sync.WaitGroup) {
	handledCh := make(chan bool, 1)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		reqIncrement = (reqIncrement + 1) % reqIdMax
		reqId := reqIncrement

		rangeHeader := req.Header.Get("Range")
		if rangeHeader == "" {
			log.Printf("[Request %d] Intercepted %s video %s full request", reqId, platform, id)
		} else {
			log.Printf("[Request %d] Intercepted %s video %s range: %s", reqId, platform, id, rangeHeader)
		}
		defer log.Printf("[Request %d] Finished", reqId)

		defer wg.Done()
		ctx, cancel := context.WithCancel(
			context.WithValue(context.Background(), "trace_id", fmt.Sprintf("Request %d", reqId)),
		)
		// TODO consider using req.Context?
		defer cancel()

		entry, err := playlist.Request(platform, id, ctx)
		if err != nil {
			log.Printf("[Request %d] Failed to load %s video, reason: %v", reqId, platform, err)
			handledCh <- false
			return
		}

		rs, err := entry.GetReadSeeker(ctx)
		if err != nil {
			log.Printf("[Request %d] Failed to load %s video, reason: %v", reqId, platform, err)
			handledCh <- false
			return
		}

		contentLength, err := entry.TotalLen()
		if err != nil {
			log.Printf("[Request %d] Failed to load %s video, reason: %v", reqId, platform, err)
			handledCh <- false
			return
		}

		log.Printf("[Request %d] Requested %s video %s is available", reqId, platform, id)
		handledCh <- true

		if rangeHeader != "" {
			entry.UpdateReqRangeStart(parseRange(rangeHeader, contentLength))
		}

		http.ServeContent(w, req, "video.mp4", entry.ModTime(), rs)
	}()

	return <-handledCh, wg
}

func handlePypyRequest(w http.ResponseWriter, req *http.Request) (bool, *sync.WaitGroup) {
	if !constants.IsPyPySite(req.Host) {
		return false, nil
	}
	if id, ok := utils.CheckPyPyRequest(req); ok {
		return handlePlatformVideoRequest("PyPyDance", id, w, req)
	}
	return false, nil
}

func handleWannaRequest(w http.ResponseWriter, req *http.Request) (bool, *sync.WaitGroup) {
	if !constants.IsWannaSite(req.Host) {
		return false, nil
	}
	if id, ok := utils.CheckWannaRequest(req); ok {
		return handlePlatformVideoRequest("WannaDance", id, w, req)
	}
	return false, nil
}

func handleBiliRequest(w http.ResponseWriter, req *http.Request) (bool, *sync.WaitGroup) {
	if !constants.IsBiliSite(req.Host) {
		return false, nil
	}
	if id, ok := utils.CheckBiliRequest(req); ok {
		return handlePlatformVideoRequest("BiliBili", id, w, req)
	}
	return false, nil
}
