package download

import (
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"log"
	"sync"
)

type downloadManager struct {
	sync.Mutex
	//utils.LoggingMutex

	stateMap    map[string]*State
	queue       []string
	maxParallel int
}

func newDownloadManager(maxParallel int) *downloadManager {
	return &downloadManager{
		stateMap:    make(map[string]*State),
		queue:       make([]string, 0),
		maxParallel: maxParallel,
	}
}
func (dm *downloadManager) CreateOrGetState(id string) *State {
	dm.Lock()
	defer dm.unlockAndUpdate()

	ds, exists := dm.stateMap[id]
	if !exists {
		cacheEntry, err := cache.OpenCacheEntry(id)
		if err != nil {
			log.Println("Skipped: ", err)
			return nil
		}
		ds = &State{
			ID: id,

			cacheEntry: cacheEntry,

			StateCh:    make(chan *State, 10),
			CancelCh:   make(chan bool, 10),
			PriorityCh: make(chan int, 10),

			Pending: true,
		}
		// Check if file is already downloaded
		if cacheEntry.IsComplete() {
			size := cacheEntry.TotalLen()
			ds.TotalSize = size
			ds.DownloadedSize = size
			ds.Done = true
		}
		dm.stateMap[id] = ds
		dm.queue = append(dm.queue, id)
	}

	return ds
}
func (dm *downloadManager) CancelDownload(ids ...string) {
	dm.Lock()
	defer dm.unlockAndUpdate()

	for _, id := range ids {
		if ds, ok := dm.stateMap[id]; ok {
			cache.ReleaseCacheEntry(id)
			close(ds.CancelCh)
			delete(dm.stateMap, id)
		}
	}
}
func (dm *downloadManager) UpdatePriorities() {
	dm.Lock()
	defer dm.Unlock()
	if len(dm.queue) == 0 {
		return
	}

	dm.queue = lo.Filter(dm.queue, func(id string, _ int) bool {
		ds, ok := dm.stateMap[id]
		return ok && !ds.Done
	})
	logger.InfoLn("Download queue:", dm.queue)
	for i, id := range dm.queue {
		ds := dm.stateMap[id]
		if ds != nil {
			ds.PriorityCh <- i
		}
	}
}
func (dm *downloadManager) CanDownload(priority int) bool {
	return priority < dm.maxParallel
}
func (dm *downloadManager) unlockAndUpdate() {
	dm.Unlock()
	dm.UpdatePriorities()
}
func (dm *downloadManager) SetMaxParallel(max int) {
	dm.maxParallel = max
	dm.UpdatePriorities()
}
func (dm *downloadManager) Prioritize(ids ...string) {
	dm.Lock()
	defer dm.unlockAndUpdate()

	if utils.IsPrefixOf(dm.queue, ids) {
		return
	}

	dm.queue = append(
		ids,
		lo.Reject(dm.queue, func(id string, _ int) bool {
			return lo.Contains(ids, id)
		})...,
	)
}

func (dm *downloadManager) CancelAllAndWait() {
	dm.Lock()
	defer dm.Unlock()
	for _, ds := range dm.stateMap {
		close(ds.CancelCh)
		ds.Lock()
		ds.Unlock()
	}
}
