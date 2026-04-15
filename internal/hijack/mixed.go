package hijack

import (
	"net/http"
	"sync"
)

type MixedProxyServer struct {
	proxyService http.Handler
}

func (s *MixedProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	if handleCacheRequest(w, r, wg) {
		wg.Wait()
		return
	}

	s.proxyService.ServeHTTP(w, r)
}
