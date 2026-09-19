package local_executables

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// bareExecutableName puts a runnable file in a fresh directory, adds that
// directory to PATH, and returns the *bare* name.
//
// A bare name is what makes the difference visible: LookPath resolves it to an
// absolute path, and that resolution is exactly what Valid() used to write back
// into d.Path.
func bareExecutableName(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	name := "vrcdp-valid-test"

	if runtime.GOOS == "windows" {
		// LookPath only finds Windows executables with an extension on PATHEXT.
		name += ".bat"
	}

	if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0o755); err != nil {
		t.Fatalf("create the test executable: %v", err)
	}

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	return name
}

// Valid() is reached under the read lock, so it must not write to the receiver.
//
// It used to rewrite d.Path with the path LookPath found. Under the read lock
// that is a data race with every other reader: two concurrent Execute calls, or
// an Execute racing SetPathAndCheck, wrote the same field.
func TestValidOnlyReadsTheBinary(t *testing.T) {
	name := bareExecutableName(t)

	d := &DownloadableBinary{Path: name}

	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if !d.Valid() {
		t.Fatalf("Valid() = false for %q, which PATH resolves to a real file", name)
	}

	if d.Path != name {
		t.Fatalf("Valid() rewrote Path from %q to %q while only the read lock is held", name, d.Path)
	}
}

// The same property under concurrency, which is what -race used to report
// through Execute: several readers of one binary.
//
// Run with -race for this to mean anything.
func TestConcurrentReadersDoNotRaceOnThePath(t *testing.T) {
	name := bareExecutableName(t)

	d := &DownloadableBinary{Path: name}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			d.mutex.RLock()
			defer d.mutex.RUnlock()

			if !d.Valid() {
				t.Errorf("Valid() = false for %q", name)
			}
		}()
	}

	wg.Wait()
}

// SetPathAndCheck is where the resolved path may be cached, because it holds the
// write lock. The cached form is what reaches yt-dlp and the rename target.
func TestSetPathAndCheckCachesTheResolvedPath(t *testing.T) {
	name := bareExecutableName(t)

	d := &DownloadableBinary{Path: name}
	d.SetPathAndCheck(name)

	if !strings.HasSuffix(d.Path, name) {
		t.Fatalf("Path = %q, want it resolved and cached as an absolute path", d.Path)
	}
	if !filepath.IsAbs(d.Path) {
		t.Fatalf("Path = %q, want an absolute path", d.Path)
	}

	// With no path there is nothing to resolve, and Path is left as given.
	empty := &DownloadableBinary{}
	empty.SetPathAndCheck("")
	if empty.Path != "" {
		t.Fatalf("Path = %q, want the empty path to stay empty", empty.Path)
	}
}
