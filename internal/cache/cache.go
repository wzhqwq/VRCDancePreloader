package cache

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

var fileMap = make(map[int]*os.File)
var cachePath string
var maxSize int

func fillInCache() {
	// fill in cache from cachePath
}

func InitCache(path string, max int) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0777)
	}

	cachePath = path
	maxSize = max
	// fillInCache()
}

func RequestCache(id int) *os.File {
	// check if file exists in cache
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

func DetachCache(id int) {
	if file, ok := fileMap[id]; ok {
		file.Close()
		delete(fileMap, id)
	}
	CleanUpCache()
}

func FlushCache(id int) *os.File {
	if file, ok := fileMap[id]; ok {
		file.Close()
		delete(fileMap, id)
	}
	return RequestCache(id)
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
