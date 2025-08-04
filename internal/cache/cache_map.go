package cache

import (
	"fmt"
	"log"
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
			return nil, fmt.Errorf("%s not supported", id)
		}
		cm.cache[id] = e
	}
	e.Open()
	log.Println("Open cache entry:", id)

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
	log.Println("Release cache entry:", id)
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
	log.Println("Close cache entry:", id)

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
