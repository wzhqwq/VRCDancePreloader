package mixed_server

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/preloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
)

// The cache and the database a preloader needs are process wide, so they are
// brought up once under a temporary root. Nothing here starts the third party
// providers: the pass-through does not resolve anything, so it does not need
// them.
var (
	setupServerTestData   = sync.OnceFunc(setupServerTestDataOnce)
	setupServerTestDataEr error
)

func setupServerTestDataOnce() {
	root, err := os.MkdirTemp("", "vrcdp-mixed-server-test")
	if err != nil {
		setupServerTestDataEr = err
		return
	}

	custom_fyne.AppDataRoot = root

	setupServerTestDataEr = persistence.InitMainDB()
}

// newServerForTest builds a mixedServer backed by a real preloader, without
// starting a listener.
func newServerForTest(t *testing.T) *mixedServer {
	t.Helper()

	setupServerTestData()
	if setupServerTestDataEr != nil {
		t.Fatalf("set up the test environment: %v", setupServerTestDataEr)
	}

	cacheSvc := cache_manager.New(cache_manager.Config{
		Path:            t.TempDir(),
		MaxVideoCache:   64,
		VideoFileFormat: 1,
	})
	if err := cacheSvc.ServiceStart(); err != nil {
		t.Fatalf("start the cache manager: %v", err)
	}
	t.Cleanup(func() { _ = cacheSvc.ServiceStop() })

	preloaderSvc := preloader.New(preloader.Config{
		EnabledRooms: []string{preloader.PyPyDanceRoomName},
		MaxPreload:   2,
	}, cacheSvc, downloader.New(downloader.DefaultConfig()))

	return newMixedServer(DefaultConfig(), &Service{preloaderSvc: preloaderSvc})
}

// The pass-through has to actually deliver the origin's response: it is the only
// thing the player gets while a queued temporary video is not cached yet, and
// the old behavior (wait for the queue, then give up) is what it replaces.
func TestPassThroughForwardsTheOriginResponse(t *testing.T) {
	srv := newServerForTest(t)

	body := []byte("a video body the origin serves")
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Range"); got != "bytes=0-3" {
			t.Errorf("the origin saw Range %q, want the client's range forwarded", got)
		}

		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(body)
	}))
	defer origin.Close()

	req := httptest.NewRequest(http.MethodGet, "http://"+origin.Listener.Addr().String()+"/videos/1.mp4", nil)
	req.Host = origin.Listener.Addr().String()
	req.Header.Set("Range", "bytes=0-3")

	// A channel that never closes: the entry stays uninitialized, which is the
	// stage the pass-through exists for.
	entryInitialized := make(chan struct{})

	recorder := newSyncRecorder()

	if !srv.handlePassThrough(recorder, req, "pypy_1", entryInitialized) {
		t.Fatal("handlePassThrough refused a request that names an upstream")
	}

	status, header, forwarded := recorder.snapshot()

	if forwarded != string(body) {
		t.Fatalf("body = %q, want the origin's %q", forwarded, body)
	}
	if status != http.StatusPartialContent {
		t.Fatalf("status = %d, want the origin's %d", status, http.StatusPartialContent)
	}
	if got := header.Get("Content-Type"); got != "video/mp4" {
		t.Fatalf("Content-Type = %q, want the origin's", got)
	}
}

