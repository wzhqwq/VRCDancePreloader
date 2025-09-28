package cache

import (
	"sync"
)

type CacheMap struct {
	sync.Mutex
	cache map[string]Entry
}

func NewCacheMap() *CacheMap {
	return &CacheMap{
		cache: make(map[string]Entry),
	}
}

func (cm *CacheMap) Open(id string) (Entry, error) {
	cm.Lock()
	defer cm.Unlock()

	e, ok := cm.cache[id]
	if !ok {
		e = NewEntry(id)
		if e == nil {
			return nil, ErrNotSupported
		}
		cm.cache[id] = e
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
	cm.Lock()
	defer cm.Unlock()

	e, ok := cm.cache[id]
	if !ok || e.Active() {
		return false
	}

	e.Close()
	delete(cm.cache, id)
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
