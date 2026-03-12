package local_executables

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
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

func downloadFile(release *third_party_api.BriefRelease, ctx context.Context, onProgress func(total, downloaded int64)) error {
	downloadPath := filepath.Join(getLocalBinaryDownloadPath(), release.Name)
	file, err := os.OpenFile(downloadPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer file.Close()

	client := requesting.GetClient(requesting.GitHubAssets)

	id := fmt.Sprintf("%s (%s)", release.Name, release.ReleaseName)
	task := download.DownloadWithoutManager(id, release.BrowserDownloadURL, ctx, client, file)

	ch := task.SubscribeChanges()
	defer ch.Close()
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			if errors.Is(err, context.Canceled) {
				return context.Cause(ctx)
			}
		case <-ch.Channel:
			if task.Done {
				return nil
			}
			if task.Error != nil {
				return task.Error
			}
			if task.TotalSize > 0 {
				onProgress(task.TotalSize, task.DownloadedSize)
			}
		}
	}
}

func DownloadAndReplace(name string, release *third_party_api.BriefRelease, ctx context.Context, onProgress func(total, downloaded int64)) error {
	err := os.MkdirAll(getLocalBinaryDownloadPath(), 0755)
	if err != nil {
		return err
	}

	err = downloadFile(release, ctx, onProgress)
	if err != nil {
		return err
	}

	return os.Rename(filepath.Join(getLocalBinaryDownloadPath(), release.Name), filepath.Join(getLocalBinariesPath(), name))
}
