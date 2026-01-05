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

func (f *File) Save() error {
	f.fileMutex.Lock()
	defer f.fileMutex.Unlock()

	if f.file.Name() != f.baseName {
		err := os.Rename(f.baseName+".dl", f.baseName)
		if err != nil {
			return err
		}

		f.file, err = openFile(f.baseName)
	}

	return nil
}

func (f *File) Rename(to string) error {
	f.fileMutex.Lock()
	defer f.fileMutex.Unlock()

	err := os.Rename(f.baseName+".dl", to)
	if err != nil {
		return err
	}

	f.file, err = openFile(f.baseName)

	return nil
}

func (f *File) ModTime() time.Time {
	f.fileMutex.Lock()
	defer f.fileMutex.Unlock()

	info, err := f.file.Stat()
	if err != nil {
		return time.Now()
	}
	return info.ModTime()
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
	totalLen := getFileSize(baseName)
	downloaded := totalLen

	file, _ := openFile(baseName)
	if file == nil {
		downloaded = getFileSize(baseName + ".dl")
		file, _ = os.Open(baseName + ".dl")
		if file == nil {
			return nil
		}
	}

	return &File{
		baseName:   baseName,
		totalLen:   totalLen,
		downloaded: downloaded,

		em: utils.NewEventManager[int64](),

		file: file,
	}
}

func openFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
}
func getFileSize(path string) int64 {
	stat, err := os.Stat(path)
	if err != nil {
		return 0
	}

	return stat.Size()
}
