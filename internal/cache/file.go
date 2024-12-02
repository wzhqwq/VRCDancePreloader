package cache

import (
	"fmt"
	"os"
)

func getCacheFileName(id string) string {
	return fmt.Sprintf("%s/%s.mp4", cachePath, id)
}

func openCacheFS(id string) *os.File {
	file, err := os.OpenFile(getCacheFileName(id), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil
	}
	return file
}
func getCacheSize(id string) int64 {
	stat, err := os.Stat(getCacheFileName(id))
	if err != nil {
		return 0
	}

	return stat.Size()
}
func openTempFile(id string) *os.File {
	file, err := os.OpenFile(fmt.Sprintf("%s/%s.mp4.dl", cachePath, id), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil
	}
	return file
}
