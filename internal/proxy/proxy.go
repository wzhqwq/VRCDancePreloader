package proxy

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
	"github.com/wzhqwq/PyPyDancePreloader/internal/playlist"
)

var runningServer *http.Server

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func handleVideoRequest(w http.ResponseWriter, req *http.Request) bool {
	if matches := regexp.MustCompile(`/api/v1/videos/(\d+)\.mp4`).FindStringSubmatch(req.URL.Path); len(matches) > 1 {
		id, err := strconv.Atoi(matches[1])
		if err != nil {
			log.Println("Failed to parse video ID:", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return true
		}

		log.Println("Intercepted video request:", id)
		reader, err := playlist.RequestPyPySong(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return true
		}

		http.ServeContent(w, req, "video.mp4", time.Now(), reader)
		return true
	}
	return false
}

func handleSongListRequest(w http.ResponseWriter, req *http.Request) bool {
	if req.URL.Path == "/api/v2/songs" {
		log.Println("Intercepted song list request")
		bodyBytes := cache.GetSongListBytes()
		// w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(bodyBytes)))
		w.Write(bodyBytes)

		return true
	}
	return false
}

func handleConnect(req *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {
	defer func() {
		if e := recover(); e != nil {
			ctx.Logf("error connecting to remote: %v", e)
			client.Write([]byte("HTTP/1.1 500 Cannot reach destination\r\n\r\n"))
		}
		client.Close()
	}()
	clientBuf := bufio.NewReadWriter(bufio.NewReader(client), bufio.NewWriter(client))
	client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	for {
		req, err := http.ReadRequest(clientBuf.Reader)
		orPanic(err)

		if req.Method == http.MethodGet {
			rw := NewRespWriter(client)
			if handleVideoRequest(rw, req) || handleSongListRequest(rw, req) {
				continue
			}
			rw.WriteHeader(http.StatusNotFound)
			rw.Write([]byte("Not found"))
		}
	}
}
func handleRequest(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	if req.Method == http.MethodGet {
		rw := NewRespWriterNoHeaderWritten()
		if handleSongListRequest(rw, req) {
			return req, rw.ToResponse(req)
		}
	}
	return req, nil
}

func Start(port string) {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true

	_, err := os.Stat("ca.crt")
	if err != nil {
		// Generate a CA certificate if it doesn't exist
		os.WriteFile("ca.crt", goproxy.CA_CERT, 0644)
	}

	proxy.OnRequest(goproxy.ReqHostIs("jd.pypy.moe:443")).HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest(goproxy.ReqHostIs("jd.pypy.moe:443")).DoFunc(handleRequest)

	proxy.OnRequest(goproxy.ReqHostIs("jd.pypy.moe:80")).HijackConnect(handleConnect)

	runningServer = &http.Server{Addr: "127.0.0.1:" + port, Handler: proxy}
	log.Println("Starting proxy server on port", port)
	if err := runningServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
	}
}

func SelfCheck() {
	// check for dial loop
}

func Stop() {
	if runningServer != nil {
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownRelease()

		if err := runningServer.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("HTTP shutdown error: %v", err)
		}
		runningServer = nil
	}
}
