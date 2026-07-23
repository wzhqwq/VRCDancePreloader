package entry

import (
	"regexp"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/continuous"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/fragmented"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/legacy_file"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_fs"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var supportedVideoIdRegex = regexp.MustCompile("^(?:pypy|wanna|dudu|yt|bili)]")

type VideoEntry struct {
	BaseCDNEntry

	baseName string

	checkingMutex sync.Mutex

	cacheFs *cache_fs.CacheFS

	fileFormat int
}

func NewVideoEntry(id string, format int, cacheFs *cache_fs.CacheFS) (CDNEntry, error) {
	logger := utils.NewLogger("Cached Video " + id)

	if !supportedVideoIdRegex.MatchString(id) {
		return nil, ErrNotSupported
	}

	e := &VideoEntry{
		BaseCDNEntry: BaseCDNEntry{
			id: id,

			logger: logger,

			etag: &Etag{
				id:      id,
				logger:  logger,
				cacheFs: cacheFs,
			},
		},
		baseName:   "video$" + id,
		fileFormat: format,
		cacheFs:    cacheFs,
	}
	e.upgradeFn = e.upgradeFile
	e.openFileFn = e.openFile

	return e, nil
}

func (e *VideoEntry) checkLegacy() bool {
	if e.cacheFs.Exists(e.baseName + ".vrcdp") {
		return false
	}
	if e.cacheFs.Exists(e.baseName+".mp4") || e.cacheFs.Exists(e.baseName+".mp4.dl") {
		return true
	}

	return false
}

func (e *VideoEntry) openFile() types.DeferredReadableFile {
	if e.checkLegacy() {
		return legacy_file.NewFile(e.baseName, e.cacheFs)
	}

	switch e.fileFormat {
	case 1:
		return continuous.NewFile(e.baseName, e.cacheFs)
	case 2:
		return fragmented.NewFile(e.baseName, e.cacheFs)
	}

	return nil
}

func (e *VideoEntry) upgradeFile() {
	if e.workingFile == nil {
		e.logger.WarnLn("Try to upgrade a closed file")
		return
	}
	if _, ok := e.workingFile.(*legacy_file.File); !ok {
		e.logger.WarnLn("Try to upgrade a non-legacy file")
		return
	}

	err := e.workingFile.Close()
	if err != nil {
		e.logger.ErrorLn("Failed to close working file", err)
		return
	}
	e.workingFile = nil

	err = e.cacheFs.DeleteWithoutExt("video$" + e.id)
	if err != nil {
		e.logger.ErrorLn("Failed to delete old file", err)
	}

	e.openFile()
}
