package mixed_server

import (
	"net/http"
	"strconv"
)

type mixedServer struct {
	http.Server

	proxy http.Handler
	http  http.Handler
}

func newMixedServer(cfg Config) *mixedServer {
	s := &mixedServer{
		Server: http.Server{
			Addr: "127.0.0.1:" + strconv.Itoa(cfg.Port),
		},
		proxy: getProxyHandler(cfg.InterceptedSites, cfg.EnableHttpsProxy),
		http:  getHttpHandler(),
	}

	s.registerHandler()

	return s
}

func (s *mixedServer) registerHandler() {
	s.Server.Handler = http.HandlerFunc(s.serveHTTP)
}

func (s *mixedServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if isProxyRequest(r) {
		s.proxy.ServeHTTP(w, r)
	} else {
		s.http.ServeHTTP(w, r)
	}
}

func (s *mixedServer) UpdatePort(cfg Config) {
	s.Server.Addr = "127.0.0.1:" + strconv.Itoa(cfg.Port)
}

func (s *mixedServer) UpdateProxy(cfg Config) {
	s.proxy = getProxyHandler(cfg.InterceptedSites, cfg.EnableHttpsProxy)
}

func isProxyRequest(r *http.Request) bool {
	// HTTPS proxy tunnel:
	// CONNECT xxx:443 HTTP/1.1
	if r.Method == http.MethodConnect {
		return true
	}

	// HTTP proxy absolute-form:
	// GET http://example.com/path HTTP/1.1
	if r.URL != nil && r.URL.IsAbs() {
		return true
	}

	return false
}