// The CONNECT path hands the handler a writer that puts the status line and the
// headers straight onto the client connection, and the body bytes right after
// them (WriterGivenRespWriter). Declaring such a response chunked is only honest
// if the bytes are framed as chunks, so this asserts what the player does: the
// response parses as HTTP and its body can be read to the end.
//
// The other pass-through tests use a recorder, which never frames anything, so
// they cannot see the difference.
func TestPassThroughOnTheConnectWriterKeepsTheResponseWellFormed(t *testing.T) {
	srv := newServerForTest(t)

	body := []byte("a video body the origin serves")
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(body)
	}))
	defer origin.Close()

	req := httptest.NewRequest(http.MethodGet, "http://"+origin.Listener.Addr().String()+"/videos/1.mp4", nil)
	req.Host = origin.Listener.Addr().String()

	var raw bytes.Buffer

	// A channel that never closes: the origin body ends by itself, so the
	// pass-through finishes without a hand-off.
	if !srv.handlePassThrough(NewWriterGivenRespWriter(&raw), req, "pypy_1", make(chan struct{})) {
		t.Fatal("handlePassThrough refused a request that names an upstream")
	}

	resp, err := http.ReadResponse(bufio.NewReader(&raw), req)
	if err != nil {
		t.Fatalf("the client cannot parse the pass-through response: %v\n--- raw response ---\n%s", err, raw.String())
	}
	defer resp.Body.Close()

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("the client cannot read the pass-through body: %v (status=%d, TransferEncoding=%v)\n--- raw response ---\n%s", err, resp.StatusCode, resp.TransferEncoding, raw.String())
	}
	if string(got) != string(body) {
		t.Fatalf("body = %q, want %q", got, body)
	}
}

// The hand-off: as soon as the cache entry can serve the video, the pass-through
// has to stop copying. It is the player that turns that into a re-request, which
// is then answered by the cache instead of the origin.
//
// The body is long and written slowly, so the copy cannot end by itself inside
// the window this test allows: the pass-through can only return because the
// hand-off interrupted it.
func TestPassThroughStopsWhenTheEntryIsInitialized(t *testing.T) {
	srv := newServerForTest(t)

	const totalChunks = 1000
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		for i := 0; i < totalChunks; i++ {
			select {
			case <-r.Context().Done():
				return
			default:
			}

			_, _ = w.Write([]byte("first chunk"))

			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

			time.Sleep(time.Millisecond)
		}
	}))
	defer origin.Close()

	req := httptest.NewRequest(http.MethodGet, "http://"+origin.Listener.Addr().String()+"/videos/1.mp4", nil)
	req.Host = origin.Listener.Addr().String()

	entryInitialized := make(chan struct{})
	// The body is written from the copy goroutine and read by the polling
	// below, so the recorder is wrapped in something that synchronises the two.
	recorder := newSyncRecorder()

	done := make(chan bool, 1)
	go func() {
		done <- srv.handlePassThrough(recorder, req, "pypy_1", entryInitialized)
	}()

	// Wait until the first chunk has been forwarded, then hand over.
	waitForBody(t, recorder, "first chunk")
	close(entryInitialized)

	select {
	case handled := <-done:
		if !handled {
			t.Fatal("handlePassThrough reported that it served nothing")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the pass-through did not stop when the cache entry became initialized")
	}
}

// syncRecorder is an http.ResponseWriter whose state can be read while the
// response is still being written. httptest.ResponseRecorder is not safe for
// that, and the pass-through writes from a goroutine of its own.
type syncRecorder struct {
	mu sync.Mutex

	header http.Header
	body   strings.Builder
	status int
}

func newSyncRecorder() *syncRecorder {
	return &syncRecorder{header: make(http.Header)}
}

func (s *syncRecorder) Header() http.Header {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.header
}

func (s *syncRecorder) WriteHeader(statusCode int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == 0 {
		s.status = statusCode
	}
}

func (s *syncRecorder) Write(data []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.status == 0 {
		s.status = http.StatusOK
	}

	return s.body.Write(data)
}

// snapshot returns the status, the header and the body as they are right now.
func (s *syncRecorder) snapshot() (int, http.Header, string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.status, s.header, s.body.String()
}

// waitForBody polls the recorder until its response holds the expected text.
func waitForBody(t *testing.T, recorder *syncRecorder, want string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, _, body := recorder.snapshot(); strings.Contains(body, want) {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}

	_, _, body := recorder.snapshot()
	t.Fatalf("timed out waiting for the origin body %q, got %q", want, body)
}
