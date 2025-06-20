package proxy

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/elazarl/goproxy"
)

var runningServer *http.Server

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func handleVideoRequest(w http.ResponseWriter, req *http.Request) bool {
	if handlePypyRequest(w, req) {
		return true
	}
	if handleWannaRequest(w, req) {
		return true
	}
	if handleBiliRequest(w, req) {
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
			if handleVideoRequest(rw, req) {
				continue
			}
			log.Println("Mismatched:", req.URL.Path)
			rw.WriteHeader(http.StatusNotFound)
			rw.Write([]byte("Not found"))
		}
	}
}
func handleRequest(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	if req.Method == http.MethodGet {
		rw := NewRespWriterNoHeaderWritten()
		if handleVideoRequest(rw, req) {
			return req, rw.ToResponse(req)
		}
	}
	return req, nil
}

func Start(port string) {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true

	proxy.OnRequest(goproxy.ReqHostIs("jd.pypy.moe:80")).HijackConnect(handleConnect)
	proxy.OnRequest(goproxy.ReqHostIs("api.udon.dance:80")).HijackConnect(handleConnect)
	proxy.OnRequest(goproxy.ReqHostIs("api.xin.moe:80")).HijackConnect(handleConnect)
	proxy.OnRequest(goproxy.ReqHostIs("jd.pypy.moe")).DoFunc(handleRequest)
	proxy.OnRequest(goproxy.ReqHostIs("api.udon.dance")).DoFunc(handleRequest)
	proxy.OnRequest(goproxy.ReqHostIs("api.xin.moe")).DoFunc(handleRequest)

	runningServer = &http.Server{Addr: "127.0.0.1:" + port, Handler: proxy}
	log.Println("Starting proxy server on port", port)

	go func() {
		if err := runningServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()
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
