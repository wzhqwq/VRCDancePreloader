package local_executables

import (
	"os"
	"path/filepath"
	"testing"
)

// makeInstalledBinary creates a file that looks like an installed binary and
// points a fresh DownloadableBinary at it.
func makeInstalledBinary(t *testing.T) *DownloadableBinary {
	t.Helper()

	path := filepath.Join(t.TempDir(), "vrcdp-remove-test.exe")
	if err := os.WriteFile(path, []byte("binary"), 0o755); err != nil {
		t.Fatalf("create the binary: %v", err)
	}

	b := NewDownloadableBinary("ytdlp")
	b.SetPathAndCheck(path)

	if b.Path() == "" {
		t.Fatalf("SetPathAndCheck(%q) did not resolve a path", path)
	}

	return b
}

// A binary that is being downloaded must not be deleted.
//
// The delete targets the installed binary, while DownloadAndReplace is about to
// rename the freshly downloaded one into that same path: a delete that lands
// first is silently undone by the rename. The refusal has to be visible, so the
// caller gets an error it can show.
func TestRemoveIsRefusedWhileDownloading(t *testing.T) {
	b := makeInstalledBinary(t)

	if !b.trySetState(idleStates, BinDownloading) {
		t.Fatal("could not move the binary into BinDownloading")
	}

	b.Remove()

	if _, err := os.Stat(b.Path()); err != nil {
		t.Fatalf("Remove deleted a binary that is being downloaded: %v", err)
	}
	if b.Err() == nil {
		t.Fatal("Remove refused silently: the caller has nothing to show")
	}
}

// Once nothing owns the state machine the delete goes through again, so the
// refusal is about the download and not about Remove in general.
func TestRemoveWorksWhenNothingOwnsTheMachine(t *testing.T) {
	b := makeInstalledBinary(t)

	path := b.Path()

	b.Remove()

	if _, err := os.Stat(path); err == nil {
		t.Fatal("Remove left the file in place")
	}
}
