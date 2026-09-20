package local_executables

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader/task"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/api"
)

// isolateAppData points the binaries directory at a fresh temporary root, so
// that a test can neither read nor write the real one.
func isolateAppData(t *testing.T) string {
	t.Helper()

	previous := custom_fyne.AppDataRoot
	root := t.TempDir()
	custom_fyne.AppDataRoot = root

	t.Cleanup(func() { custom_fyne.AppDataRoot = previous })

	return root
}

// B-a — every accessor reads a field that the download goroutine writes, so they
// all have to share the lock with the writers.
//
// Run with -race for this to mean anything: without the lock the two sides have
// no happens-before edge at all, no matter how the two loops interleave.
func TestAccessorsAreRaceFreeWhileTheStateChanges(t *testing.T) {
	d := NewDownloadableBinary("ytdlp")

	stop := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < 4; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-stop:
					return
				default:
				}

				_ = d.State()
				_ = d.Err()
				_ = d.Info()
				_ = d.Release()
				_ = d.Path()
				_ = d.Task()
				_ = d.HasUpdates()
				_ = d.UpdateText()
			}
		}()
	}

	states := []DownloadableState{
		BinCheckingLocal, BinInitial, BinCheckingUpdates,
		BinUpdateAvailable, BinDownloading, BinDownloaded,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(stop)

		for i := 0; i < 200; i++ {
			for _, state := range states {
				d.setState(state)
			}

			d.setErr(errors.New("transient failure"))
			d.setRelease(&api.BriefRelease{Version: "1.2.3"})
			d.setInfo(BinaryInfo{Exists: true, Version: "1.2.2", Size: 42})
		}
	}()

	wg.Wait()
}

// B-b — Upgrade must not start a download while another pass owns the state
// machine.
//
// The old entry points read the state and then wrote it in two steps, so the
// question they answered ("is anything else running?") and the answer they gave
// ("I am") were not one critical section. The states that were missing from the
// old guard are the interesting ones: a version check in flight, and the window
// in which a finished download waits for the rename that installs it.
//
// The download itself is made to fail immediately, and without touching the
// network: the binaries directory is a regular file, so MkdirAll cannot create
// the download directory. A download that did start is therefore observable as a
// recorded error plus a move to BinUpdateAvailable.
func TestUpgradeIsNotReentrant(t *testing.T) {
	root := isolateAppData(t)

	if err := os.WriteFile(filepath.Join(root, "binaries"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("create the file that stands in for the binaries directory: %v", err)
	}

	for _, owner := range []DownloadableState{BinCheckingUpdates, BinDownloading, BinDownloaded} {
		t.Run(string(owner), func(t *testing.T) {
			d := NewDownloadableBinary("ytdlp")

			d.mutex.Lock()
			d.release = &api.BriefRelease{
				GitHubReleaseAsset: api.GitHubReleaseAsset{
					Name:               "tool.exe",
					BrowserDownloadURL: "https://example.invalid/tool.exe",
				},
				Version: "9.9.9",
			}
			d.state = owner
			d.mutex.Unlock()

			d.Upgrade()

			if got := d.State(); got != owner {
				t.Fatalf("State = %q after Upgrade while %q owned the machine, want it left alone", got, owner)
			}
			if err := d.Err(); err != nil {
				t.Fatalf("Upgrade started a download while %q owned the machine: %v", owner, err)
			}
		})
	}
}

// B-c — CancelDownload reads the task pointer that downloadFile clears.
//
// downloadFile publishes it under the write lock and clears it under the write
// lock again when the download is over, so the reader has to take the lock as
// well. The loop below reproduces that writer (the same two locked stores) many
// times while this goroutine cancels: with the read left unsynchronised, -race
// reports the pair, which is the failure this test is about. Run with -race.
func TestCancelDownloadReadsTheTaskUnderTheLock(t *testing.T) {
	d := NewDownloadableBinary("ytdlp")

	// Nothing in flight: the deferred cleanup has already cleared the pointer,
	// which is a state the GUI can still click "cancel" in, and it must not panic.
	d.CancelDownload()

	// Cancel only touches the task, so a task without providers is enough here.
	inFlight := task.NewTask("tool.exe (tool)", nil, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)

		for i := 0; i < 500; i++ {
			d.mutex.Lock()
			d.task = inFlight
			d.mutex.Unlock()

			d.mutex.Lock()
			d.task = nil
			d.mutex.Unlock()
		}
	}()

	for i := 0; i < 500; i++ {
		d.CancelDownload()
	}

	<-done

	// And the pointer is really gone once the download is over.
	d.mutex.Lock()
	left := d.task
	d.mutex.Unlock()

	if left != nil {
		t.Fatalf("task = %v, want it cleared by the download cleanup", left)
	}
}

// B-d — the local probe must not run while a download owns the state machine.
//
// BinDownloaded is the window between unpacking the new executable and renaming
// it into place. Init used to be refused only during a version check and during
// a download, so Remove (which calls Init) could probe there and overwrite the
// finished download with what the *old* path still resolves to — here, nothing
// installed at all.
func TestInitSkipsWhileDownloaded(t *testing.T) {
	isolateAppData(t)

	d := NewDownloadableBinary("ytdlp")

	d.mutex.Lock()
	d.info = BinaryInfo{Exists: true, Version: "old"}
	d.state = BinDownloaded
	d.mutex.Unlock()

	d.Init()

	if got := d.State(); got != BinDownloaded {
		t.Fatalf("State = %q after Init while a downloaded file waited for its rename, want BinDownloaded", got)
	}

	if got := d.Info(); !got.Exists || got.Version != "old" {
		t.Fatalf("Info = %+v, want the probe to have been skipped", got)
	}
}

// B-e — the questions that read several fields at once must read them in one
// critical section.
//
// HasUpdates and UpdateText need release, state and info together. Composing them
// out of the accessors would take the read lock twice, and a writer queued
// between the two acquisitions turns the second RLock into a deadlock — which is
// what the writer below is there to provoke. The bounded wait reports it instead
// of hanging the test binary forever.
func TestCompositeReadersDoNotNestTheLock(t *testing.T) {
	d := NewDownloadableBinary("ytdlp")

	d.mutex.Lock()
	d.release = &api.BriefRelease{Version: "1.0.1"}
	d.info = BinaryInfo{Exists: true, Version: "1.0.0", Size: 10}
	d.state = BinUpdateAvailable
	d.mutex.Unlock()

	stop := make(chan struct{})
	var writer sync.WaitGroup

	writer.Add(1)
	go func() {
		defer writer.Done()

		for {
			select {
			case <-stop:
				return
			default:
			}

			d.mutex.Lock()
			d.mutex.Unlock()
		}
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)

		for i := 0; i < 20000; i++ {
			if !d.HasUpdates() {
				t.Error("HasUpdates = false, want true while the release is newer than the local version")
				return
			}
			if text := d.UpdateText(); text != "→ 1.0.1" {
				t.Errorf("UpdateText = %q, want %q", text, "→ 1.0.1")
				return
			}
		}
	}()

	select {
	case <-done:
		close(stop)
		writer.Wait()
	case <-time.After(10 * time.Second):
		t.Fatal("HasUpdates/UpdateText did not return: the composite reader takes the read lock a second time, and a queued writer deadlocks it")
	}
}
