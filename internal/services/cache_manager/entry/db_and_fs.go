package entry

import (
	persistence2 "github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
)

func (e *BaseCDNEntry) updateMeta() error {
	size, created := e.workingFile.Stat()
	if e.etag != nil {
		size += e.etag.GetSize()
	}
	return e.meta.UpdateInfo(size, e.workingFile.ModTime(), created)
}

func (e *BaseCDNEntry) syncWithFS() {
	if e.workingFile == nil {
		return
	}

	e.meta = e.getOrRecordMeta()
	e.meta.Access()
	e.meta.SetPartial(!e.workingFile.IsComplete())
	if e.meta.CreatedTime.IsZero() || e.meta.RemoteLastModified != e.workingFile.ModTime() {
		err := e.updateMeta()
		if err != nil {
			e.logger.ErrorLn("Failed to update meta info:", err)
		}
	}
	if e.etag != nil {
		e.etag.Read()
	}
}

func (e *BaseCDNEntry) getOrRecordMeta() *persistence2.CacheMeta {
	return persistence2.AddCacheMetaIfNotExists(e.id, "video", func() *persistence2.CacheMeta {
		size, created := e.workingFile.Stat()
		return persistence2.NewCacheMeta(e.id, "video", size, e.workingFile.ModTime(), created)
	})
}
