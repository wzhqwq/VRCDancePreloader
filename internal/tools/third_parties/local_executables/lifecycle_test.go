package local_executables

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
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
	// Deliberately no destroy() here. A ConfigurableTool is started and stopped
	// once in the lifetime of the process (see the note on tool.go's stopCh), so a
	// test must not stop it: a second run of this test (-count=2) would panic on
	// the already closed channel. The pump goroutine simply lives until the test
	// binary exits.

	// initialize ran with the zero Config, so no binary was probed and there is no
	// version to report on this machine either way.
	if Get("ytdlp").Info().Version != "" {
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

// An archive that offers several executables used to be resolved by zip entry
// order, which is not an intent: the one that came first was installed. It is
// refused instead.
func TestUnzipExecutableRefusesAnAmbiguousArchive(t *testing.T) {
	root := t.TempDir()

	writeZip := func(name string, entries ...string) string {
		t.Helper()

		path := filepath.Join(root, name)

		file, err := os.Create(path)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}

		w := zip.NewWriter(file)
		for _, entry := range entries {
			f, err := w.Create(entry)
			if err != nil {
				t.Fatalf("add %s: %v", entry, err)
			}
			if _, err := f.Write([]byte("payload")); err != nil {
				t.Fatalf("write %s: %v", entry, err)
			}
		}
		if err := w.Close(); err != nil {
			t.Fatalf("close %s: %v", name, err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("close %s: %v", name, err)
		}

		return path
	}

	two := writeZip("two.zip", "yt-dlp.exe", "yt-dlp_min.exe")

	if _, err := UnzipExecutable(two); err == nil {
		t.Fatal("UnzipExecutable accepted an archive with two executables and picked one")
	}

	// A single executable is still extracted, so the refusal is about ambiguity
	// and not about archives in general.
	one := writeZip("one.zip", "readme.txt", "yt-dlp.exe")

	extracted, err := UnzipExecutable(one)
	if err != nil {
		t.Fatalf("UnzipExecutable refused an unambiguous archive: %v", err)
	}
	if !strings.HasSuffix(extracted, "yt-dlp.exe") {
		t.Fatalf("extracted %q, want the single executable in the archive", extracted)
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
