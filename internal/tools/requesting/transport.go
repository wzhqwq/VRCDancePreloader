package requesting

import (
	"net/http"
	"sync"
)

// gate is the only RoundTripper installed on every http.Client owned by this
// package. While no mock is installed it forwards each request to base (the real
// network); once a mock is installed the mock answers the request and base is
// never consulted, so a mock can never fall back to the real network by accident.
//
// base is immutable and always non-nil: it is the configured *http.Transport when
// a proxy is configured, and http.DefaultTransport otherwise.
type gate struct {
	base http.RoundTripper
}

func (g *gate) RoundTrip(req *http.Request) (*http.Response, error) {
	if m := Mock(); m != nil {
		return m.RoundTrip(req)
	}
	return g.base.RoundTrip(req)
}

// CloseIdleConnections releases the pooled connections of the transport behind
// this gate. http.Client.CloseIdleConnections looks for this method on the
// client's Transport, so without it (the gate is not an *http.Transport) the
// pooled connections of a replaced or shut down client would never be released.
func (g *gate) CloseIdleConnections() {
	if closer, ok := g.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

// newGate wraps base, replacing a nil base with http.DefaultTransport so that
// the gate always has somewhere to forward to.
func newGate(base http.RoundTripper) *gate {
	if base == nil {
		base = http.DefaultTransport
	}
	return &gate{base: base}
}

// The installed mock is looked up per request instead of being pushed into every
// gate.
//
// The previous design kept a slice of every gate ever created so that SetMock
// could reach them, which meant the package held a strong reference to every
// gate, and through gate.base to every *http.Transport it had ever wrapped:
// each proxy change (SetProxy builds a new client) pinned one more transport —
// with its connection pool — for the rest of the process lifetime, and the
// transport built by createProxyClient keeps idle connections alive. Looking the
// mock up dynamically keeps the same semantics (clients created before or after
// SetMock all honour it) with no registry at all, and no registry means there is
// nothing to leak.
var (
	mockMu     sync.RWMutex
	activeMock http.RoundTripper
)

// SetMock installs rt as the mock of every client managed by this package,
// including clients created afterwards (initClient, SetProxy, proxy hot
// switching). Passing nil is equivalent to ClearMock.
func SetMock(rt http.RoundTripper) {
	mockMu.Lock()
	activeMock = rt
	mockMu.Unlock()
}

// Mock returns the mock that is currently installed, or nil when mocking is
// disabled.
func Mock() http.RoundTripper {
	mockMu.RLock()
	defer mockMu.RUnlock()
	return activeMock
}

// ClearMock uninstalls the mock; every client goes back to the real network.
func ClearMock() {
	SetMock(nil)
}
