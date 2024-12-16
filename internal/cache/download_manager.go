package cache

import (
	"sync"
)

type downloadManager struct {
	sync.Mutex
	//utils.LoggingMutex

	stateMap    map[string]*DownloadState
	queue       []string
	maxParallel int
}

var dm *downloadManager = nil

func newDownloadManager(maxParallel int) *downloadManager {
	return &downloadManager{
		stateMap:    make(map[string]*DownloadState),
		queue:       make([]string, 0),
		maxParallel: maxParallel,
	}
}
func (dm *downloadManager) CreateOrGetState(id string) *DownloadState {
	dm.Lock()
	ds, exists := dm.stateMap[id]
	if !exists {
		ds = &DownloadState{
			StateCh:    make(chan *DownloadState, 10),
			CancelCh:   make(chan bool, 10),
			PriorityCh: make(chan int, 10),

			Pending: true,
		}
		// Check if file is already downloaded
		// NOTE: The cache file is either completely written to disk, or never written at all
		if size := getCacheSize(id); size > 0 {
			ds.TotalSize = size
			ds.DownloadedSize = size
			ds.Done = true
		}
		dm.stateMap[id] = ds
		dm.queue = append(dm.queue, id)
	}

	// NOTE: We need to unlock here early to prevent deadlock
	dm.Unlock()

	// Update priorities if the state is new
	if !exists {
		dm.UpdatePriorities()
	}

	return ds
}
func (dm *downloadManager) CancelDownload(id string) {
	dm.Lock()
	defer dm.unlockAndUpdate()
	ds, ok := dm.stateMap[id]
	if ok {
		close(ds.CancelCh)
		delete(dm.stateMap, id)
	}
}
func (dm *downloadManager) UpdatePriorities() {
	dm.Lock()
	defer dm.Unlock()
	if len(dm.queue) == 0 {
		return
	}

	for i := 0; i < len(dm.queue); i++ {
		ds, ok := dm.stateMap[dm.queue[i]]
		if !ok || ds.Done {
			dm.queue = append(dm.queue[:i], dm.queue[i+1:]...)
			i--
		} else {
			//log.Printf("Priority of %s: %d\n", dm.queue[i], i)
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
func (dm *downloadManager) Prioritize(id string) {
	dm.Lock()
	defer dm.unlockAndUpdate()
	if _, ok := dm.stateMap[id]; ok {
		index := -1
		for i, v := range dm.queue {
			if v == id {
				index = i
				break
			}
		}
		if index != -1 {
			dm.queue = append(dm.queue[:index], dm.queue[index+1:]...)
			dm.queue = append([]string{id}, dm.queue...)
		}
	}
}
