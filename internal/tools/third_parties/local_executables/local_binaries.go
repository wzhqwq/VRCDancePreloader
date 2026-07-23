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

	d.Task = task.NewTask(id, task.NewRangedRemoteProvider(release.BrowserDownloadURL, client), task.NewLocalFileProvider(file))
	go d.Task.Download()
	defer func() {
		d.Task = nil
	}()

	ch := d.Task.SubscribeChanges()
	defer ch.Close()
	var lastNotify time.Time
	for {
		select {
		case <-ch.Channel:
			if d.Task.State == task.TaskCompleted {
				return nil
			}
			if d.Task.Error != nil {
				return d.Task.Error
			}
			if d.Task.TotalSize > 0 && time.Since(lastNotify) > time.Millisecond*500 {
				d.em.NotifySubscribers(BinProgress)
				lastNotify = time.Now()
			}
		}
	}
}

func (d *DownloadableBinary) raiseIntegrityLevel(ctx context.Context) error {
	// icacls path /setintegritylevel medium
	if d.lowLevel.CompareAndSwap(true, false) {
		cmd := exec.CommandContext(ctx, "icacls", d.Path, "/setintegritylevel", "M")
		return cmd.Run()
	}
	return nil
}

func (d *DownloadableBinary) resumeIntegrityLevel() error {
	if d.lowLevel.CompareAndSwap(false, true) {
		cmd := exec.Command("icacls", d.Path, "/setintegritylevel", "L")
		return cmd.Run()
	}
	return nil
}

var integrityLevelRegex = regexp.MustCompile(`([^\\]+) Mandatory Level`)

func (d *DownloadableBinary) checkIntegrityLevel() {
	d.lowLevel.Store(false)

	ctx, cancel := d.generateContext(3 * time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "icacls", d.Path)

	output, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			logger.InfoLn("stderr:\n" + string(ee.Stderr))
		}

		logger.WarnLn("Failed to determine integrity level of", d.Path)
	}

	matches := integrityLevelRegex.FindStringSubmatch(string(output))
	if len(matches) != 2 {
		d.lowLevel.Store(false)
	} else {
		d.lowLevel.Store(matches[1] == "Low")
	}
}

func (d *DownloadableBinary) SetPathAndCheck(path string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.Path = path
	if d.Valid() {
		d.checkIntegrityLevel()
	}
}

var ErrExecutableNotFound = errors.New("executable not found")
var ErrParsingReleaseVersion = errors.New("failed to parse release version")

func (d *DownloadableBinary) Execute(ctx context.Context, arg ...string) (string, error) {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	if !d.Valid() {
		return "", ErrExecutableNotFound
	}

	err := d.raiseIntegrityLevel(ctx)
	if err != nil {
		return "", err
	}
	defer func() {
		err := d.resumeIntegrityLevel()
		if err != nil {
			logger.ErrorLn("Failed to resume integrity level of ", d.Path, ":", err)
		}
	}()

	cmd := exec.CommandContext(ctx, d.Path, arg...)
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

func (d *DownloadableBinary) RequestRunnableIntegrity(ctx context.Context) error {
	d.mutex.RLock()
	if !d.Valid() {
		return ErrExecutableNotFound
	}

	return d.raiseIntegrityLevel(ctx)
}

func (d *DownloadableBinary) ReleaseRunnableIntegrity() {
	d.mutex.RUnlock()
	if !d.Valid() {
		return
	}

	err := d.resumeIntegrityLevel()
	if err != nil {
		logger.ErrorLn("Failed to resume integrity level of ", d.Path, ":", err)
	}
}

func (d *DownloadableBinary) DownloadAndReplace() error {
	err := os.MkdirAll(getLocalBinaryDownloadPath(), 0755)
	if err != nil {
		return err
	}

	downloadedExecutable := filepath.Join(getLocalBinaryDownloadPath(), d.Release.Name)
	defer func() {
		if _, err := os.Stat(downloadedExecutable); err == nil {
			if err := os.Remove(downloadedExecutable); err != nil {
				logger.ErrorLn("Failed to remove downloaded executable: ", err)
			}
		}
	}()

	err = d.downloadFile(d.Release)
	if err != nil {
		return err
	}

	if strings.HasSuffix(d.Release.Name, ".zip") {
		downloadedExecutable, err = UnzipExecutable(downloadedExecutable)
		if err != nil {
			return err
		}
	}

	d.setState(BinDownloaded)

	d.mutex.Lock()
	defer d.mutex.Unlock()
	return os.Rename(downloadedExecutable, d.Path)
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

	path := filepath.Join(filepath.Dir(zipPath), f.Name)

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
