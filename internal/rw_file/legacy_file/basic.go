package legacy_file

import (
	"context"
	"io"
	"os"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewLogger("Cache File (Legacy)")

type File struct {
	baseName string
	totalLen int64

	downloaded int64

	em *utils.EventManager[int64]

	fileMutex sync.RWMutex
	file      *os.File
}

func (f *File) Clear() error {
	f.fileMutex.RLock()
	defer f.fileMutex.RUnlock()

	err := f.file.Truncate(0)
	if err != nil {
		return err
	}

	f.downloaded = 0

	return nil
}

func (f *File) IsRequestFulfilled() bool {
	return f.IsComplete()
}

func (f *File) NotifyRequestStart(start int64) {
	// Do nothing
}
func (f *File) MarkDownloading() {
	// Do nothing
}

func (f *File) Close() error {
	f.fileMutex.RLock()
	defer f.fileMutex.RUnlock()

	return f.file.Close()
}

func (f *File) ModTime() time.Time {
	return time.Time{}
}

func (f *File) GetDownloadOffset() int64 {
	return f.downloaded
}

func (f *File) GetDownloadedBytes() int64 {
	return f.downloaded
}

func (f *File) IsComplete() bool {
	return f.totalLen > 0 && f.downloaded >= f.totalLen
}

func (f *File) TotalLen() int64 {
	return f.totalLen
}
func (f *File) UpdateRemoteInfo(contentLength int64, _ time.Time) {
	f.totalLen = contentLength
}

func (f *File) RequestRs(ctx context.Context) io.ReadSeeker {
	if f.totalLen == 0 {
		return nil
	}
	return rw_file.NewRSWithContext(f, f.totalLen, ctx)
}

func NewFile(baseName string) *File {
	totalLen := int64(0)
	downloaded := int64(0)

	f, ok := cache_fs.Get(baseName + ".mp4")
	if ok {
		totalLen = getFileSize(f)
		downloaded = totalLen
	} else {
		f, ok = cache_fs.Get(baseName + ".mp4.dl")
		if ok {
			downloaded = getFileSize(f)
		}
	}

	return &File{
		baseName:   baseName,
		totalLen:   totalLen,
		downloaded: downloaded,

		em: utils.NewEventManager[int64](),

		file: f,
	}
}

func getFileSize(file *os.File) int64 {
	stat, err := file.Stat()
	if err != nil {
		return 0
	}

	return stat.Size()
}
