package entry

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
)

func (e *BaseEntry) syncWithFS() {
	if e.workingFile == nil {
		return
	}

	e.meta = e.getOrRecordMeta()
	e.meta.Access()
	e.meta.SetPartial(!e.workingFile.IsComplete())
	if e.meta.CreatedTime.IsZero() || e.meta.RemoteLastModified != e.workingFile.ModTime() {
		size, created := e.workingFile.Stat()
		err := e.meta.UpdateInfo(size, e.workingFile.ModTime(), created)
		if err != nil {
			e.logger.ErrorLn("Failed to update meta info:", err)
		}
	}
	e.readEtag()
}

func (e *BaseEntry) getOrRecordMeta() *persistence.CacheMeta {
	return persistence.AddCacheMetaIfNotExists(e.id, "video", func() *persistence.CacheMeta {
		size, created := e.workingFile.Stat()
		return persistence.NewCacheMeta(e.id, "video", size, e.workingFile.ModTime(), created)
	})
}

func (e *BaseEntry) checkIfCompleteAndSync() bool {
	complete := e.workingFile.IsComplete()
	if complete && e.meta.Partial {
		e.meta.SetPartial(false)
	}
	return complete
}
