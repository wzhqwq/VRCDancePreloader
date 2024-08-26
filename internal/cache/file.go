package cache

import (
	"fmt"
	"os"
)

func getCacheFileName(id int) string {
	return fmt.Sprintf("%s/%d.mp4", cachePath, id)
}

func openCacheFS(id int) *os.File {
	file, err := os.OpenFile(getCacheFileName(id), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil
	}
	return file
}
func getCacheSize(id int) int64 {
	stat, err := os.Stat(getCacheFileName(id))
	if err != nil {
		return 0
	}

	return stat.Size()
}
func openTempFile(id int) *os.File {
	file, err := os.OpenFile(fmt.Sprintf("%s/temp_%d.mp4", cachePath, id), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil
	}
	return file
}
