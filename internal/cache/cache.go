package cache

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var fileMap = make(map[int]*os.File)
var mutexMap = make(map[int]*sync.Mutex)

var mapMutex = &sync.Mutex{}

var cachePath string
var maxSize int

func InitCache(path string, max int) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0777)
	}

	cachePath = path
	maxSize = max

	go pw.Render()
}

func StopCache() {
}

func OpenCache(id int) *os.File {
	mapMutex.Lock()
	defer mapMutex.Unlock()

	if file, ok := fileMap[id]; ok {
		return file
	}

	file, err := os.OpenFile(fmt.Sprintf("%s/%d.mp4", cachePath, id), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil
	}

	fileMap[id] = file
	return file
}
func closeCache(id int) {
	mapMutex.Lock()
	defer mapMutex.Unlock()

	if file, ok := fileMap[id]; ok {
		file.Close()
		delete(fileMap, id)
		delete(mutexMap, id)
	}
}

func DetachCache(id int) {
	closeCache(id)
	CleanUpCache()
}

func RemoveCache(id int) {
	mapMutex.Lock()
	defer mapMutex.Unlock()

	if file, ok := fileMap[id]; ok {
		file.Close()
		delete(fileMap, id)
		delete(mutexMap, id)
		os.Remove(fmt.Sprintf("%s/%d.mp4", cachePath, id))
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

		idStr := strings.Split(file.Name(), ".")[0]
		id, err := strconv.Atoi(idStr)
		if err == nil && fileMap[id] != nil {
			continue
		}

		os.Remove(cachePath + "/" + file.Name())
		totalSize -= int(file.Size())
	}
}
