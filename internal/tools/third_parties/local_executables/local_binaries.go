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

	"github.com/samber/lo"
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

func getLocalBinariesPath() string {
	if custom_fyne.AppDataRoot == "" {
		// not in fyne mode, use relative path
		return "./binaries"
	}
	return filepath.Join(custom_fyne.AppDataRoot, "binaries")
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

//func (d *DownloadableBinary) raiseIntegrityLevel(ctx context.Context) error {
//	// icacls path /setintegritylevel medium
//	if d.lowLevel.CompareAndSwap(true, false) {
//		cmd := exec.CommandContext(ctx, "icacls", d.Path, "/setintegritylevel", "M")
//		return cmd.Run()
//	}
//	return nil
//}

//	func (d *DownloadableBinary) resumeIntegrityLevel() error {
//		if d.lowLevel.CompareAndSwap(false, true) {
//			cmd := exec.Command("icacls", d.Path, "/setintegritylevel", "L")
//			return cmd.Run()
//		}
//		return nil
//	}
var integrityLevelRegex = regexp.MustCompile(`([^\\]+) Mandatory Level`)

// checkIntegrityLevel inspects the executable at path and raises it to Medium
// integrity when Windows marked it as Low.
//
// The path is a parameter instead of a field read because the two callers reach
// it under different locks: SetPathAndCheck holds the write lock (and knows the
// resolved path), while Upgrade has just renamed the new file into place and
// holds nothing.
func (d *DownloadableBinary) checkIntegrityLevel(path string) {
	//d.lowLevel.Store(false)

	ctx, cancel := d.generateContext(3 * time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "icacls", path)

	output, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			logger.InfoLn("stderr:\n" + string(ee.Stderr))
		}

		logger.WarnLn("Failed to determine integrity level of", path)
	}

	matches := integrityLevelRegex.FindStringSubmatch(string(output))
	if len(matches) == 2 && matches[1] == "Low" {
		logger.InfoLn("The integrity level of", path, "is Low, we should raise that to `Medium` to allow it to access filesystem")
		cmd = exec.CommandContext(ctx, "icacls", path, "/setintegritylevel", "M")
		if cmd.Run() != nil {
			logger.WarnLn("Failed to raise the integrity level of", path, ", the executable may encounter problems while running")
		}
	}
	//if len(matches) != 2 {
	//	d.lowLevel.Store(false)
	//} else {
	//	d.lowLevel.Store(matches[1] == "Low")
	//}
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
		d.checkIntegrityLevel(resolved)
	}
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

	//err := d.raiseIntegrityLevel(ctx)
	//if err != nil {
	//	return "", err
	//}
	//defer func() {
	//	err := d.resumeIntegrityLevel()
	//	if err != nil {
	//		logger.ErrorLn("Failed to resume integrity level of ", d.Path, ":", err)
	//	}
	//}()

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

//	func (d *DownloadableBinary) RequestRunnableIntegrity(ctx context.Context) error {
//		d.mutex.RLock()
//		if !d.Valid() {
//			return ErrExecutableNotFound
//		}
//
//		return d.raiseIntegrityLevel(ctx)
//	}
//
//	func (d *DownloadableBinary) ReleaseRunnableIntegrity() {
//		d.mutex.RUnlock()
//		if !d.Valid() {
//			return
//		}
//
//		err := d.resumeIntegrityLevel()
//		if err != nil {
//			logger.ErrorLn("Failed to resume integrity level of ", d.Path, ":", err)
//		}
//	}
//
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
	return os.Rename(downloadedExecutable, d.path)
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

	f, found := lo.Find(r.File, func(f *zip.File) bool {
		return strings.HasSuffix(f.Name, ".exe")
	})
	if !found {
		return "", errors.New("there is no executable file in the archive")
	}

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
