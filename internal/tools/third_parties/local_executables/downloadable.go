package local_executables

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader/task"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/api"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type DownloadableChange string

const (
	BinProgress DownloadableChange = "bin_progress"
	BinVersion  DownloadableChange = "bin_version"
	BinState    DownloadableChange = "bin_state"
)

type DownloadableState string

const (
	BinCheckingLocal   DownloadableState = "bin_checking_local"
	BinInitial         DownloadableState = "bin_initial"
	BinCheckingUpdates DownloadableState = "bin_checking_updates"
	BinUpdateAvailable DownloadableState = "bin_update_available"
	BinIllegal         DownloadableState = "bin_illegal"
	BinDownloading     DownloadableState = "bin_downloading"
	BinDownloaded      DownloadableState = "bin_downloaded"
)

// DownloadableBinary is one external executable (yt-dlp or deno) as the rest of
// the program sees it: where it lives, which version is installed, which release
// is available, and which download is in flight.
//
// It must not be copied after first use: it contains a mutex.
type DownloadableBinary struct {
	// Name is fixed by NewDownloadableBinary, so it is read without the lock:
	// the GUI renders it on every refresh.
	Name string

	// mutex guards every field below it. Two different things live under it:
	//
	//   - path, together with the "may I replace / run the file right now"
	//     protocol: SetPathAndCheck, Remove and the rename at the end of
	//     DownloadAndReplace take the write lock, Execute and RequestRunnable
	//     hold the read lock;
	//   - the observable state machine (state, err, info, release, task), which
	//     the download goroutine writes while the GUI, the availability pump in
	//     tool.go and the accessors below read it.
	//
	// Two rules follow, and both matter:
	//
	//  1. never call an accessor while holding mutex. sync.RWMutex is not
	//     re-entrant, and a second RLock blocks as soon as a writer is queued,
	//     so an accessor calling another accessor can deadlock. Composite
	//     questions use the *Locked helpers instead.
	//  2. publish events with em.NotifySubscribers only after releasing mutex:
	//     subscribers run in the calling goroutine, so notifying under the lock
	//     deadlocks any subscriber that reads the binary back.
	mutex sync.RWMutex

	path    string
	state   DownloadableState
	err     error
	info    BinaryInfo
	release *api.BriefRelease
	task    *task.Task

	em *utils.EventManager[DownloadableChange]

	// probing marks a local probe (Init) as being in flight. The state machine
	// cannot express it: Init runs in BinCheckingLocal, which is also the state a
	// fresh binary starts in, so two concurrent probes both see an idle machine and
	// both run --version.
	probing bool

	// integrityChecked records that the file installed at d.path has had its
	// integrity level inspected. Everything that starts the executable depends on
	// it, so it is invalidated by everything that can change that file:
	// SetPathAndCheck, DownloadAndReplace (which renames a freshly downloaded
	// binary over it) and Remove. It is atomic because the exec path reads and
	// sets it while holding only the read lock.
	integrityChecked atomic.Bool

	stopCh chan struct{}
}

func NewDownloadableBinary(name string) *DownloadableBinary {
	return &DownloadableBinary{
		Name: name,

		state: BinCheckingLocal,

		em: utils.NewEventManager[DownloadableChange](),

		stopCh: make(chan struct{}),
	}
}

