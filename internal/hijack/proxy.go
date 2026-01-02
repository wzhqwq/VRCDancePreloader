package hijack

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"

	"github.com/elazarl/goproxy"
)

var runningServer *http.Server
var proxy *goproxy.ProxyHttpServer

var logger = utils.NewLogger("Hijacking")

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

func handleVideoRequest(w http.ResponseWriter, req *http.Request) (ok bool, wg *sync.WaitGroup) {
	defer func() {
		if e := recover(); e != nil {
			logger.ErrorLn("Error when processing request:", e)
			logger.DebugLn(string(debug.Stack()))
			logger.WarnLn("Fallback to direct access")
		}
	}()
	ok, wg = handlePypyRequest(w, req)
	if ok {
		return
	}
	ok, wg = handleWannaRequest(w, req)
	if ok {
		return
	}
	ok, wg = handleBiliRequest(w, req)
	if ok {
		return
	}
	return
}

func handleConnect(_ *http.Request, client net.Conn, _ *goproxy.ProxyCtx) {
	defer func() {
		if e := recover(); e != nil {
			logger.ErrorLn("error connecting to remote:", e)
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
			rw := NewWriterGivenRespWriter(client)
			if ok, wg := handleVideoRequest(rw, req); ok {
				wg.Wait()
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

// for common request
func handleRequest(req *http.Request, _ *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	defer func() {
		if e := recover(); e != nil {
			logger.ErrorLn("Error when processing request:", e)
			logger.DebugLn(string(debug.Stack()))
			logger.WarnLn("Fallback to direct access")
		}
	}()

	if req.Method == http.MethodGet {
		rw, respCh := NewDeferredRespWriter(req)
		if ok, wg := handleVideoRequest(rw, req); ok {
			go func() {
				defer rw.CloseWriter()
				wg.Wait()
			}()
			return req, <-respCh
		}
	}
	return req, nil
}

func Start(sites []string, enableHttps bool, port int) error {
	proxy = goproxy.NewProxyHttpServer()

	// for http proxy using CONNECT first
	for _, site := range sites {
		proxy.OnRequest(goproxy.ReqHostIs(site + ":80")).HijackConnect(handleConnect)
	}

	// for https proxy
	if enableHttps {
		for _, site := range sites {
			if constants.IsHttpsSite(site) {
				proxy.OnRequest(goproxy.ReqHostIs(site + ":443")).HandleConnect(goproxy.AlwaysMitm)
				proxy.OnRequest(goproxy.ReqHostIs(site + ":443")).DoFunc(handleRequest)
			}
		}
	}

	// for Windows system proxy which won't start with CONNECT
	for _, site := range sites {
		proxy.OnRequest(goproxy.ReqHostIs(site)).DoFunc(handleRequest)
	}

	runningServer = &http.Server{Addr: "127.0.0.1:" + strconv.Itoa(port), Handler: proxy}
	logger.InfoLn("Starting server on port", port)

	if err := runningServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func SelfCheck() {
	// check for dial loop
}

func Stop() {
	if runningServer != nil {
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownRelease()

		if err := runningServer.Shutdown(shutdownCtx); err != nil {
			logger.FatalLn("HTTP shutdown error:", err)
		}
		runningServer = nil
	}
}
