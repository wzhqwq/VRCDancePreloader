package cache

import (
	"fmt"
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
		e = OpenEntry(id)
		if e == nil {
			return nil, fmt.Errorf("%s not supported", id)
		}
		cm.cache[id] = e
	}
	return e, nil
}
func (cm *CacheMap) Close(id string) {
	cm.Lock()
	defer cm.Unlock()
	e, ok := cm.cache[id]
	if !ok {
		return
	}
	e.Close()
	delete(cm.cache, id)
}
func (cm *CacheMap) IsActive(id string) bool {
	cm.Lock()
	defer cm.Unlock()
	_, ok := cm.cache[id]
	return ok
}
