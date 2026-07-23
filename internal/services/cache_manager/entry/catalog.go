package entry

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/continuous"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_fs"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type CatalogEntry struct {
	BaseCDNEntry

	baseName string

	cacheFs *cache_fs.CacheFS
}

func NewCatalogEntry(id string, cacheFs *cache_fs.CacheFS) CDNEntry {
	logger := utils.NewLogger("Cached Catalog of " + id)

	e := &CatalogEntry{
		BaseCDNEntry: BaseCDNEntry{
			id: id,

			logger: logger,
		},
		baseName: "catalog$" + id,
		cacheFs:  cacheFs,
	}
	e.openFileFn = e.openFile

	return e
}

func (e *CatalogEntry) openFile() types.DeferredReadableFile {
	return continuous.NewFile(e.baseName, e.cacheFs)
}
