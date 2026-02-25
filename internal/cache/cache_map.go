package cache

import (
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache/entry"
)

type CacheMap struct {
	sync.Mutex
	cache map[string]entry.Entry
}

func NewCacheMap() *CacheMap {
	return &CacheMap{
		cache: make(map[string]entry.Entry),
	}
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
	CleanUpCache()

	return true
}
func (cm *CacheMap) IsActive(id string) bool {
	cm.Lock()
	defer cm.Unlock()

	e, ok := cm.cache[id]
	if !ok {
		return false
	}

	return e.Active()
}
