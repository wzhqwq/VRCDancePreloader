package download

var dm *downloadManager = nil

func InitDownloadManager(maxParallel int) {
	dm = newDownloadManager(maxParallel)
}

func CancelDownload(id string) {
	dm.CancelDownload(id)
}

func Prioritize(id string) {
	dm.Prioritize(id)
}

func StopAllAndWait() {
	dm.CancelAllAndWait()
}
