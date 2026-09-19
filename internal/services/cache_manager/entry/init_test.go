package entry

import (
	"errors"
	"testing"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/legacy_file"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// errStop ends the Init loop with a plain error, so that a test can observe what
// happened without letting Init reach updateMeta and the database.
var errStop = errors.New("test: stop here")

// stubFile satisfies types.DeferredReadableFile by embedding the interface and
// implementing only Init.
//
// Anything else the code under test touches would panic on the nil embedded
// value, which is the signal we want: these tests must not need a cache
// directory, and Init must not reach past Init.
type stubFile struct {
	types.DeferredReadableFile

	initErr   error
	initCalls int
	closeCall int
}

func (f *stubFile) Init(int64, time.Time) error {
	f.initCalls++
	return f.initErr
}

func (f *stubFile) Close() error {
	f.closeCall++
	return nil
}

func newTestEntry() *BaseCDNEntry {
	return &BaseCDNEntry{logger: utils.NewLogger("Test Entry")}
}

// P0-1 — a legacy entry whose upgrade leaves no working file must fail, not
// panic.
//
// upgradeFn *replaces* e.workingFile. The real upgradeFile closed the legacy
// file, deleted it, opened a replacement and then discarded the return value, so
// the loop carried on with a nil interface and the next Init call was a nil
// dereference. Setting workingFile to nil here reproduces exactly that.
func TestLegacyUpgradeWithoutAReplacementFileFails(t *testing.T) {
	legacy := &stubFile{initErr: legacy_file.ErrLegacyDeprecated}

	e := newTestEntry()
	e.workingFile = legacy

	upgrades := 0
	e.upgradeFn = func() {
		upgrades++
		e.workingFile = nil // the discarded replacement
	}

	if err := e.Init(1024, time.Time{}, ""); err == nil {
		t.Fatal("Init reported success for an entry with no working file")
	}

	if upgrades != 1 {
		t.Fatalf("upgradeFn ran %d times, want exactly 1", upgrades)
	}
	if legacy.initCalls != 1 {
		t.Fatalf("the legacy file was initialized %d times, want 1", legacy.initCalls)
	}
}

// An entry that never had a working file must be reported rather than
// dereferenced.
func TestInitRejectsAnEntryWithoutAWorkingFile(t *testing.T) {
	e := newTestEntry()

	if err := e.Init(1024, time.Time{}, ""); err == nil {
		t.Fatal("Init reported success for an entry without a working file")
	}
}

// A successful upgrade has to carry on with the replacement.
func TestSuccessfulUpgradeContinuesWithTheReplacement(t *testing.T) {
	legacy := &stubFile{initErr: legacy_file.ErrLegacyDeprecated}
	replacement := &stubFile{initErr: errStop}

	e := newTestEntry()
	e.workingFile = legacy
	e.upgradeFn = func() { e.workingFile = replacement }

	if err := e.Init(1024, time.Time{}, ""); !errors.Is(err, errStop) {
		t.Fatalf("Init = %v, want the replacement's error %v", err, errStop)
	}

	if legacy.initCalls != 1 {
		t.Fatalf("the legacy file was initialized %d times, want 1", legacy.initCalls)
	}
	if replacement.initCalls != 1 {
		t.Fatalf("the replacement was initialized %d times, want 1", replacement.initCalls)
	}
}

// An upgrade that cannot fix the state must not be retried forever: the old loop
// called upgradeFn again on every iteration, so a failing upgradeFile meant an
// unbounded close/delete/reopen cycle.
//
// The retry is run under a timeout so that a regression fails the test instead of
// hanging the whole suite.
func TestUpgradeIsAttemptedOnlyOnce(t *testing.T) {
	legacy := &stubFile{initErr: legacy_file.ErrLegacyDeprecated}

	e := newTestEntry()
	e.workingFile = legacy

	upgrades := 0
	e.upgradeFn = func() { upgrades++ } // deliberately does not fix anything

	done := make(chan error, 1)
	go func() { done <- e.Init(1024, time.Time{}, "") }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Init reported success while the file is still legacy")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Init did not return: the upgrade was retried without a bound")
	}

	if upgrades != 1 {
		t.Fatalf("upgradeFn ran %d times, want exactly 1", upgrades)
	}
}

// When the upgrade cannot produce a usable file, the unusable one has to be
// removed. Leaving it in place would make every later attempt read the same
// legacy file and fail identically, so the entry would never become cacheable
// again — the file, not just the entry, is the poison.
func TestFailedUpgradeDiscardsTheUnusableFiles(t *testing.T) {
	legacy := &stubFile{initErr: legacy_file.ErrLegacyDeprecated}

	e := newTestEntry()
	e.workingFile = legacy
	e.upgradeFn = func() {} // cannot fix anything

	discards := 0
	e.discardFn = func() { discards++ }

	if err := e.Init(1024, time.Time{}, ""); err == nil {
		t.Fatal("Init reported success while the file is still legacy")
	}

	if discards != 1 {
		t.Fatalf("discardFn ran %d times, want exactly 1", discards)
	}
	if e.workingFile != nil {
		t.Fatal("the discarded entry still holds a working file")
	}
	// The close has to happen before the delete: on Windows a still open handle
	// makes it fail, and a half removed entry is the worst outcome.
	if legacy.closeCall != 1 {
		t.Fatalf("the unusable file was closed %d times, want exactly 1", legacy.closeCall)
	}
}

// The other failure shape: the upgrade replaced the file with something that
// could not be opened. There is nothing to close, but the leftovers still have to
// go.
func TestUpgradeWithoutAReplacementFileAlsoDiscards(t *testing.T) {
	legacy := &stubFile{initErr: legacy_file.ErrLegacyDeprecated}

	e := newTestEntry()
	e.workingFile = legacy
	e.upgradeFn = func() { e.workingFile = nil }

	discards := 0
	e.discardFn = func() { discards++ }

	if err := e.Init(1024, time.Time{}, ""); err == nil {
		t.Fatal("Init reported success for an entry with no working file")
	}

	if discards != 1 {
		t.Fatalf("discardFn ran %d times, want exactly 1", discards)
	}
}

// A successful upgrade keeps its replacement: nothing may be deleted.
func TestSuccessfulUpgradeKeepsItsFiles(t *testing.T) {
	legacy := &stubFile{initErr: legacy_file.ErrLegacyDeprecated}
	replacement := &stubFile{initErr: errStop}

	e := newTestEntry()
	e.workingFile = legacy
	e.upgradeFn = func() { e.workingFile = replacement }
	e.discardFn = func() { t.Error("discardFn ran for an upgrade that succeeded") }

	if err := e.Init(1024, time.Time{}, ""); !errors.Is(err, errStop) {
		t.Fatalf("Init = %v, want the replacement's error %v", err, errStop)
	}
}
