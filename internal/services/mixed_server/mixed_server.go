package mixed_server

import (
	"net/http"
)

type mixedServer struct {
	http.Server

	proxy http.Handler
	http  http.Handler

	svc *Service
}

func newMixedServer(cfg Config, svc *Service) *mixedServer {
	s := &mixedServer{svc: svc}
	s.proxy = s.getProxyHandler(cfg.InterceptedSites, cfg.EnableHttpsProxy)
	s.http = s.getHttpHandler()

	s.Server.Handler = s

	return s
}

func (s *mixedServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if isProxyRequest(r) {
		s.proxy.ServeHTTP(w, r)
	} else {
		s.http.ServeHTTP(w, r)
	}
}

func (s *mixedServer) UpdateProxy(cfg Config) {
	s.proxy = s.getProxyHandler(cfg.InterceptedSites, cfg.EnableHttpsProxy)
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
