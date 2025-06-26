package proxy

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/elazarl/goproxy"
)

var runningServer *http.Server
var proxy *goproxy.ProxyHttpServer

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}

// copied/converted from https.go
func dial(ctx context.Context, network, addr string) (c net.Conn, err error) {
	if proxy.Tr.DialContext != nil {
		return proxy.Tr.DialContext(ctx, network, addr)
	}
	var d net.Dialer
	return d.DialContext(ctx, network, addr)
}

// copied/converted from https.go
func connectDial(ctx context.Context, network, addr string) (c net.Conn, err error) {
	if proxy.ConnectDial == nil {
		return dial(ctx, network, addr)
	}
	return proxy.ConnectDial(network, addr)
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

func handleConnect(_ *http.Request, client net.Conn, ctx *goproxy.ProxyCtx) {
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
		}

		remote, err := connectDial(req.Context(), "tcp", req.Host+":80")
		orPanic(err)
		remoteBuf := bufio.NewReadWriter(bufio.NewReader(remote), bufio.NewWriter(remote))
		orPanic(req.Write(remoteBuf))
		orPanic(remoteBuf.Flush())
		resp, err := http.ReadResponse(remoteBuf.Reader, req)
		orPanic(err)
		orPanic(resp.Write(clientBuf.Writer))
		orPanic(clientBuf.Flush())

		remote.Close()
	}
}

// for https
func handleRequest(req *http.Request, _ *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	if req.Method == http.MethodGet {
		rw := NewStandaloneRespWriter()
		if handleVideoRequest(rw, req) {
			return req, rw.ToResponse(req)
		}
	}
	return req, nil
}

func Start(sites []string, enableHttps bool, port int) {
	proxy = goproxy.NewProxyHttpServer()

	// for http proxy using CONNECT first
	for _, site := range sites {
		proxy.OnRequest(goproxy.ReqHostIs(site + ":80")).HijackConnect(handleConnect)
	}

	// for https proxy
	if enableHttps {
		for _, site := range sites {
			proxy.OnRequest(goproxy.ReqHostIs(site + ":443")).HandleConnect(goproxy.AlwaysMitm)
			proxy.OnRequest(goproxy.ReqHostIs(site + ":443")).DoFunc(handleRequest)
		}
	}

	// for Windows system proxy which won't start with CONNECT
	for _, site := range sites {
		proxy.OnRequest(goproxy.ReqHostIs(site)).DoFunc(handleRequest)
	}

	runningServer = &http.Server{Addr: "127.0.0.1:" + strconv.Itoa(port), Handler: proxy}
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
