package cache

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

const (
	AllCacheFileRegex      = `^((?:pypy|yt|wanna|bili)_.+)\.(?:mp4|mp4\.(?:dl|vrcdp))$`
	CompleteCacheFileRegex = `^((?:pypy|yt|wanna|bili)_.+)\.mp4$`
	PartialCacheFileRegex  = `^((?:pypy|yt|wanna|bili)_.+)\.mp4\.(?:dl|vrcdp)$`
)

var cachePath string
var maxSize int64
var keepFavorites bool
var fileFormat int

var cacheMap = NewCacheMap()
var cleanUpChan = make(chan struct{}, 1)

var localFileEm = utils.NewEventManager[string]()
var dirWatcher *fsnotify.Watcher

var managerLogger = utils.NewLogger("Cache Manager")

func SetupCache(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0777)
	}

	cachePath = path

	go func() {
		err := watchCacheDir()
		if err != nil {
			managerLogger.ErrorLn("Failed to watch cache directory:", err)
		}
	}()
}

func StopCache() {
	if dirWatcher != nil {
		dirWatcher.Close()
	}
	CleanUpCache()
}

func SetMaxSize(size int64) {
	maxSize = size
	CleanUpCache()
}
func GetMaxSize() int64 {
	return maxSize
}
func SetKeepFavorites(b bool) {
	keepFavorites = b
}
func SetFileFormat(format int) {
	fileFormat = format
}

func CleanUpCache() {
	// Only one cleanup operation can be running at a time
	select {
	case cleanUpChan <- struct{}{}:
		go func() {
			managerLogger.InfoLn("Cleaning up cache ...")
			defer func() {
				managerLogger.InfoLn("Cleaned up cache")
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

			// remove files until total size is less than maxSize
			for _, file := range files {
				if totalSize < maxSize {
					break
				}

				id := strings.Split(file.Name(), ".")[0]
				if cacheMap.IsActive(id) {
					continue
				}
				if keepFavorites && persistence.IsFavorite(id) {
					continue
				}
				if persistence.IsInAllowList(id) {
					continue
				}

				err := os.Remove(filepath.Join(cachePath, file.Name()))
				if err != nil {
					managerLogger.WarnLn("Failed to remove ", file.Name(), ":", err)
				}
				totalSize -= file.Size()
			}
		}()
	default:
	}
}

func GetLocalCacheInfos() []types.CacheFileInfo {
	entries, err := os.ReadDir(cachePath)
	if err != nil {
		return nil
	}

	var infos []types.CacheFileInfo
	for _, entry := range entries {
		matches := regexp.MustCompile(AllCacheFileRegex).FindStringSubmatch(entry.Name())
		if len(matches) == 0 {
			continue
		}
		id := matches[1]
		info, err := entry.Info()
		if err != nil {
			continue
		}
		infos = append(infos, types.CacheFileInfo{
			ID:        id,
			Size:      info.Size(),
			IsActive:  cacheMap.IsActive(id),
			IsPartial: strings.HasSuffix(entry.Name(), ".dl"),
		})
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Size > infos[j].Size
	})

	return infos
}

func GetLocalCacheInfo(id string) types.CacheFileInfo {
	entries, err := os.ReadDir(cachePath)

	if err == nil {
		for _, entry := range entries {
			matches := regexp.MustCompile(AllCacheFileRegex).FindStringSubmatch(entry.Name())
			if len(matches) == 0 || matches[0] != id {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			return types.CacheFileInfo{
				ID:        id,
				Size:      info.Size(),
				IsActive:  cacheMap.IsActive(id),
				IsPartial: strings.HasSuffix(entry.Name(), ".dl"),
			}
		}
	}

	return types.CacheFileInfo{
		ID:        id,
		Size:      0,
		IsActive:  false,
		IsPartial: false,
	}
}

func RemoveLocalCacheById(id string) error {
	if cacheMap.IsActive(id) {
		return nil
	}

	videoTrunkPath := filepath.Join(cachePath, id+".mp4.vrcdp")
	videoPath := filepath.Join(cachePath, id+".mp4")
	videoDlPath := filepath.Join(cachePath, id+".mp4.dl")

	if _, err := os.Stat(videoTrunkPath); err == nil {
		err := os.Remove(videoTrunkPath)
		if err != nil {
			return err
		}
	}
	if _, err := os.Stat(videoPath); err == nil {
		err := os.Remove(videoPath)
		if err != nil {
			return err
		}
	}
	if _, err := os.Stat(videoDlPath); err == nil {
		err := os.Remove(videoDlPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func OpenCacheEntry(id string, logger utils.LoggerImpl) (Entry, error) {
	logger.InfoLn("Open cache entry:", id)
	return cacheMap.Open(id)
}

func ReleaseCacheEntry(id string, logger utils.LoggerImpl) {
	logger.InfoLn("Release cache entry:", id)
	cacheMap.Release(id)
	go func() {
		<-time.After(time.Second)
		if cacheMap.CloseIfInactive(id) {
			managerLogger.InfoLn("Closed cache entry:", id)
			localFileEm.NotifySubscribers("*" + id)
		}
	}()
}

func SubscribeLocalFileEvent() *utils.EventSubscriber[string] {
	return localFileEm.SubscribeEvent()
}

func watchCacheDir() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = watcher.Add(cachePath)
	if err != nil {
		return err
	}

	dirWatcher = watcher
	defer func() {
		dirWatcher = nil
	}()

	managerLogger.InfoLn("Watching directory:", cachePath)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			path := event.Name
			fileName := filepath.Base(path)
			matches := regexp.MustCompile(AllCacheFileRegex).FindStringSubmatch(fileName)
			if len(matches) == 0 {
				continue
			}
			id := matches[1]
			if event.Op.Has(fsnotify.Write) {
				localFileEm.NotifySubscribers("+" + id)
			}
			if event.Op.Has(fsnotify.Remove) {
				localFileEm.NotifySubscribers("-" + id)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}

			return err
		}
	}
}
