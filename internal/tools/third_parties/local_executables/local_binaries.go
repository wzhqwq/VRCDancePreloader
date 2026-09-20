package local_executables

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader/task"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/api"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewLogger("Executable Manager")

func getLocalBinary(name string) (string, bool) {
	p := filepath.Join(getLocalBinariesPath(), name)
	if _, err := os.Stat(p); err != nil {
		return "", false
	}
	return p, true
}

// appDataRoot is where this tool keeps its files. Outside fyne mode there is no
// AppDataRoot, and everything then lives next to the working directory — the
// binaries and the temporary files used to disagree about that.
func appDataRoot() string {
	if custom_fyne.AppDataRoot == "" {
		// not in fyne mode, use a relative path
		return "."
	}
	return custom_fyne.AppDataRoot
}

func getLocalBinariesPath() string {
	return filepath.Join(appDataRoot(), "binaries")
}

func getLocalBinaryDownloadPath() string {
	return filepath.Join(getLocalBinariesPath(), "download")
}

func (d *DownloadableBinary) downloadFile(release *api.BriefRelease) error {
	downloadPath := filepath.Join(getLocalBinaryDownloadPath(), release.Name)
	file, err := os.OpenFile(downloadPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer file.Close()

	client := requesting.GetClient(requesting.GitHubAssets)
	id := fmt.Sprintf("%s (%s)", release.Name, release.ReleaseName)

	t := task.NewTask(id, task.NewRangedRemoteProvider(release.BrowserDownloadURL, client), task.NewLocalFileProvider(file))

	// The pointer is published and cleared under the write lock: CancelDownload
	// reads it from the GUI goroutine. The download itself works on the local
	// value, so it never reads the field back.
	d.mutex.Lock()
	d.task = t
	d.mutex.Unlock()

	go t.Download()
	defer func() {
		d.mutex.Lock()
		d.task = nil
		d.mutex.Unlock()
	}()

	ch := t.SubscribeChanges()
	defer ch.Close()
	var lastNotify time.Time
	for {
		select {
		case <-d.stopCh:
			// The tool is stopping: a download nothing will collect must not keep
			// Upgrade (and its caller) blocked past the shutdown.
			return errors.New("the tool is shutting down")
		case <-ch.Channel:
			state, err := t.StateAndError()
			if state == task.TaskCompleted {
				return nil
			}
			if err != nil {
				return err
			}
			if t.TotalSize() > 0 && time.Since(lastNotify) > time.Millisecond*500 {
				d.em.NotifySubscribers(BinProgress)
				lastNotify = time.Now()
			}
		}
	}
}

var integrityLevelRegex = regexp.MustCompile(`([^\\]+) Mandatory Level`)

// runIcacls runs icacls and returns its output.
//
// A variable so that a test can observe the integrity check without reading (and,
// for a Low level, rewriting) the ACLs of the machine it runs on.
var runIcacls = func(ctx context.Context, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, "icacls", args...).Output()
}

// checkIntegrityLevel inspects the executable at path and raises it to Medium
// integrity when Windows marked it as Low.
//
// The path is a parameter instead of a field read so that this can run without
// taking the lock: it starts a process, and its only caller already knows the
// path (see ensureIntegrityCheckedLocked).
func (d *DownloadableBinary) checkIntegrityLevel(path string) {
	ctx, cancel := d.generateContext(3 * time.Second)
	defer cancel()

	output, err := runIcacls(ctx, path)
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			logger.InfoLn("stderr:\n" + string(ee.Stderr))
		}

		logger.WarnLn("Failed to determine integrity level of", path)
	}

	matches := integrityLevelRegex.FindStringSubmatch(string(output))
	if len(matches) != 2 {
		// Not a failure, and not worth a word: "Mandatory Label" is a label *name*
		// the system prints only for files whose integrity level was set explicitly,
		// so the ordinary file has no such line at all and simply runs at the
		// default Medium. It is not localized output — only a file that carries a
		// label can be Low, which is the one case left to raise.
		return
	}

	if matches[1] == "Low" {
		logger.InfoLn("The integrity level of", path, "is Low, we should raise that to `Medium` to allow it to access filesystem")
		if _, err := runIcacls(ctx, path, "/setintegritylevel", "M"); err != nil {
			logger.WarnLn("Failed to raise the integrity level of", path, ", the executable may encounter problems while running")
		}
	}
}

func (d *DownloadableBinary) SetPathAndCheck(path string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.path = path

	// Cache the resolved path here, where the write lock is legitimately held:
	// the resolved form is what gets passed to yt-dlp and used as a rename
	// target, and Valid() cannot do this itself because it is also called under
	// the read lock.
	if resolved := d.resolvePath(); resolved != "" {
		d.path = resolved
	}

	// The integrity check is deliberately *not* done here: it runs up to two
	// icacls processes with a 3s timeout each, and this write lock is what the
	// GUI and the video request path wait on. Execute guarantees it instead,
	// before it starts the executable (see ensureIntegrityCheckedLocked). A new
	// path may point at a file that was never inspected, so the old answer is
	// dropped.
	d.integrityChecked.Store(false)
}

// ensureIntegrityCheckedLocked inspects the installed executable unless it has
// already been inspected. Callers must hold the lock.
//
// Windows refuses to start a Low integrity executable, so nothing may run before
// this has happened once for the file that is about to run. It runs icacls while
// the read lock Execute holds for the whole child run is held, which is
// deliberate: writers are kept out for the whole command anyway, so the
// inspection only extends that by its own duration, and in exchange the path is
// read under the same lock that protects the rename.
func (d *DownloadableBinary) ensureIntegrityCheckedLocked() {
	if d.integrityChecked.Load() {
		return
	}

	d.checkIntegrityLevel(d.path)

	// Stored after the check returns, so the flag never claims a file that was not
	// inspected; two concurrent Execute calls may both see false and inspect twice
	// (icacls is idempotent, so the cost is a repeated call, never a wrong level).
	// A writer cannot replace the file in between: that needs the write lock.
	d.integrityChecked.Store(true)
}

