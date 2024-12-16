package cache

import (
	"os"
	"sort"
	"strings"
	"sync"
)

var fileMap = make(map[string]*os.File)
var fileMapMutex = &sync.Mutex{}

var cachePath string
var maxSize int

func InitCache(path string, max int, maxParallel int) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0777)
	}

	cachePath = path
	maxSize = max
	dm = newDownloadManager(maxParallel)

	err := loadSongs()
	if err != nil {
		return err
	}
	return nil
}

func StopCache() {
}

func OpenCache(id string) *os.File {
	fileMapMutex.Lock()
	defer fileMapMutex.Unlock()

	if file, ok := fileMap[id]; ok {
		return file
	}

	if file := openCacheFS(id); file != nil {
		fileMap[id] = file
		return file
	}

	return nil
}
func closeCache(id string) {
	fileMapMutex.Lock()
	defer fileMapMutex.Unlock()

	if file, ok := fileMap[id]; ok {
		file.Close()
		delete(fileMap, id)
	}
}

func DetachCache(id string) {
	closeCache(id)
	CleanUpCache()
}

func RemoveCache(id string) {
	fileMapMutex.Lock()
	defer fileMapMutex.Unlock()

	if file, ok := fileMap[id]; ok {
		file.Close()
		delete(fileMap, id)
		os.Remove(getCacheFileName(id))
	}
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
		if fileMap[id] != nil {
			continue
		}

		os.Remove(cachePath + "/" + file.Name())
		totalSize -= int(file.Size())
	}
}
