package hijack

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
)

//var reqIncrement = 0

var (
	numericIdRegex     = regexp.MustCompile("[0-9]+")
	pypyVideoPathRegex = regexp.MustCompile(`/api/v1/videos/(\d+)\.mp4`)
	biliVideoPathRegex = regexp.MustCompile(`/(BV[a-zA-Z0-9]+)`)
)

func handlePlatformVideoRequest(platform, id string, w http.ResponseWriter, req *http.Request) (bool, *sync.WaitGroup) {
	handledCh := make(chan bool, 1)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		//reqIncrement++
		//reqId := reqIncrement
		//log.Printf("handle request %d", reqId)
		//defer log.Printf("request finished %d", reqId)

		entry, err := playlist.Request(platform, id, ctx)
		if err != nil {
			log.Printf("Failed to load %s video: %v", platform, err)
			handledCh <- false
			return
		}

		rs, err := entry.GetReadSeeker(ctx)
		if err != nil {
			log.Printf("Failed to load %s video: %v", platform, err)
			handledCh <- false
			return
		}

		contentLength := entry.TotalLen()
		if contentLength == 0 {
			log.Printf("Failed to load %s video", platform)
			handledCh <- false
			return
		}

		log.Printf("Requested %s video %s is available", platform, id)
		handledCh <- true

		rangeHeader := req.Header.Get("Range")
		if rangeHeader == "" {
			log.Printf("Intercepted %s video %s full request", platform, id)
		} else {
			log.Printf("Intercepted %s video %s range: %s", platform, id, rangeHeader)
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
	if matches := pypyVideoPathRegex.FindStringSubmatch(req.URL.Path); len(matches) > 1 {
		id := matches[1]
		if !numericIdRegex.MatchString(id) {
			log.Printf("Invalid PyPyDance id %s", id)
			return false, nil
		}

		return handlePlatformVideoRequest("PyPyDance", id, w, req)
	}
	return false, nil
}

func handleWannaRequest(w http.ResponseWriter, req *http.Request) (bool, *sync.WaitGroup) {
	if !constants.IsWannaSite(req.Host) {
		return false, nil
	}
	if req.URL.Path == "/Api/Songs/play" {
		q := req.URL.Query()
		id := q.Get("id")
		if !numericIdRegex.MatchString(id) {
			log.Printf("Invalid WannaDance id %s", id)
			return false, nil
		}

		return handlePlatformVideoRequest("WannaDance", id, w, req)
	}
	return false, nil
}

func handleBiliRequest(w http.ResponseWriter, req *http.Request) (bool, *sync.WaitGroup) {
	if !constants.IsBiliSite(req.Host) {
		return false, nil
	}
	if matches := biliVideoPathRegex.FindStringSubmatch(req.URL.Path); len(matches) > 1 {
		id := matches[1]

		return handlePlatformVideoRequest("BiliBili", id, w, req)
	}
	return false, nil
}