var ErrExecutableNotFound = errors.New("executable not found")
var ErrParsingReleaseVersion = errors.New("failed to parse release version")

func (d *DownloadableBinary) Execute(ctx context.Context, arg ...string) (string, error) {
	// The read lock is held for the whole run of the child process, so that
	// nobody replaces or deletes the file while it is executing. Both Valid and
	// the path below therefore read the fields directly; calling the accessors
	// here would take the read lock a second time.
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if !d.Valid() {
		return "", ErrExecutableNotFound
	}

	// Nothing may be started before the file has been inspected once: see
	// ensureIntegrityCheckedLocked.
	d.ensureIntegrityCheckedLocked()

	cmd := exec.CommandContext(ctx, d.path, arg...)
	output, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			logger.InfoLn("stderr:\n" + string(ee.Stderr))
		}

		return "", fmt.Errorf("failed to execute '%s': %v", cmd.String(), err)
	}

	return string(output), nil
}

// RequestRunnable takes the read lock for as long as the caller uses the
// executable, and reports ErrExecutableNotFound when there is nothing usable to
// run.
//
// The error path releases the lock itself, so that acquisition and release have
// the ordinary shape:
//
//	if err := d.RequestRunnable(); err != nil {
//		return err
//	}
//	defer d.ReleaseRunnable()
//
// The previous version deliberately kept the lock on failure and relied on the
// caller's unconditional ReleaseRunnable to drop it. That happened to pair up in
// the single existing call site, but the shape above — the one anybody would
// write — left the mutex read locked forever, so every later Lock()
// (SetPathAndCheck, DownloadAndReplace, Init, ...) blocked silently.
func (d *DownloadableBinary) RequestRunnable() error {
	d.mutex.RLock()
	if !d.Valid() {
		d.mutex.RUnlock()
		return ErrExecutableNotFound
	}
	return nil
}

func (d *DownloadableBinary) ReleaseRunnable() {
	d.mutex.RUnlock()
}

func (d *DownloadableBinary) DownloadAndReplace() error {
	// One snapshot of the release for the whole operation: it is what names the
	// download file and decides whether the archive has to be unpacked, and
	// reading the field again halfway through would race with CheckUpdates.
	release := d.Release()
	if release == nil {
		return errors.New("there is no release to download")
	}

	err := os.MkdirAll(getLocalBinaryDownloadPath(), 0755)
	if err != nil {
		return err
	}

	downloadedExecutable := filepath.Join(getLocalBinaryDownloadPath(), release.Name)
	defer func() {
		if _, err := os.Stat(downloadedExecutable); err == nil {
			if err := os.Remove(downloadedExecutable); err != nil {
				logger.ErrorLn("Failed to remove downloaded executable: ", err)
			}
		}
	}()

	err = d.downloadFile(release)
	if err != nil {
		return err
	}

	if strings.HasSuffix(release.Name, ".zip") {
		downloadedExecutable, err = UnzipExecutable(downloadedExecutable)
		if err != nil {
			return err
		}
	}

	d.setState(BinDownloaded)

	d.mutex.Lock()
	defer d.mutex.Unlock()

	if err := os.Rename(downloadedExecutable, d.path); err != nil {
		// On Windows the usual cause is that the installed binary is still running
		// (Init executes --version right after every upgrade), which makes the rename
		// fail with "Access is denied".
		return fmt.Errorf("failed to install the downloaded binary as %s: %w", d.path, err)
	}

	// The file at this path is no longer the one that was inspected: the next
	// Execute inspects this one before it starts it.
	d.integrityChecked.Store(false)

	return nil
}

// isInside reports whether path is baseDir itself or something below it.
func isInside(baseDir, path string) bool {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return false
	}

	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func UnzipExecutable(zipPath string) (string, error) {
	defer func() {
		err := os.Remove(zipPath)
		if err != nil {
			logger.ErrorLn("Failed to remove zip file", zipPath, err)
		} else {
			logger.InfoLn("Removed zip file", zipPath)
		}
	}()

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var executables []*zip.File
	for _, f := range r.File {
		if strings.HasSuffix(f.Name, ".exe") {
			executables = append(executables, f)
		}
	}

	if len(executables) == 0 {
		return "", errors.New("there is no executable file in the archive")
	}
	if len(executables) > 1 {
		// Taking one of them would be arbitrary — zip entry order decides, not
		// intent — so an archive that offers several is refused rather than
		// installed as whichever came first. The archive comes from a remote
		// release, and a future one may well bundle a variant.
		return "", fmt.Errorf("the archive holds %d executables, which one to install is ambiguous", len(executables))
	}

	f := executables[0]

	baseDir := filepath.Dir(zipPath)

	// The entry name comes from a downloaded archive, so it may be absolute or
	// contain "..". filepath.Join already cleans the result, which makes a plain
	// containment check enough.
	path := filepath.Join(baseDir, f.Name)
	if !isInside(baseDir, path) {
		return "", fmt.Errorf("the archive entry %q points outside the download directory", f.Name)
	}

	dstFile, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return "", err
	}
	defer dstFile.Close()

	rc, err := f.Open()
	if err != nil {
		return "", err
	}

	_, err = io.Copy(dstFile, rc)
	if err != nil {
		return "", err
	}

	return path, nil
}

type BinaryInfo struct {
	Exists  bool
	Version string
	Size    int64
}