func (d *DownloadableBinary) generateContext(dur time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	go func() {
		select {
		case <-d.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

// Accessors.
//
// These are the only way to read a binary from another goroutine. Each one takes
// the lock exactly once, and answers a complete question inside that single
// critical section; a question made of several fields is answered by the
// *Locked helpers below rather than by calling one accessor from another.

// Path is the path the executable is read from and replaced at.
func (d *DownloadableBinary) Path() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.path
}

// State is where the state machine is right now.
func (d *DownloadableBinary) State() DownloadableState {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.state
}

// Err is the error of the last failed operation, or nil.
func (d *DownloadableBinary) Err() error {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.err
}

// Info is what the last local probe found. BinaryInfo is a value, so the caller
// gets a copy instead of a view that is rewritten underneath it.
func (d *DownloadableBinary) Info() BinaryInfo {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.info
}

// Release is the release CheckUpdates found, or nil when there is none (already
// up to date, or the check failed: Err tells the two apart).
//
// The pointed-to value is only written by CheckUpdates when it publishes a new
// one, so a reader may keep and use the pointer.
func (d *DownloadableBinary) Release() *api.BriefRelease {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.release
}

// Task is the download that is currently in flight, or nil.
//
// The pointer may be cleared right after it is returned — that only means the
// download finished — and the task itself is safe to query from any goroutine.
func (d *DownloadableBinary) Task() *task.Task {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.task
}

// resolvePath returns the path of the executable when there is a runnable one,
// and "" otherwise.
//
// It expects the caller to hold mutex, and never writes to the receiver: it is
// reached under the read lock (Execute, RequestRunnable) as well as under the
// write lock (SetPathAndCheck), and caching the resolved path here was a write
// under a read lock: two concurrent Execute calls, or an Execute racing
// SetPathAndCheck, wrote the same field.
func (d *DownloadableBinary) resolvePath() string {
	if d.path == "" {
		return ""
	}

	p, err := exec.LookPath(d.path)
	if err != nil {
		return ""
	}

	return p
}

// Valid reports whether there is a runnable executable.
//
// It expects the caller to hold mutex (read or write) — see resolvePath. Callers
// that hold the write lock cache the resolved path; see SetPathAndCheck.
func (d *DownloadableBinary) Valid() bool {
	return d.resolvePath() != ""
}

// trySetState moves the state machine from one of the states in `from` to `to`
// inside a single critical section, and reports whether this call is the one
// that did it.
//
// Reading the state and then setting it in two steps is what made the entry
// points below re-entrant: two GUI events could both read "update available"
// before either of them wrote "downloading", and both then ran
// DownloadAndReplace on the same file, each renaming its own copy into place.
//
// Like every state change it publishes BinState after releasing the lock.
func (d *DownloadableBinary) trySetState(from []DownloadableState, to DownloadableState) bool {
	d.mutex.Lock()
	moved := slices.Contains(from, d.state)
	if moved {
		d.state = to
	}
	d.mutex.Unlock()

	if moved {
		d.em.NotifySubscribers(BinState)
	}

	return moved
}

func (d *DownloadableBinary) setState(state DownloadableState) {
	d.mutex.Lock()
	d.state = state
	d.mutex.Unlock()

	d.em.NotifySubscribers(BinState)
}

// setErr records the error without publishing anything: every caller pairs it
// with a state change (or, in Remove, with an explicit notification), which is
// what tells the GUI to read the error back.
func (d *DownloadableBinary) setErr(err error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.err = err
}

func (d *DownloadableBinary) setInfo(info BinaryInfo) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.info = info
}

func (d *DownloadableBinary) setRelease(release *api.BriefRelease) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.release = release
}

// idleStates are the states in which no pass owns the state machine: nothing is
// being probed, checked, downloaded, or unpacked-awaiting-its-rename.
//
// Every entry point that has to claim the machine starts from this list, inside
// the same critical section as the transition, so that "is anybody else
// running?" and "I am" cannot be split apart by another goroutine:
//
//   - Init moves into BinCheckingLocal (the state that means "the local
//     situation is not known yet"), so it must not run during a version check —
//     that check ends by writing the release this probe would be compared
//     against — nor during a download, nor in the window between unpacking a new
//     executable and renaming it into place, where probing would report the
//     *old* binary and overwrite the state of a successful download;
//   - CheckUpdates moves into BinCheckingUpdates, and Upgrade moves into
//     BinDownloading, for the same reason: neither may start while another pass
//     owns the machine.
//
// BinCheckingLocal is in the list even though Init is the one moving into it:
// the constructor starts in that state, so refusing it would refuse the startup
// probe. That overlap also means the state machine alone cannot tell a probe that
// is already running from one that has not started yet, so Init carries its own
// `probing` flag on top of this list.
//
// The GUI only ever offers these entry points from the idle states (it hides the
// upgrade button during a download, and disables it during a check), so the
// list is not narrower than what the program asks for.
var idleStates = []DownloadableState{
	BinCheckingLocal,
	BinInitial,
	BinUpdateAvailable,
	BinIllegal,
}

func (d *DownloadableBinary) Init() {
	// One probe at a time: see the note on probing. The flag is released on every
	// exit path, the early returns below included.
	d.mutex.Lock()
	if d.probing {
		d.mutex.Unlock()
		return
	}
	d.probing = true
	d.mutex.Unlock()

	defer func() {
		d.mutex.Lock()
		d.probing = false
		d.mutex.Unlock()
	}()

	// Claim the machine for this probe: see idleStates.
	if !d.trySetState(idleStates, BinCheckingLocal) {
		return
	}

	ctx, cancel := d.generateContext(3 * time.Second)
	defer cancel()

	var info BinaryInfo
	switch d.Name {
	case "ytdlp":
		info = GetLocalYtDlpInfo(ctx)
	case "deno":
		info = GetLocalDenoInfo(ctx)
	default:
		d.setErr(fmt.Errorf("unknown downloadable %s", d.Name))
		d.setState(BinIllegal)
		return
	}

	logger.InfoLn("Local version of", d.Name, "is", info.Version)

	d.setInfo(info)

	d.setState(BinInitial)
	d.em.NotifySubscribers(BinVersion)
}

