package cache

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var cachePath string
var maxSize int64
var keepFavorites bool
var cacheMap = NewCacheMap()
var cleanUpChan = make(chan struct{}, 1)

func SetupCache(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0777)
	}

	cachePath = path
}

func SetMaxSize(size int64) {
	maxSize = size
	CleanUpCache()
}
func SetKeepFavorites(b bool) {
	keepFavorites = b
}

func InitSongList() error {
	err := loadSongs()
	if err != nil {
		return err
	}
	return nil
}

func CleanUpCache() {
	// Only one cleanup operation can be running at a time
	select {
	case cleanUpChan <- struct{}{}:
		go func() {
			log.Println("Cleaning up cache ...")
			defer func() {
				log.Println("Cleaned up cache")
				<-cleanUpChan
			}()

			// remove files from cache until total size is less than maxSize
			entries, err := os.ReadDir(cachePath)
			if err != nil {
				return
			}

			files := make([]os.FileInfo, len(entries))
			totalSize := int64(0)
			for i, entry := range entries {
				files[i], _ = entry.Info()
				totalSize += files[i].Size()
			}
			if totalSize <= maxSize {
				return
			}

			// sort entries by last modified time
			sort.Slice(files, func(i, j int) bool {
				return files[i].ModTime().Before(files[j].ModTime())
			})

			favorites := persistence.GetFavorite()

			// remove files until total size is less than maxSize
			for _, file := range files {
				if totalSize < maxSize {
					break
				}

				id := strings.Split(file.Name(), ".")[0]
				if cacheMap.IsActive(id) {
					continue
				}
				if keepFavorites && favorites.IsFavorite(id) {
					continue
				}

				err := os.Remove(filepath.Join(cachePath, file.Name()))
				if err != nil {
					log.Println("[Warning] Failed to remove ", file.Name(), ":", err)
				}
				totalSize -= file.Size()
			}
		}()
	default:
	}
}

func OpenCacheEntry(id string) (Entry, error) {
	return cacheMap.Open(id)
}

func CloseCacheEntry(id string) {
	cacheMap.Close(id)
}
