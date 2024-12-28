package cache

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var cachePath string
var maxSize int
var cacheMap = NewCacheMap()

func SetupCache(path string, max int) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0777)
	}

	cachePath = path
	maxSize = max
}

func InitSongList() error {
	err := loadSongs()
	if err != nil {
		return err
	}
	return nil
}

func CleanUpCache() {
	// remove files from cache until total size is less than maxSize
	entries, err := os.ReadDir(cachePath)
	if err != nil {
		return
	}

	files := make([]os.FileInfo, len(entries))
	totalSize := 0
	for i, entry := range entries {
		files[i], _ = entry.Info()
		totalSize += int(files[i].Size())
	}

	// sort entries by last modified time
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().Before(files[j].ModTime())
	})

	// remove files until total size is less than maxSize
	for _, file := range files {
		if totalSize < maxSize {
			break
		}

		id := strings.Split(file.Name(), ".")[0]
		if cacheMap.IsActive(id) {
			continue
		}

		os.Remove(filepath.Join(cachePath, file.Name()))
		totalSize -= int(file.Size())
	}
}

func OpenCacheEntry(id string) (Entry, error) {
	return cacheMap.Open(id)
}

func CloseCacheEntry(id string) {
	cacheMap.Close(id)
}
