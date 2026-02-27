package video_cache

import (
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache/entry"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewLogger("Video Cache")

type CacheMap struct {
	sync.Mutex
	cache map[string]entry.Entry

	cleanUpCh chan struct{}
	stopCh    chan struct{}
}

func NewCacheMap() *CacheMap {
	return &CacheMap{
		cache: make(map[string]entry.Entry),

		cleanUpCh: make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
	}
}

func (cm *CacheMap) EventLoop() {
	for {
		select {
		case <-cm.cleanUpCh:
			cm.cleanUp()
		case <-cm.stopCh:
			return
		}
	}
}

func (cm *CacheMap) Stop() {
	close(cm.stopCh)
}

func (cm *CacheMap) findOrCreate(id string) (entry.Entry, error) {
	cm.Lock()
	defer cm.Unlock()

	e, ok := cm.cache[id]
	if !ok {
		e = NewEntry(id)
		if e == nil {
			return nil, entry.ErrNotSupported
		}
		cm.cache[id] = e
	}

	return e, nil
}

func (cm *CacheMap) removeIfInactive(id string) (entry.Entry, bool) {
	cm.Lock()
	defer cm.Unlock()

	e, ok := cm.cache[id]
	if !ok || e.Active() {
		return nil, false
	}
	delete(cm.cache, id)

	return e, true
}

func (cm *CacheMap) Open(id string) (entry.Entry, error) {
	e, err := cm.findOrCreate(id)
	if err != nil {
		return nil, err
	}
	e.Open()
	return e, nil
}
func (cm *CacheMap) Release(id string) {
	cm.Lock()
	defer cm.Unlock()

	e, ok := cm.cache[id]
	if !ok {
		return
	}
	e.Release()
}
func (cm *CacheMap) CloseIfInactive(id string) bool {
	e, ok := cm.removeIfInactive(id)
	if !ok {
		return false
	}

	e.Close()
	cm.CleanUp()

	return true
}
func (cm *CacheMap) IsActive(id string) bool {
	cm.Lock()
	defer cm.Unlock()

	return cm.isActive(id)
}

func (cm *CacheMap) isActive(id string) bool {
	e, ok := cm.cache[id]
	if !ok {
		return false
	}

	return e.Active()
}

func (cm *CacheMap) CleanUp() {
	select {
	case cm.cleanUpCh <- struct{}{}:
	default:
	}
}
