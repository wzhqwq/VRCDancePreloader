package preloader

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/trunk"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_fs"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/secrets"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/catalog"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/local_executables"
)

// preloadQueueSlot is the manager every id below resolves to. The preloader hands
// each id to every manager, and a manager drops the ids it has no task for, so
// which one holds the queue does not matter as long as the assertions look at
// the same one.
const preloadQueueSlot = "pypy"

// The preloader reaches the song and cache layers, and those are process wide:
// the database and the third party providers are brought up once, under a
// temporary root, and are never isolated per test. Reopening them per test would
// only leak a second database handle and start the providers' loops twice.
var (
	testDataRoot    string
	setupTestData   = sync.OnceFunc(setupTestDataOnce)
	setupTestDataEr error
	startTestTools  = sync.OnceFunc(startToolsOnce)
)

func setupTestDataOnce() {
	testDataRoot, setupTestDataEr = os.MkdirTemp("", "vrcdp-preloader-test")
	if setupTestDataEr != nil {
		return
	}

	custom_fyne.AppDataRoot = testDataRoot

	if err := persistence.InitMainDB(); err != nil {
		setupTestDataEr = err
	}
}

// startToolsOnce brings up the tools the third party providers depend on, in the
// order the host wires them, and leaves them running.
//
// The providers have to exist for a song to be constructed and for a download
// task to resolve its remote; their loops subscribe to the tools below, so those
// have to be running first. Shutting them down between tests is not safe: their
// loops run in goroutines that no test joins, and Shutdown then races with a
// loop that is still touching the tool it is closing.
//
// The mock answers the availability self checks and the catalog requests, so the
// tests never touch the network.
func startToolsOnce() {
	requesting.SetMock(requesting.NewMockTransport(requesting.MockAvailabilityRules()...))

	requesting.New(requesting.DefaultConfig()).Start()
	// secrets.DefaultConfig reads the OS keyring; the empty config is the same
	// thing without touching it.
	secrets.New(secrets.Config{}).Start()
	local_executables.New(local_executables.DefaultConfig()).Start()
	catalog.New().Start()
	third_parties.New(third_parties.DefaultConfig()).Start()
}

// newPreloadFixture builds a preloader with a real cache and a real download
// manager, without running ServiceStart (which would also start a retry ticker).
func newPreloadFixture(t *testing.T) *Service {
	t.Helper()

	setupTestData()
	if setupTestDataEr != nil {
		t.Fatalf("set up the test environment: %v", setupTestDataEr)
	}

	startTestTools()

	cacheSvc := cache_manager.New(cache_manager.Config{
		Path:            t.TempDir(),
		MaxVideoCache:   64,
		VideoFileFormat: 1,
	})
	if err := cacheSvc.ServiceStart(); err != nil {
		t.Fatalf("start the cache manager: %v", err)
	}
	t.Cleanup(func() { _ = cacheSvc.ServiceStop() })

	downloaderSvc := downloader.New(downloader.DefaultConfig())
	downloaderSvc.InstallManagersForTest(preloadQueueSlot, "default")

	return New(Config{
		EnabledRooms: []string{PyPyDanceRoomName},
		MaxPreload:   2,
	}, cacheSvc, downloaderSvc)
}

// newPlaylistWith makes a playlist of the given ids.
func newPlaylistWith(t *testing.T, ids ...string) *playlist.PlayList {
	t.Helper()

	items := make([]*song.StatefulSong, 0, len(ids))
	for _, id := range ids {
		items = append(items, song.CreateStatefulSongByInternalId(id))
	}

	pl := &playlist.PlayList{Items: items}
	t.Cleanup(func() {
		for _, item := range pl.GetItemsSnapshot() {
			item.Destroy()
		}
	})

	return pl
}

// A late request for a video must not overtake the one that is playing.
//
// The queue is in the state the preload loop leaves it in: a task per upcoming
// song, the current one first. A request for the next song — which is what the
// room sends when it is about to be played — must not push the current one out
// of the only download slot.
func TestRequestDoesNotOvertakeTheCurrentSong(t *testing.T) {
	requireRaceCleanTools(t)

	svc := newPreloadFixture(t)
	pl := newPlaylistWith(t, "pypy_1", "pypy_2")
	svc.UsePlaylistForTest(pl)

	// The queue as the preload loop builds it: a task per song, current first.
	svc.downloaderSvc.WithFrozenQueue(func() {
		for _, item := range pl.GetItemsSnapshot() {
			svc.makeSureDownloading(item)
		}
		svc.downloaderSvc.Prioritize("pypy_1")
	})

	// A playlist song binds its task from a goroutine of its own, so the queue
	// is only complete once both songs are in it.
	waitFor(t, func() bool {
		return len(svc.downloaderSvc.QueueOrderForTest()) >= 2
	}, "both songs to be queued")

	order, ok := svc.RequestQueueOrderForTest("pypy_2")
	if !ok {
		t.Fatal("the requested video is in the playlist, so the request must not have gone to the origin")
	}

	if len(order) < 2 || order[0] != "pypy_1" || order[1] != "pypy_2" {
		t.Fatalf("queue order = %v, want the current song (pypy_1) before the requested one (pypy_2)", order)
	}

	allowed := permitsOf(t, svc)

	if !allowed["pypy_1"] {
		t.Fatalf("the current song lost its permit while the request was served (permits %v)", allowed)
	}
	if allowed["pypy_2"] {
		t.Fatalf("the requested song took the only permit from the current song (permits %v)", allowed)
	}
}

