package preloader

import (
	"net/http"
	"testing"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/internal_id"
)

// The two download states the song does not retry must not leave their task in
// the download queue.
//
// Disabled and Refused are terminal: the song state machine returns from its
// download loop on both, and nothing will ever run the task again. The manager
// only drops *completed* tasks from its queue (UpdatePriorities), so a task that
// ended in one of these states used to stay there — at the head of the queue,
// holding one of the maxParallel slots — and every song behind it never got a
// permit. That is the "zombie at the head of the queue" that starved the rest of
// the playlist until the song was removed.
//
// These tests pin the fix from the song side: the terminal branches cancel the
// task, and cancelling is what takes it out of the manager. The retry branch is
// deliberately not covered here — a task waiting for its retry keeps its slot on
// purpose, and that is asserted in the downloader's own queue tests.
//
// Both tests need the whole binding path (a real cache session, a real download
// manager, the real provider of the id), so they share this package's fixture.
// The failure itself is deterministic and offline: the provider is put in a
// state that makes it refuse, which is a real branch of its resolve function,
// not an injected error.

// terminalWait bounds the wait for the download to fail. It is larger than
// waitFor's default because the providers schedule their requests: PyPy leaves
// five seconds between two of them, and the video resolve waits for the slot the
// song's own catalog request took.
const terminalWait = 30 * time.Second

// waitForTerminal polls cond until it holds, or fails the test. It is waitFor
// with a bound of its own; see terminalWait.
func waitForTerminal(t *testing.T, cond func() bool, what string) {
	t.Helper()

	deadline := time.Now().Add(terminalWait)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", what)
}

// requireTerminalTask starts the download of id and asserts that the manager
// holds its task before waiting for the song to reach want and for the task to
// be released.
func requireTerminalTask(t *testing.T, svc *Service, id string, want song.DownloadStatus) {
	t.Helper()

	item := song.CreateStatefulSongByInternalId(id)
	t.Cleanup(item.Destroy)

	// The task exists as soon as bind has run, which makeSureDownloading does
	// synchronously. Asserting it here keeps the test honest: without a task in
	// the manager there would be nothing to release, and the "it is gone" check
	// below would pass for the wrong reason.
	svc.makeSureDownloading(item)

	if !svc.downloaderSvc.HoldsTaskForTest(id) {
		t.Fatalf("the download manager does not hold a task for %s, so there is nothing to release", id)
	}

	waitForTerminal(t, func() bool {
		return item.StateMachine().DownloadStatus == want
	}, "the song to reach the terminal download state")

	// The definitive evidence of the leak: the manager still holds the task
	// after the download loop has returned.
	waitForTerminal(t, func() bool {
		return !svc.downloaderSvc.HoldsTaskForTest(id)
	}, "the download manager to drop the task of a song that will not be retried")
}

// TestTerminalDownloadDisabledReleasesTheTask — ErrFeatureDisabled (the user
// turned the platform off) ends in Disabled, and the task must be gone.
func TestTerminalDownloadDisabledReleasesTheTask(t *testing.T) {
	requireRaceCleanTools(t)

	startTestTools()

	// The provider is registered with video allowed (third_parties.DefaultConfig);
	// taking it back out is what makes resolveVideoUrl return
	// errYouTubeVideoDisabled, which wraps ErrFeatureDisabled.
	const id = internal_id.YtInternalPrefix + "dcancel1"

	provider := third_parties.GetProviderById(id)
	if provider == nil {
		t.Fatalf("no provider is registered for %s", id)
	}
	provider.SetAllowResources(nil)

	requireTerminalTask(t, newPreloadFixture(t), id, song.Disabled)
}

// TestTerminalDownloadRefusedReleasesTheTask — a refused request (the CDN
// answers with a redirection instead of the video) ends in Refused, and the
// task must be gone.
func TestTerminalDownloadRefusedReleasesTheTask(t *testing.T) {
	requireRaceCleanTools(t)

	// newPreloadFixture installs the strict mock (through startTestTools) first,
	// and startTestTools runs only once per test binary, so the rules below have
	// to be installed after it to survive.
	svc := newPreloadFixture(t)

	// The redirection is what makes directResolve report ErrRefused, and the
	// Location deliberately points at www.youtube.com: the client is configured
	// to hand back the response instead of following that one, because that is
	// how an intercepted YouTube request is detected in production. The
	// everything-else rule answers the catalog request: the fixture's mock
	// refuses unmatched requests instead of reaching the network, and 404 is
	// unrecoverable for a catalog, so it fails once instead of retrying.
	requesting.SetMock(requesting.NewMockTransport(
		requesting.MockRule{
			Name:   "pypy-video-refused",
			Method: http.MethodGet,
			Host:   "api.pypy.dance",
			Path:   "/video",
			Status: http.StatusFound,
			Header: http.Header{"Location": []string{"https://www.youtube.com/watch?v=refused"}},
		},
		requesting.MockRule{Name: "everything-else", Status: http.StatusNotFound},
	))

	const id = internal_id.PyPyInternalPrefix + "9"

	requireTerminalTask(t, svc, id, song.Refused)
}
