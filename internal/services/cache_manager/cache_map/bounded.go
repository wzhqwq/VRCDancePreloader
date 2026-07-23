package cache_map

import (
	"errors"
	"fmt"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type boundedCacheMap struct {
	baseCacheMap

	configMu      sync.RWMutex
	maxSize       int64
	keepFavorites bool

	cleanupRunMu sync.Mutex

	cleanupScheduleMu sync.Mutex
	cleanupScheduled  bool
	cleanupPending    bool
}

func NewBoundedCacheMap(name string, maxSize int64, keepFavorites bool, newEntryFn NewEntryFn, removeCacheFn func(name string) error) CacheMap {
	m := &boundedCacheMap{
		baseCacheMap: baseCacheMap{
			cache: make(map[string]*cacheSlot),

			newEntryFn:    newEntryFn,
			removeCacheFn: removeCacheFn,

			logger: utils.NewLogger(name),
		},
		maxSize:       maxSize,
		keepFavorites: keepFavorites,
	}
	m.onEntryInactive = m.RequestCleanUp

	return m
}

func (c *boundedCacheMap) RequestCleanUp() {
	if c.shuttingDown {
		return
	}

	c.cleanupScheduleMu.Lock()

	if c.cleanupScheduled {
		c.cleanupPending = true
		c.cleanupScheduleMu.Unlock()
		return
	}

	c.cleanupScheduled = true
	c.cleanupScheduleMu.Unlock()

	c.wg.Go(c.cleanupWorker)
}

func (c *boundedCacheMap) cleanupWorker() {
	for {
		c.CleanUp()

		c.cleanupScheduleMu.Lock()

		if !c.cleanupPending {
			c.cleanupScheduled = false
			c.cleanupScheduleMu.Unlock()
			return
		}

		c.cleanupPending = false
		c.cleanupScheduleMu.Unlock()
	}
}

func (c *boundedCacheMap) CleanUp() {
	c.cleanupRunMu.Lock()
	defer c.cleanupRunMu.Unlock()

	c.cleanUpOnce()
}

type cleanupSnapshot struct {
	totalSize int64
	records   []persistence.CleanupCandidate
}

func (c *boundedCacheMap) loadCleanupSnapshot() (
	cleanupSnapshot,
	error,
) {
	tx, err := persistence.BeginCacheCleanupTx()
	if err != nil {
		return cleanupSnapshot{}, fmt.Errorf(
			"begin cache cleanup transaction: %w",
			err,
		)
	}
	defer tx.Finish()

	totalSize, err := tx.Summarize("video")
	if err != nil {
		return cleanupSnapshot{}, fmt.Errorf(
			"summarize video cache: %w",
			err,
		)
	}

	records, err := tx.ListCandidates("video")
	if err != nil {
		return cleanupSnapshot{}, fmt.Errorf(
			"list video cache cleanup candidates: %w",
			err,
		)
	}

	return cleanupSnapshot{
		totalSize: totalSize,
		records:   records,
	}, nil
}

func (c *boundedCacheMap) cleanupConfig() CleanupConfig {
	c.configMu.RLock()
	defer c.configMu.RUnlock()

	return CleanupConfig{
		MaxSize:       c.maxSize,
		KeepFavorites: c.keepFavorites,
	}
}

func (c *boundedCacheMap) cleanUpOnce() {
	var initialSize int64 = -1

	for {
		cfg := c.cleanupConfig()

		snapshot, err := c.loadCleanupSnapshot()
		if err != nil {
			c.logger.ErrorLn("Failed to inspect cache for cleanup:", err)
			return
		}

		if initialSize < 0 {
			initialSize = snapshot.totalSize
		}

		if snapshot.totalSize <= cfg.MaxSize {
			if initialSize > snapshot.totalSize {
				c.logger.InfoLn(
					"Cleaned up cache,",
					utils.PrettyByteSize(initialSize),
					"->",
					utils.PrettyByteSize(snapshot.totalSize),
				)
			}
			return
		}

		c.logger.InfoLn(
			"Cleaning up cache:",
			utils.PrettyByteSize(snapshot.totalSize),
			"limit:",
			utils.PrettyByteSize(cfg.MaxSize),
		)

		removedAny := false
		estimatedSize := snapshot.totalSize

		for _, record := range snapshot.records {
			cfg = c.cleanupConfig()

			if estimatedSize <= cfg.MaxSize {
				break
			}

			if cfg.KeepFavorites &&
				persistence.IsFavorite(record.ID) {

				continue
			}

			err := c.Remove(record.ID)

			switch {
			case err == nil:
				removedAny = true

				if record.Size >= estimatedSize {
					estimatedSize = 0
				} else {
					estimatedSize -= record.Size
				}

			case errors.Is(err, ErrCacheEntryActive):
				continue

			case errors.Is(err, ErrCacheMapShuttingDown):
				return

			default:
				c.logger.ErrorLn(
					"Failed to remove cache entry",
					record.ID+":",
					err,
				)
			}
		}

		if !removedAny {
			c.logger.WarnLn(
				"Cache remains above the configured limit, but no "+
					"removable entries are currently available:",
				utils.PrettyByteSize(snapshot.totalSize),
				"limit:",
				utils.PrettyByteSize(cfg.MaxSize),
			)
			return
		}
	}
}

func (c *boundedCacheMap) SetCleanup(cfg CleanupConfig) {
	c.configMu.Lock()

	shrunken := cfg.MaxSize < c.maxSize || (c.keepFavorites && !cfg.KeepFavorites)

	c.maxSize = cfg.MaxSize
	c.keepFavorites = cfg.KeepFavorites

	c.configMu.Unlock()

	if shrunken {
		c.RequestCleanUp()
	}
}
