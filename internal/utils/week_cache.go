package utils

import (
	"container/list"
	"sync"
	"weak"
)

// modified code from https://gist.github.com/hxzhouh/1945d4a1e5a6567f084628d60b63f125

type CacheItem[T any] struct {
	key   string
	value T
}
type WeakCache[T any] struct {
	sync.Mutex

	cache   map[string]weak.Pointer[list.Element] // Use weak references to store values
	storage Storage
}

// Storage is a fixed-length cache based on doubly linked tables and weak
type Storage struct {
	capacity int // Maximum size of the cache
	list     *list.List
}

// NewWeakCache creates a fixed-length weak reference cache.
func NewWeakCache[T any](capacity int) *WeakCache[T] {
	return &WeakCache[T]{
		cache:   make(map[string]weak.Pointer[list.Element]),
		storage: Storage{capacity: capacity, list: list.New()},
	}
}

// Set adds or updates cache entries
func (c *WeakCache[T]) Set(key string, value T) {
	c.Lock()
	defer c.Unlock()

	if elem, exists := c.cache[key]; exists {
		if elemValue := elem.Value(); elemValue != nil {
			elemValue.Value = &CacheItem[T]{key: key, value: value}
			c.storage.list.MoveToFront(elemValue)
			elemWeak := weak.Make(elemValue)
			c.cache[key] = elemWeak
			return
		} else {
			c.removeElement(key)
		}
	}
	// remove the oldest unused element if capacity is full
	if c.storage.list.Len() >= c.storage.capacity {
		c.evict()
	}

	// Add new element
	elem := c.storage.list.PushFront(&CacheItem[T]{key: key, value: value})
	elemWeak := weak.Make(elem)
	c.cache[key] = elemWeak
}

// Get gets the value of the cached item
func (c *WeakCache[T]) Get(key string) (T, bool) {
	c.Lock()
	defer c.Unlock()

	if elem, exists := c.cache[key]; exists {
		// Check if the weak reference is still valid
		if elemValue := elem.Value(); elemValue != nil {
			// Moving to the head of the chain indicates the most recent visit
			c.storage.list.MoveToFront(elemValue)
			return elemValue.Value.(*CacheItem[T]).value, true
		} else {
			c.removeElement(key)
		}
	}

	var nothing T

	return nothing, false
}

// evict removes the cache item that has not been used for the longest time
func (c *WeakCache[T]) evict() {
	if elem := c.storage.list.Back(); elem != nil {
		item := elem.Value.(*CacheItem[T])
		c.removeElement(item.key)
	}
}

// removeElement removes elements from chains and dictionaries.
func (c *WeakCache[T]) removeElement(key string) {
	if elem, exists := c.cache[key]; exists {
		// Check if the weak reference is still valid
		if elemValue := elem.Value(); elemValue != nil {
			c.storage.list.Remove(elemValue)
		}
		delete(c.cache, key)
	}
}

// Debug prints the contents of the cache
//func (c *WeakCache[T]) Debug() {
//	fmt.Println("Cache content:")
//	for k, v := range c.cache {
//		if v.Value() != nil {
//			fmt.Printf("Key: %s, Value: %v\n", k, v.Value().Value.(*CacheItem).value)
//		}
//	}
//}

func (c *WeakCache[T]) CleanCache() {
	c.storage.list.Init()
	c.storage.capacity = 0
}
