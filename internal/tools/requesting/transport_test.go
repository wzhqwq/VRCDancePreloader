package requesting

import (
	"net/http"
	"testing"
)

// recordingTransport answers every request with a canned response and counts how
// often its pooled connections were released.
type recordingTransport struct {
	closed int
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return newMockResponse(req, http.StatusOK, nil, []byte("ok"), 2), nil
}

func (r *recordingTransport) CloseIdleConnections() {
	r.closed++
}

// T1: replacing a client releases the pooled connections of the transport it
// leaves behind.
//
// Nothing else can: the replaced *http.Client is unreachable after SetProxy, its
// transport is only reachable through it, and a pooled connection keeps its own
// transport alive through the connection's goroutine.
func TestSetProxyReleasesTheReplacedTransport(t *testing.T) {
	p := NewProxyProvider("", "transport_test", testCase{})

	recorder := &recordingTransport{}
	p.client = &http.Client{Transport: newGate(recorder)}

	p.SetProxy("")

	if recorder.closed != 1 {
		t.Fatalf("CloseIdleConnections calls on the replaced transport = %d, want 1", recorder.closed)
	}
}

// T2: shutting the tool down releases the connections of the client that stays in
// the map (SetProxy can revive it afterwards, so it is not garbage either).
func TestShutdownReleasesTheTransport(t *testing.T) {
	p := NewProxyProvider("", "transport_test", testCase{})

	recorder := &recordingTransport{}
	p.client = &http.Client{Transport: newGate(recorder)}

	p.Shutdown()

	if recorder.closed != 1 {
		t.Fatalf("CloseIdleConnections calls on shutdown = %d, want 1", recorder.closed)
	}
}

// T3: a hot switch updates ProxyUrl.
//
// That field is what yt-dlp is handed (local_executables/ytdlp.go passes it to
// --proxy) and what the availability log describes, so a stale value keeps
// sending the *previous* proxy to yt-dlp after the user changed it.
func TestSetProxyUpdatesProxyUrl(t *testing.T) {
	p := NewProxyProvider("", "transport_test", testCase{})
	if p.ProxyUrl != "" {
		t.Fatalf("ProxyUrl = %q after construction, want empty", p.ProxyUrl)
	}

	p.SetProxy("http://127.0.0.1:3128")
	if p.ProxyUrl != "http://127.0.0.1:3128" {
		t.Fatalf("ProxyUrl = %q after switching to a proxy, want the new value", p.ProxyUrl)
	}

	p.SetProxy("")
	if p.ProxyUrl != "" {
		t.Fatalf("ProxyUrl = %q after switching back to no proxy, want empty", p.ProxyUrl)
	}
}

// T4: the installed mock is honoured by clients that already existed and by the
// client a later SetProxy installs.
//
// This is what replaced the gate registry: the mock is looked up when a request
// is made instead of being pushed into every gate, so no registry has to hold a
// strong reference to every gate ever created (each of which pins an
// *http.Transport).
func TestMockAppliesToExistingAndReplacedClients(t *testing.T) {
	p := NewProxyProvider("", "transport_test", testCase{})

	SetMock(NewMockTransport(MockRule{Name: "late", Host: "late.test", Path: "/x", Body: []byte("mocked")}))
	t.Cleanup(ClearMock)

	if _, body := mustGet(t, p, "http://late.test/x", nil); body != "mocked" {
		t.Fatalf("a client built before SetMock did not honour it: body = %q", body)
	}

	p.SetProxy("")

	if _, body := mustGet(t, p, "http://late.test/x", nil); body != "mocked" {
		t.Fatalf("the client installed by SetProxy did not honour the mock: body = %q", body)
	}
}
