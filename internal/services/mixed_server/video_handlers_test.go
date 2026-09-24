package mixed_server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// platformVideoId is the mapping every per-platform handler used to spell out,
// and it is what decides whether a request is intercepted at all. A regression
// here is a video that silently stops being cached, so the whole table is pinned.
func TestPlatformVideoIdTable(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		target   string
		platform string
		id       string
		ok       bool
	}{
		{
			name:     "pypy api",
			host:     "api.pypy.dance",
			target:   "http://api.pypy.dance/video?id=123",
			platform: "PyPyDance",
			id:       "pypy_123",
			ok:       true,
		},
		{
			name:     "pypy legacy path",
			host:     "api.pypy.dance",
			target:   "http://api.pypy.dance/videos/123.mp4",
			platform: "PyPyDance",
			id:       "123",
			ok:       true,
		},
		{
			name:     "pypy api without an id",
			host:     "api.pypy.dance",
			target:   "http://api.pypy.dance/video",
			platform: "PyPyDance",
			ok:       false,
		},
		{
			name:     "wanna",
			host:     "api.udon.dance",
			target:   "http://api.udon.dance/Api/Songs/play?id=456",
			platform: "WannaDance",
			id:       "wanna_456",
			ok:       true,
		},
		{
			name:     "dudu",
			host:     "api.dudufit.dance",
			target:   "http://api.dudufit.dance/api/v1/videos/789",
			platform: "DuDuFitDance",
			id:       "dudu_789",
			ok:       true,
		},
		{
			name:     "bilibili",
			host:     "www.bilibili.com",
			target:   "https://www.bilibili.com/video/BV1xx411c7mD",
			platform: "BiliBili",
			id:       "bili_BV1xx411c7mD",
			ok:       true,
		},
		{
			name:     "youtube local",
			host:     "www.youtube.com",
			target:   "https://www.youtube.com/local?id=abc",
			platform: "YouTube",
			id:       "yt_abc",
			ok:       true,
		},
		{
			name:   "a host we do not intercept",
			host:   "example.com",
			target: "http://example.com/video.mp4",
			ok:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			req.Host = tt.host

			platform, id, ok := platformVideoId(req)
			if ok != tt.ok {
				t.Fatalf("platformVideoId(%s) ok = %v, want %v", tt.target, ok, tt.ok)
			}
			if !ok {
				return
			}

			if platform != tt.platform || id != tt.id {
				t.Fatalf("platformVideoId(%s) = (%s, %s), want (%s, %s)", tt.target, platform, id, tt.platform, tt.id)
			}
		})
	}
}

// canPassThrough decides whether the streaming path is allowed to dial the
// origin itself. It has to be true for a request that came through the proxy —
// otherwise a queued temporary video could not be served at all — and false for
// one that reached the local /cached or /download endpoint, where there is no
// upstream host to name.
func TestCanPassThroughOnlyForProxiedRequests(t *testing.T) {
	t.Run("http proxy request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/video?id=1", nil)
		req.Host = "api.pypy.dance"

		if !canPassThrough(req) {
			t.Fatal("an origin-form request with a host must be servable from the origin")
		}
	})

	t.Run("mitm request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.dudufit.dance/api/v1/videos/1.mp4", nil)

		if !canPassThrough(req) {
			t.Fatal("a MITM request carries an absolute URL and must be servable from the origin")
		}
	})

	t.Run("local cached endpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/cached?id=pypy_1", nil)
		// httptest fills Host in; the local endpoints are the case where the
		// request never named an upstream at all.
		req.Host = ""

		if canPassThrough(req) {
			t.Fatal("a request to the local endpoint has no upstream destination")
		}
	})

	t.Run("a method that cannot be replayed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodHead, "/video?id=1", nil)
		req.Host = "api.pypy.dance"

		if canPassThrough(req) {
			t.Fatal("only a GET or POST body can be replayed upstream")
		}
	})
}

// The destination of a replay is the URL for a proxied request and the host plus
// path for one read off the CONNECT path, which carries no scheme at all.
func TestPassThroughDestinationCompletesTheURL(t *testing.T) {
	mitm := httptest.NewRequest(http.MethodGet, "https://api.dudufit.dance/api/v1/videos/1.mp4", nil)
	if got := passThroughDestination(mitm).String(); got != "https://api.dudufit.dance/api/v1/videos/1.mp4" {
		t.Fatalf("destination = %s, want the request's own URL", got)
	}

	connect := httptest.NewRequest(http.MethodGet, "/video?id=1", nil)
	connect.Host = "api.pypy.dance"
	if got := passThroughDestination(connect).String(); got != "http://api.pypy.dance/video?id=1" {
		t.Fatalf("destination = %s, want the host completed with the path", got)
	}

	if got := passThroughDialAddress(connect); got != "api.pypy.dance:80" {
		t.Fatalf("dial address = %s, want the default HTTP port added", got)
	}

	withPort := httptest.NewRequest(http.MethodGet, "/video?id=1", nil)
	withPort.Host = "127.0.0.1:8080"
	if got := passThroughDialAddress(withPort); got != "127.0.0.1:8080" {
		t.Fatalf("dial address = %s, want the request's own port kept", got)
	}
}
