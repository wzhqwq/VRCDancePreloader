package entry

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
)

func (e *BaseEntry) syncWithFS() {
	e.meta = e.getOrRecordMeta()
	e.meta.Access()
	e.meta.SetPartial(!e.workingFile.IsComplete())
	e.readEtag()
}

func (e *BaseEntry) getOrRecordMeta() *persistence.CacheMeta {
	return persistence.AddCacheMetaIfNotExists(e.id, "video", func() *persistence.CacheMeta {
		size, created := e.workingFile.Stat()
		return persistence.NewCacheMeta(e.id, "video", size, e.workingFile.ModTime(), created)
	})
}
