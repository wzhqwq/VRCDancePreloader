package cache_map

import (
	"errors"
	"fmt"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager/entry"
)

func (cm *baseCacheMap) Remove(id string) error {
	for {
		cm.mu.Lock()

		if cm.shuttingDown {
			cm.mu.Unlock()
			return ErrCacheMapShuttingDown
		}

		slot, ok := cm.cache[id]
		if !ok {
			slot = &cacheSlot{
				state: slotRemoving,
				done:  make(chan struct{}),
			}
			cm.cache[id] = slot

			cm.mu.Unlock()

			return cm.removePersistent(id, slot, nil)
		}

		switch slot.state {
		case slotReady:
			if slot.refs != 0 {
				refs := slot.refs
				cm.mu.Unlock()

				return fmt.Errorf(
					"%w: id=%q refs=%d",
					ErrCacheEntryActive,
					id,
					refs,
				)
			}

			slot.generation++

			if slot.idleTimer != nil {
				slot.idleTimer.Stop()
				slot.idleTimer = nil
			}

			slot.state = slotRemoving
			slot.done = make(chan struct{})

			e := slot.entry

			cm.mu.Unlock()

			return cm.removePersistent(id, slot, e)

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

func (cm *baseCacheMap) removePersistent(
	id string,
	expected *cacheSlot,
	e entry.CDNEntry,
) error {
	var closeErr error
	if e != nil {
		closeErr = e.Close()
	}

	var removeErr error
	if cm.removeCacheFn == nil {
		removeErr = errors.New("removeCacheFn is nil")
	} else {
		removeErr = cm.removeCacheFn(id)
	}

	cm.mu.Lock()

	current, ok := cm.cache[id]
	if ok && current == expected {
		delete(cm.cache, id)
	}

	done := expected.done
	expected.done = nil

	cm.mu.Unlock()
	close(done)

	switch {
	case closeErr != nil && removeErr != nil:
		return errors.Join(
			fmt.Errorf("close cache entry %q: %w", id, closeErr),
			fmt.Errorf("remove persistent cache %q: %w", id, removeErr),
		)

	case closeErr != nil:
		return fmt.Errorf(
			"close cache entry %q: %w",
			id,
			closeErr,
		)

	case removeErr != nil:
		return fmt.Errorf(
			"remove persistent cache %q: %w",
			id,
			removeErr,
		)

	default:
		return nil
	}
}
