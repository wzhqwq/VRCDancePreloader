package cache_map

import (
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager/entry"
)

type slotState uint8

const (
	slotCreating slotState = iota
	slotReady
	slotClosing
	slotRemoving
)

type cacheSlot struct {
	// 以下字段全部由 baseCacheMap.mu 保护。
	state slotState
	entry entry.CDNEntry

	refs int64

	// generation 用于废弃旧的空闲关闭任务。
	generation uint64
	idleTimer  *time.Timer

	// creating、closing 和 removing 状态下非 nil。
	// 等待者复制 channel 后释放 cm.mu，再等待其关闭。
	done chan struct{}
}

func (cm *baseCacheMap) armIdleCloseLocked(
	id string,
	slot *cacheSlot,
) {
	slot.generation++
	generation := slot.generation

	if slot.idleTimer != nil {
		slot.idleTimer.Stop()
	}

	slot.idleTimer = time.AfterFunc(cm.idleTimeout, func() {
		cm.evictIdle(id, slot, generation)
	})
}

func (cm *baseCacheMap) evictIdle(
	id string,
	expected *cacheSlot,
	generation uint64,
) {
	cm.mu.Lock()

	current, ok := cm.cache[id]
	if !ok ||
		current != expected ||
		cm.shuttingDown ||
		current.state != slotReady ||
		current.refs != 0 ||
		current.generation != generation {

		cm.mu.Unlock()
		return
	}

	current.state = slotClosing
	current.done = make(chan struct{})
	current.idleTimer = nil

	e := current.entry
	done := current.done

	cm.mu.Unlock()

	closeErr := e.Close()

	cm.mu.Lock()

	current, stillRegistered := cm.cache[id]
	if stillRegistered && current == expected {
		delete(cm.cache, id)
	}

	expected.done = nil

	cm.mu.Unlock()
	close(done)

	if closeErr != nil {
		cm.logger.ErrorLn(
			"Failed to close idle cache entry:",
			id,
			closeErr,
		)
		return
	}

	cm.logger.InfoLn("Closed idle cache entry:", id)
}
