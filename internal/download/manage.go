package download

import "github.com/wzhqwq/VRCDancePreloader/internal/utils"

var dm *downloadManager = nil
var logger = utils.NewUniqueLogger()

func InitDownloadManager(maxParallel int) {
	dm = newDownloadManager(maxParallel)
}

func CancelDownload(ids ...string) {
	dm.CancelDownload(ids...)
}

func Prioritize(ids ...string) {
	dm.Prioritize(ids...)
}

func StopAllAndWait() {
	dm.CancelAllAndWait()
}

func SetMaxParallel(n int) {
	dm.SetMaxParallel(n)
}
