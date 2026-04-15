package download

import (
	"context"
	"io"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/stability"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewLogger("Downloader")

func CancelDownload(ids ...string) {
	if len(ids) == 0 {
		return
	}
	managers["pypy"].CancelDownload(ids...)
	managers["default"].CancelDownload(ids...)
}

func Prioritize(ids ...string) {
	if len(ids) == 0 {
		return
	}
	managers["pypy"].Prioritize(ids...)
	managers["default"].Prioritize(ids...)
}

func QueueTransaction() func() {
	cancelPyPy := managers["pypy"].QueueTransaction()
	cancelDefault := managers["default"].QueueTransaction()
	return func() {
		cancelPyPy()
		cancelDefault()
	}
}

func StopAllAndWait() {
	cancel := stability.PanicIfTimeout("download_StopAllAndWait")
	defer cancel()
	managers["pypy"].Destroy()
	managers["default"].Destroy()
}

func SetMaxParallel(n int) {
	managers["pypy"].SetMaxParallel(n)
	managers["default"].SetMaxParallel(n)
}

func Download(id string) *Task {
	dm := findManager(id)
	task := dm.CreateOrGetPausedTask(id)
	if task == nil {
		return nil
	}
	go func() {
		task.Download(false)
		// re-calculate priorities after download
		dm.UpdatePriorities()
	}()

	return task
}

func DownloadWithoutManager(id string, url string, ctx context.Context, client *requesting.ClientProvider, writer io.Writer) *Task {
	task := newTaskWithoutManager(id)
	go func() {
		task.Error = task.DownloadWithoutManager(url, ctx, client, writer)
		task.notifyStateChange()
	}()
	return task
}

func UpdateRequestEta(id string, eta time.Time, duration time.Duration) {
	findManager(id).UpdateRequestEta(id, eta, duration)
}

func Retry(task *Task) {
	go func() {
		task.Download(true)
		task.manager.UpdatePriorities()
	}()
}
