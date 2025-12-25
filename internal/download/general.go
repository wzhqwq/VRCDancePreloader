package download

import (
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/stability"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewUniqueLogger("Downloader")

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
	ds := dm.CreateOrGetState(id)
	if ds == nil {
		return nil
	}
	go func() {
		ds.Download(false)
		dm.UpdatePriorities()
	}()

	return ds
}

func RestartTaskIfTooSlow(id string, timeRemaining time.Duration) {
	findManager(id).RestartTaskIfTooSlow(id, timeRemaining)
}

func Retry(task *Task) {
	go func() {
		task.Download(true)
		task.manager.UpdatePriorities()
	}()
}
