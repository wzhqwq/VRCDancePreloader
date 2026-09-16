package cache_manager

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager/cache_map"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager/entry"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var nilLogger = utils.NewLogger("Closed Cache")

type cdnFileSession struct {
	id string

	cacheMap cache_map.CacheMap
	entry    entry.CDNEntry

	entryEm sync.RWMutex

	s *Service
}

func (m *cdnFileSession) GetResource(ctx context.Context) (io.ReadSeeker, int64, time.Time, error) {
	return m.entry.GetResource(ctx)
}

func (m *cdnFileSession) UpdateReqRangeStart(start int64) {
	m.entry.UpdateReqRangeStart(start)
}

func (m *cdnFileSession) IsForceExpiration() bool {
	return m.s.Cfg.ForceExpiration
}

func (m *cdnFileSession) Open(id string, loggers ...utils.LoggerImpl) (err error) {
	m.entryEm.Lock()
	defer m.entryEm.Unlock()

	if m.entry != nil {
		return
	}
	m.entry, err = m.cacheMap.Open(id)
	if err == nil {
		m.id = id
		for _, l := range loggers {
			l.InfoLn("Open cache entry:", id)
		}
	}
	return
}

func (m *cdnFileSession) Close(loggers ...utils.LoggerImpl) {
	m.entryEm.Lock()
	defer m.entryEm.Unlock()

	m.cacheMap.Release(m.id)
	m.entry = nil
	if m.id != "" {
		for _, l := range loggers {
			l.InfoLn("Release cache entry:", m.id)
		}
		m.id = ""
	}
}

func (m *cdnFileSession) AcquireFile() (types.DeferredReadableFile, error) {
	m.entryEm.RLock()
	defer m.entryEm.RUnlock()

	if m.entry == nil {
		return nil, errors.New("cache entry might be closed")
	}
	return m.entry.AcquireFile()
}

func (m *cdnFileSession) ReleaseFile() {
	m.entryEm.RLock()
	defer m.entryEm.RUnlock()

	if m.entry == nil {
		return
	}
	m.entry.ReleaseFile()
}

func (m *cdnFileSession) Etag() string {
	m.entryEm.RLock()
	defer m.entryEm.RUnlock()

	if m.entry == nil {
		return ""
	}
	return m.entry.Etag()
}

func (m *cdnFileSession) ReconcileRemoteInfo(info *types.RemoteHttpResourceInfo) {
	m.entryEm.RLock()
	defer m.entryEm.RUnlock()

	if m.entry == nil {
		return
	}
	m.entry.ReconcileRemoteInfo(info)
}

func (m *cdnFileSession) MarkComplete() {
	m.entryEm.RLock()
	defer m.entryEm.RUnlock()

	if m.entry == nil {
		return
	}
	m.entry.MarkComplete()
}

func (m *cdnFileSession) Logger() utils.LoggerImpl {
	m.entryEm.RLock()
	defer m.entryEm.RUnlock()

	if m.entry == nil {
		return nilLogger
	}
	return m.entry.Logger()
}

var _ types.CDNFileSession = &cdnFileSession{}

func newCdnFileSession(cacheMap cache_map.CacheMap, s *Service) *cdnFileSession {
	return &cdnFileSession{
		cacheMap: cacheMap,

		s: s,
	}
}