func (d *DownloadableBinary) CheckUpdates() {
	// Claim the machine for this check: see idleStates.
	if !d.trySetState(idleStates, BinCheckingUpdates) {
		return
	}

	d.setErr(nil)

	ctx, cancel := d.generateContext(10 * time.Second)
	defer cancel()

	var release *api.BriefRelease
	var err error
	switch d.Name {
	case "ytdlp":
		release, err = GetLatestYtDlp(ctx, YtDlpStable)
	case "deno":
		release, err = GetLatestDeno(ctx)
	default:
		d.setErr(fmt.Errorf("unknown downloadable %s", d.Name))
		d.setState(BinIllegal)
		return
	}

	d.setRelease(release)

	if err != nil {
		d.setErr(err)
	}
	if release == nil {
		d.setState(BinInitial)
		return
	}

	logger.InfoLn("New version of", d.Name, "is available:", release.Version)

	d.setState(BinUpdateAvailable)
}

func (d *DownloadableBinary) Upgrade() {
	if d.Release() == nil {
		return
	}

	// The check and the transition are one critical section: with the two steps
	// apart, two events (a double click on the GUI button) could both pass the
	// check and run DownloadAndReplace concurrently on the same download file.
	// See idleStates for the states this refuses.
	if !d.trySetState(idleStates, BinDownloading) {
		return
	}

	d.setErr(nil)

	err := d.DownloadAndReplace()

	if err != nil {
		if !errors.Is(err, context.Canceled) {
			d.setErr(err)
		}
		d.setState(BinUpdateAvailable)
		return
	}

	logger.InfoLn("Downloaded latest version of", d.Name)

	d.setState(BinCheckingLocal)

	// The check is not repeated here: DownloadAndReplace has dropped the previous
	// answer (the file it just renamed into place was never inspected), and the
	// probe below starts that file through Execute, which inspects it first.
	d.Init()
}

// CancelDownload cancels the download that is in flight, if there is one.
//
// downloadFile clears the pointer under the write lock when the download is
// over, so it has to be read under the lock as well: reading it as a plain field
// raced with that cleanup. Task.Cancel only touches the task, never this binary,
// so it is safe to call while holding the read lock.
func (d *DownloadableBinary) CancelDownload() {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if d.task != nil {
		d.task.Cancel()
	}
}

// Stop cancels the running download and closes the cancellation channel.
//
// It belongs to the single stop of the tool, which is why it needs no guard: see
// the note on tool.go's stopCh.
func (d *DownloadableBinary) Stop() {
	d.CancelDownload()
	close(d.stopCh)
}

func (d *DownloadableBinary) Remove() {
	// Refuse while a download owns the state machine.
	//
	// The delete targets the installed binary, while DownloadAndReplace is about to
	// rename the freshly downloaded one into that very path. Whichever order the two
	// take, a delete that lands first is silently undone by that rename — the user
	// would see "deleted, and still there". So the answer is a refusal the caller can
	// show, not a delete that quietly loses. (Cancelling the download and then
	// deleting would be a different feature, not a fix.)
	d.mutex.RLock()
	downloading := d.state == BinDownloading
	d.mutex.RUnlock()

	if downloading {
		d.mutex.Lock()
		d.err = errors.New("the binary is being downloaded, cancel the download before removing it")
		d.mutex.Unlock()

		logger.InfoLn("Refused to remove", d.Path(), "while it is being downloaded")

		// Outside the lock: the subscriber runs in this goroutine.
		d.em.NotifySubscribers(BinState)
		return
	}

	// The write lock is held around the delete so that it stays serialized with
	// SetPathAndCheck and with the rename at the end of DownloadAndReplace.
	d.mutex.Lock()
	path := d.path
	err := os.Remove(path)
	if err != nil {
		d.err = err
	}

	// There is no file to have inspected any more.
	d.integrityChecked.Store(false)
	d.mutex.Unlock()

	if err != nil {
		logger.InfoLn("Failed to remove", path)
		// Outside the lock: the subscriber runs in this goroutine.
		d.em.NotifySubscribers(BinState)
	}

	// Probe again after the delete, and after releasing the lock: Init takes the
	// lock itself, and it refuses to run while a check, a download, or the
	// window between a finished download and its rename owns the machine (see
	// idleStates) — which is exactly the case that used to re-probe with the
	// stale path.
	d.Init()
}

func (d *DownloadableBinary) Subscribe() *utils.EventSubscriber[DownloadableChange] {
	return d.em.SubscribeEvent()
}

// hasUpdatesLocked answers HasUpdates for a caller that already holds the lock.
func (d *DownloadableBinary) hasUpdatesLocked() bool {
	return d.release != nil && d.state != BinCheckingLocal && d.release.Version != d.info.Version
}

func (d *DownloadableBinary) HasUpdates() bool {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	return d.hasUpdatesLocked()
}

// UpdateText describes the available update. It reads release, state and info in
// one critical section, through hasUpdatesLocked: calling HasUpdates from here
// would take the read lock a second time, and a writer queued between the two
// acquisitions would deadlock it.
func (d *DownloadableBinary) UpdateText() string {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if !d.hasUpdatesLocked() {
		return ""
	}

	if d.info.Exists {
		return "→ " + d.release.Version
	}

	return "↓ " + d.release.Version
}
