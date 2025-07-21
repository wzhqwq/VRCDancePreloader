package download

var dm *downloadManager = nil

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
