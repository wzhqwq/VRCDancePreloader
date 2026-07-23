package mixed_server

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/elazarl/goproxy"
	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
)

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

func (s *mixedServer) handleVideoRequest(w http.ResponseWriter, req *http.Request) (bool, *sync.WaitGroup) {
	defer func() {
		if e := recover(); e != nil {
			s.svc.L().ErrorLn("Error when processing request:", e)
			s.svc.L().DebugLn(string(debug.Stack()))
			s.svc.L().WarnLn("Fallback to direct access")
		}
	}()
	wg := &sync.WaitGroup{}
	//s.svc.L().InfoLn("We got:", req.Method, req.URL.String())
	if s.handlePypyRequest(w, req, wg) ||
		s.handleWannaRequest(w, req, wg) ||
		s.handleDuDuRequest(w, req, wg) ||
		s.handleBiliRequest(w, req, wg) ||
		s.handleYouTubeRequest(w, req, wg) {
		return true, wg
	}
	return false, nil
}

func (s *mixedServer) handleConnect(_ *http.Request, client net.Conn, _ *goproxy.ProxyCtx) {
	defer func() {
		if e := recover(); e != nil {
			s.svc.L().ErrorLn("error connecting to remote:", e)
			client.Write([]byte("HTTP/1.1 500 Cannot reach destination\r\n\r\n"))
		}
		client.Close()
	}()
	clientBuf := bufio.NewReadWriter(bufio.NewReader(client), bufio.NewWriter(client))
	client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	for {
		req, err := http.ReadRequest(clientBuf.Reader)
		orPanic(err)

		if req.Method == http.MethodGet || req.Method == http.MethodPost {
			rw := NewWriterGivenRespWriter(client)
			if ok, wg := s.handleVideoRequest(rw, req); ok {
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
func (s *mixedServer) handleRequest(req *http.Request, _ *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	defer func() {
		if e := recover(); e != nil {
			s.svc.L().ErrorLn("Error when processing request:", e)
			s.svc.L().DebugLn(string(debug.Stack()))
			s.svc.L().WarnLn("Fallback to direct access")
		}
	}()

	if req.Method == http.MethodGet || req.Method == http.MethodPost {
		rw, respCh := NewDeferredRespWriter(req)
		if ok, wg := s.handleVideoRequest(rw, req); ok {
			go func() {
				wg.Wait()
				rw.CloseWriter()
			}()
			return req, <-respCh
		}
	}
	return req, nil
}

func (s *mixedServer) getProxyHandler(sites []string, enableHttps bool) http.Handler {
	proxy = goproxy.NewProxyHttpServer()

	// for http proxy using CONNECT first
	for _, site := range sites {
		proxy.OnRequest(goproxy.ReqHostIs(site + ":80")).HijackConnect(s.handleConnect)
	}

	// for https proxy
	if enableHttps {
		for _, site := range sites {
			if constants.IsHttpsSite(site) {
				proxy.OnRequest(goproxy.ReqHostIs(site + ":443")).HandleConnect(goproxy.AlwaysMitm)
				proxy.OnRequest(goproxy.ReqHostIs(site + ":443")).DoFunc(s.handleRequest)
			}
		}
	}

	// for Windows system proxy which won't start with CONNECT
	for _, site := range sites {
		proxy.OnRequest(goproxy.ReqHostIs(site)).DoFunc(s.handleRequest)
	}

	return proxy
}

func SelfCheck() {
	// check for dial loop
}
