package trunk

import (
	"os"
	"runtime"
	"testing"
)

// A1 — the err check used to be inverted, so the success path never read the
// creation time (the branch was only taken on failure) and the failure path
// called Sys() on a nil FileInfo.
func TestStatReadsCreationTimeOnSuccess(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "trunk")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	file := &File{file: f, FullSize: 4096}

	size, created := file.Stat()

	if want := bodyOffset + 4096; size != want {
		t.Fatalf("size = %d, want %d", size, want)
	}

	// The creation time comes from the Win32 attribute data, which only exists
	// on Windows; elsewhere the zero value is the documented outcome.
	if runtime.GOOS == "windows" && created.IsZero() {
		t.Fatal("creation time is zero: the success path was skipped again")
	}
}

// A1 — a failing Stat must return, not dereference the nil FileInfo. A closed
// file is the cheapest way to make it fail.
func TestStatSurvivesAStatError(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "trunk")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	file := &File{file: f, FullSize: 4096}

	size, created := file.Stat()

	if want := bodyOffset + 4096; size != want {
		t.Fatalf("size = %d, want %d", size, want)
	}
	if !created.IsZero() {
		t.Fatalf("created = %v, want the zero time", created)
	}
}
