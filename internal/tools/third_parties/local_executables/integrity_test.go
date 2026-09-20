package local_executables

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// stubIcacls replaces the icacls runner and returns a function reporting how
// often it was called.
//
// The real call reads (and, for a Low level, rewrites) the ACLs of a file on the
// machine running the test, so this seam is what makes "was the executable
// inspected?" observable at all.
func stubIcacls(t *testing.T) func() int {
	t.Helper()

	var mu sync.Mutex
	calls := 0

	previous := runIcacls
	runIcacls = func(ctx context.Context, args ...string) ([]byte, error) {
		mu.Lock()
		calls++
		mu.Unlock()

		// A healthy executable: the check stops after reading the level and never
		// tries to raise it.
		return []byte(`Mandatory Label\Medium Mandatory Level`), nil
	}

	t.Cleanup(func() { runIcacls = previous })

	return func() int {
		mu.Lock()
		defer mu.Unlock()

		return calls
	}
}

// installTestBinary puts a file that LookPath accepts where the test wants it.
//
// Only the extension matters: the child itself never starts, because everything
// this test asserts happens before that.
func installTestBinary(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create the binaries directory: %v", err)
	}
	if err := os.WriteFile(path, []byte("not a real executable"), 0o755); err != nil {
		t.Fatalf("install the test binary at %s: %v", path, err)
	}
}

// C1-② — the integrity level is inspected before the executable is started, and
// no longer while SetPathAndCheck holds the write lock.
//
// A Low integrity executable is refused by Windows the moment it starts, so the
// check is what makes yt-dlp work at all — but it runs up to two icacls processes
// with a 3s timeout each, and SetPathAndCheck holds the write lock that the GUI
// and the video request path wait on. The check therefore moved to the exec path,
// where it is guaranteed before the child runs and paid once per installed file.
func TestExecuteInspectsTheIntegrityLevelBeforeStarting(t *testing.T) {
	root := isolateAppData(t)

	binaryPath := filepath.Join(root, "binaries", "vrcdp-integrity-test.exe")
	installTestBinary(t, binaryPath)

	calls := stubIcacls(t)

	d := NewDownloadableBinary("ytdlp")
	d.SetPathAndCheck(binaryPath)

	if got := calls(); got != 0 {
		t.Fatalf("inspections = %d after SetPathAndCheck, want 0: the check must not run under its write lock", got)
	}

	// The child fails to start, which is fine: the inspection has to happen before
	// the attempt either way.
	if _, err := d.Execute(context.Background(), "--version"); err == nil {
		t.Fatal("Execute succeeded on a file that is not an executable")
	}

	if got := calls(); got != 1 {
		t.Fatalf("inspections = %d after Execute, want the executable inspected exactly once before it was started", got)
	}

	if _, err := d.Execute(context.Background(), "--version"); err == nil {
		t.Fatal("Execute succeeded on a file that is not an executable")
	}

	if got := calls(); got != 1 {
		t.Fatalf("inspections = %d after a second Execute, want the already inspected file to be left alone", got)
	}

	// A different path is a different file, and nothing has inspected that one.
	secondPath := filepath.Join(root, "binaries", "vrcdp-integrity-test-2.exe")
	installTestBinary(t, secondPath)

	d.SetPathAndCheck(secondPath)

	if _, err := d.Execute(context.Background(), "--version"); err == nil {
		t.Fatal("Execute succeeded on a file that is not an executable")
	}

	if got := calls(); got != 2 {
		t.Fatalf("inspections = %d after the path changed, want the new file inspected once as well", got)
	}
}
