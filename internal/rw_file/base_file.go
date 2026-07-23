package rw_file

import (
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/trunk"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_fs"
)

type BaseFile struct {
	baseName string
	File     *trunk.File

	//cacheFs *cache_fs.CacheFS
}

func (f *BaseFile) Close() error {
	return f.File.Close()
}

func (f *BaseFile) TotalLen() int64 {
	return f.File.FullSize
}
func (f *BaseFile) ModTime() time.Time {
	return f.File.LastModified
}
func (f *BaseFile) Stat() (int64, time.Time) {
	return f.File.Stat()
}
func (f *BaseFile) Init(contentLength int64, lastModified time.Time) error {
	return f.File.Init(contentLength, lastModified)
}
func (f *BaseFile) IsComplete() bool {
	return f.File.Completed
}

func ConstructBaseFile(baseName string, cacheFs *cache_fs.CacheFS) BaseFile {
	return BaseFile{
		baseName: baseName,
		File:     trunk.NewTrunkFile(baseName, cacheFs),
		//cacheFs:  cacheFs,
	}
}