// A video that is not in the playlist is served from the origin instead of being
// given a queue position in front of the playlist.
//
// The random-play queue cannot be read, so such a request is legitimate — but it
// is also not something the room is waiting on, so the download it gets stays at
// the tail and the request itself is answered from the origin.
func TestRequestForAnUnknownVideoPassesThrough(t *testing.T) {
	requireRaceCleanTools(t)

	svc := newPreloadFixture(t)
	pl := newPlaylistWith(t, "pypy_1", "pypy_2")
	svc.UsePlaylistForTest(pl)

	// The state the preload loop leaves behind, so that the unknown video has
	// something real to be queued behind.
	svc.downloaderSvc.WithFrozenQueue(func() {
		for _, item := range pl.GetItemsSnapshot() {
			svc.makeSureDownloading(item)
		}
		svc.downloaderSvc.Prioritize("pypy_1")
	})

	waitFor(t, func() bool {
		return len(svc.downloaderSvc.QueueOrderForTest()) >= 2
	}, "both playlist songs to be queued")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := svc.Request("pypy_999", ctx)
	if !errors.Is(err, ErrPassThrough) {
		t.Fatalf("Request(pypy_999) = %v, want ErrPassThrough", err)
	}

	order := svc.downloaderSvc.QueueOrderForTest()

	if len(order) == 0 || order[0] != "pypy_1" {
		t.Fatalf("queue order = %v, want the current song still first", order)
	}
	if len(order) > 0 && order[len(order)-1] != "pypy_999" {
		t.Fatalf("queue order = %v, want the unknown video at the tail", order)
	}

	allowed := permitsOf(t, svc)

	if !allowed["pypy_1"] {
		t.Fatalf("the unknown video took the permit from the current song (permits %v)", allowed)
	}
	if allowed["pypy_999"] {
		t.Fatalf("the unknown video was authorized (permits %v)", allowed)
	}
}

// A video that is already fully cached is served from the cache: that is the
// best answer available, and it costs the queue nothing.
func TestRequestForAnUnknownVideoUsesACompleteCache(t *testing.T) {
	requireRaceCleanTools(t)

	svc := newPreloadFixture(t)
	svc.UsePlaylistForTest(newPlaylistWith(t, "pypy_1"))

	const id = "pypy_777"
	writeCompleteCache(t, svc, id)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resource, err := svc.Request(id, ctx)
	if err != nil {
		t.Fatalf("Request(%s) = %v, want the cached video", id, err)
	}
	if resource == nil {
		t.Fatalf("Request(%s) returned no resource for a complete cache entry", id)
	}

	for _, queued := range svc.downloaderSvc.QueueOrderForTest() {
		if queued == id {
			t.Fatalf("a fully cached video was queued for download")
		}
	}
}

// permitsOf reads the permit published for each queued task.
func permitsOf(t *testing.T, svc *Service) map[string]bool {
	t.Helper()

	allowed := make(map[string]bool)

	for _, permit := range svc.downloaderSvc.ManagerNamedForTest(preloadQueueSlot).PermitsForTest() {
		allowed[permit.ID] = permit.Allowed
	}

	return allowed
}

// requireRaceCleanTools skips a test whose fixture has to start the third party
// tool (and therefore a download loop) when the race detector is on.
//
// Starting that tool is not race clean today, and the detector flags packages
// outside this one rather than the preloader:
//
//   - internal/song: StateMachine.Destroy writes DownloadStatus (state_machine.go
//     :287) while the download loop goroutine is still running its own
//     SwitchDownloadStatus/StartDownloadLoop. That is the "song concurrency"
//     batch, not this package.
//
// The third party races that used to be listed here are fixed: PlatformProvider
// .mode and BaseProvider.allowResources are atomics now, and every platform
// provider starts a single loop (the constructor no longer starts one).
//
// A pre-existing defect of another package must not be reported as a failure
// here, so the tests that need that tool are skipped under -race. They run in the
// ordinary build.
func requireRaceCleanTools(t *testing.T) {
	t.Helper()

	if raceEnabled {
		t.Skip("skipping under -race: internal/song races its own state machine while a download loop runs (StateMachine.Destroy vs SwitchDownloadStatus)")
	}
}

// waitFor polls cond until it holds, or fails the test.
func waitFor(t *testing.T, cond func() bool, what string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", what)
}

// writeCompleteCache writes a complete cache file for id, the way the cache
// backend writes one, so that serving a request for it needs no download.
func writeCompleteCache(t *testing.T, svc *Service, id string) {
	t.Helper()

	cfs, err := cache_fs.New(svc.cacheSvc.Cfg.Path)
	if err != nil {
		t.Fatalf("open the cache directory: %v", err)
	}

	file := trunk.NewTrunkFile("video$"+id, cfs)
	if file == nil {
		t.Fatalf("could not create the cache file of %s", id)
	}
	if err := file.Init(1024, time.Unix(0, 0)); err != nil {
		t.Fatalf("initialize the cache file of %s: %v", id, err)
	}
	file.MarkCompleted()
	if err := file.Close(); err != nil {
		t.Fatalf("close the cache file of %s: %v", id, err)
	}
}
