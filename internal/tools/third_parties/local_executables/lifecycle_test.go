package local_executables

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The pump must answer from the version, not from the kind of event it received.
//
// Subscribe delivers DownloadableChange values (BinVersion, BinState,
// BinProgress), so the old `ver != ""` test was true for every event and reported
// yt-dlp as available as soon as anything happened — even with no yt-dlp at all.
func TestAvailabilityPumpReadsTheVersionNotTheEventKind(t *testing.T) {
	if err := initialize(); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	t.Cleanup(func() { _ = destroy() })

	// initialize ran with the zero Config, so no binary was probed and there is no
	// version to report on this machine either way.
	if Get("ytdlp").Info.Version != "" {
		t.Skip("this environment resolved a yt-dlp version; the pump cannot be tested")
	}

	Get("ytdlp").em.NotifySubscribers(BinVersion)

	// The pump runs in its own goroutine, so poll instead of asserting once: the
	// wrong answer must never appear.
	deadline := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(deadline) {
		if YtDlpAvailable() {
			t.Fatal("the pump reported yt-dlp as available after an event, although there is no version")
		}
	}
}

// The version parser handles external release metadata, so a value it cannot
// represent is reported rather than panicked.
func TestParseYtDlpVersionReportsUnparsableNumbers(t *testing.T) {
	for _, tag := range []string{"", "nonsense", "99999999999.1.1", "2024.99999999999.1"} {
		if _, ok := parseYtDlpVersion(tag); ok {
			t.Fatalf("parseYtDlpVersion(%q) accepted a value it cannot represent", tag)
		}
	}
}

// A zip entry may be absolute or contain "..", and this archive is downloaded from
// a remote release: writing outside the download directory is refused.
func TestUnzipExecutableRefusesEntriesOutsideTheDownloadDirectory(t *testing.T) {
	root := t.TempDir()
	downloadDir := filepath.Join(root, "download")

	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		t.Fatalf("create the download directory: %v", err)
	}

	zipPath := filepath.Join(downloadDir, "tool.zip")

	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create the archive: %v", err)
	}

	w := zip.NewWriter(file)

	entry, err := w.Create("../escaped.exe")
	if err != nil {
		t.Fatalf("add the escaping entry: %v", err)
	}
	if _, err := entry.Write([]byte("payload")); err != nil {
		t.Fatalf("write the escaping entry: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close the archive: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close the file: %v", err)
	}

	if _, err := UnzipExecutable(zipPath); err == nil {
		t.Fatal("UnzipExecutable accepted an entry pointing outside the download directory")
	}

	// ../escaped.exe would have landed next to the download directory.
	if _, err := os.Stat(filepath.Join(root, "escaped.exe")); err == nil {
		t.Fatal("the escaping entry was written outside the download directory")
	}
}
