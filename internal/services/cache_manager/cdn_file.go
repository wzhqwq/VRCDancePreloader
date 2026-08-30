package cache_manager

import (
	"context"
	"io"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager/cache_map"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager/entry"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type cdnFileSession struct {
	id string

	cacheMap cache_map.CacheMap
	entry    entry.CDNEntry

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
	return m.entry.AcquireFile()
}

func (m *cdnFileSession) ReleaseFile() {
	m.entry.ReleaseFile()
}

func (m *cdnFileSession) Etag() string {
	return m.entry.Etag()
}

func (m *cdnFileSession) ReconcileRemoteInfo(info *types.RemoteHttpResourceInfo) {
	m.entry.ReconcileRemoteInfo(info)
}

func (m *cdnFileSession) MarkComplete() {
	m.entry.MarkComplete()
}

func (m *cdnFileSession) Logger() utils.LoggerImpl {
	return m.entry.Logger()
}

var _ types.CDNFileSession = &cdnFileSession{}

func newCdnFileSession(cacheMap cache_map.CacheMap, s *Service) *cdnFileSession {
	return &cdnFileSession{
		cacheMap: cacheMap,

		s: s,
	}
}
