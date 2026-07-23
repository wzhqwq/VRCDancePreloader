package cache_map

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager/entry"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type CacheMap interface {
	CloseAll()
	Open(id string) (entry.CDNEntry, error)
	Release(id string)
	IsActive(id string) bool
	Remove(id string) error

	// optional

	RequestCleanUp()
	SetCleanup(cfg CleanupConfig)
}

var (
	ErrCacheMapShuttingDown = errors.New("cache map is shutting down")
	ErrCacheEntryActive     = errors.New("cache entry is active")
	ErrNilCacheEntry        = errors.New("cache entry factory returned nil")
)

type CleanupConfig struct {
	MaxSize       int64
	KeepFavorites bool
}

type NewEntryFn func(id string) (entry.CDNEntry, error)
type RemoveCacheFn func(name string) error

type baseCacheMap struct {
	mu sync.Mutex

	cache        map[string]*cacheSlot
	shuttingDown bool
	shutdownDone chan struct{}

	idleTimeout time.Duration

	newEntryFn    NewEntryFn
	removeCacheFn RemoveCacheFn

	logger utils.LoggerImpl

	onEntryInactive func()

	wg sync.WaitGroup
}

func NewCacheMap(name string, newEntryFn NewEntryFn, removeCacheFn RemoveCacheFn) CacheMap {
	return &baseCacheMap{
		cache: make(map[string]*cacheSlot),

		idleTimeout: time.Second,

		newEntryFn:    newEntryFn,
		removeCacheFn: removeCacheFn,

		logger: utils.NewLogger(name),
	}
}

func (cm *baseCacheMap) CloseAll() {
	cm.mu.Lock()

	if cm.shuttingDown {
		done := cm.shutdownDone
		cm.mu.Unlock()

		if done != nil {
			<-done
		}
		return
	}

	cm.shuttingDown = true
	cm.shutdownDone = make(chan struct{})

	cm.wg.Wait()

	var (
		entries []entry.CDNEntry
		pending []chan struct{}
	)

	for id, slot := range cm.cache {
		slot.generation++

		if slot.idleTimer != nil {
			slot.idleTimer.Stop()
			slot.idleTimer = nil
		}

		switch slot.state {
		case slotReady:
			entries = append(entries, slot.entry)
			delete(cm.cache, id)

		case slotCreating, slotClosing, slotRemoving:
			pending = append(pending, slot.done)

		default:
			panic("cache map: invalid slot state")
		}
	}

	shutdownDone := cm.shutdownDone
	cm.mu.Unlock()

	for _, e := range entries {
		if err := e.Close(); err != nil {
			cm.logger.ErrorLn(
				"Failed to close cache entry during shutdown:",
				err,
			)
		}
	}

	for _, done := range pending {
		<-done
	}

	cm.mu.Lock()
	close(shutdownDone)
	cm.mu.Unlock()
}

func (cm *baseCacheMap) Open(id string) (entry.CDNEntry, error) {
	for {
		cm.mu.Lock()

		if cm.shuttingDown {
			cm.mu.Unlock()
			return nil, ErrCacheMapShuttingDown
		}

		slot, ok := cm.cache[id]
		if !ok {
			slot = &cacheSlot{
				state: slotCreating,
				done:  make(chan struct{}),
			}
			cm.cache[id] = slot

			cm.mu.Unlock()

			return cm.createAndAcquire(id, slot)
		}

		switch slot.state {
		case slotReady:
			slot.refs++

			slot.generation++

			if slot.idleTimer != nil {
				slot.idleTimer.Stop()
				slot.idleTimer = nil
			}

			e := slot.entry
			cm.mu.Unlock()
			return e, nil

		case slotCreating, slotClosing, slotRemoving:
			done := slot.done
			cm.mu.Unlock()

			<-done
			continue

		default:
			cm.mu.Unlock()
			panic("cache map: invalid slot state")
		}
	}
}

func (cm *baseCacheMap) createAndAcquire(id string, expected *cacheSlot) (entry.CDNEntry, error) {
	created, err := cm.newEntryFn(id)
	if err == nil && created == nil {
		err = ErrNilCacheEntry
	}

	if err == nil {
		created.EnsureOpen()
	}

	if err != nil {
		if created != nil {
			if closeErr := created.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf(
					"close failed cache entry %q: %w",
					id,
					closeErr,
				))
			}
		}

		cm.finishFailedCreation(id, expected)
		return nil, err
	}

	cm.mu.Lock()

	current, stillRegistered := cm.cache[id]
	if cm.shuttingDown ||
		!stillRegistered ||
		current != expected ||
		current.state != slotCreating {

		if stillRegistered && current == expected {
			delete(cm.cache, id)
		}

		done := expected.done
		expected.done = nil

		cm.mu.Unlock()

		closeErr := created.Close()
		close(done)

		if closeErr != nil {
			cm.logger.ErrorLn(
				"Failed to close cache entry created during shutdown:",
				id,
				closeErr,
			)
		}

		return nil, ErrCacheMapShuttingDown
	}

	expected.entry = created
	expected.state = slotReady
	expected.refs = 1
	expected.generation++

	done := expected.done
	expected.done = nil

	cm.mu.Unlock()
	close(done)

	return created, nil
}

func (cm *baseCacheMap) finishFailedCreation(id string, expected *cacheSlot) {
	cm.mu.Lock()

	current, ok := cm.cache[id]
	if ok && current == expected {
		delete(cm.cache, id)
	}

	done := expected.done
	expected.done = nil

	cm.mu.Unlock()
	close(done)
}

func (cm *baseCacheMap) Release(id string) {
	var requestCleanup func()

	cm.mu.Lock()

	slot, ok := cm.cache[id]
	if !ok {
		cm.mu.Unlock()
		return
	}

	if slot.state != slotReady {
		cm.mu.Unlock()
		panic(fmt.Sprintf(
			"cache map: Release(%q) called while slot state is %d",
			id,
			slot.state,
		))
	}

	if slot.refs <= 0 {
		cm.mu.Unlock()
		panic(fmt.Sprintf(
			"cache map: Release(%q) called without matching Open",
			id,
		))
	}

	slot.refs--

	if slot.refs == 0 && !cm.shuttingDown {
		cm.armIdleCloseLocked(id, slot)
		requestCleanup = cm.onEntryInactive
	}

	cm.mu.Unlock()

	if requestCleanup != nil {
		requestCleanup()
	}
}

func (cm *baseCacheMap) IsActive(id string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	slot, ok := cm.cache[id]
	return ok && slot.state == slotReady && slot.refs > 0
}
func (cm *baseCacheMap) RequestCleanUp() {
}

func (cm *baseCacheMap) SetCleanup(_ CleanupConfig) {
}
