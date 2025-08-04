package rw_file

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

type File struct {
	baseName string
	totalLen int64

	downloaded int64

	em *utils.EventManager[int64]

	fileMutex sync.RWMutex
	file      *os.File
}

func (f *File) Append(bytes []byte) (int, error) {
	f.fileMutex.RLock()
	defer func() {
		f.fileMutex.RUnlock()

		if f.downloaded == f.totalLen {
			go func() {
				err := f.Rename(f.baseName)
				if err != nil {
					log.Printf("Failed to rename file: %s, err: %s", f.baseName, err)
				}
			}()
		}
	}()

	offset, err := f.file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	n, err := f.file.WriteAt(bytes, offset)
	if err != nil {
		return 0, err
	}
	f.downloaded += int64(n)

	return n, nil
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

func (f *File) RequestRange(offset, length int64, closeCh chan struct{}) error {
	if f.IsComplete() {
		return nil
	}

	ch := f.em.SubscribeEvent()
	defer ch.Close()

	for {
		select {
		case l := <-ch.Channel:
			if l >= offset+length {
				return nil
			}
		case <-closeCh:
			return io.ErrClosedPipe
		}
	}
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	f.fileMutex.RLock()
	defer f.fileMutex.RUnlock()

	return f.file.ReadAt(p, off)
}

func (f *File) Rename(to string) error {
	f.fileMutex.Lock()
	defer f.fileMutex.Unlock()

	dst, err := openFile(to)
	if err != nil {
		return err
	}
	_, err = f.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	_, err = io.Copy(dst, f.file)
	if err != nil {
		dst.Close()
		os.Remove(to)
		return err
	}
	err = dst.Sync()
	if err != nil {
		dst.Close()
		os.Remove(to)
		return err
	}

	f.file.Close()
	os.Remove(f.file.Name())

	f.file = dst

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

func (f *File) GetTotalLength(getter func() int64) int64 {
	if f.totalLen == 0 {
		f.totalLen = getter()
		f.file.Truncate(f.totalLen)
	}
	return f.totalLen
}

func (f *File) RequestRsc() io.ReadSeekCloser {
	if f.totalLen == 0 {
		return nil
	}
	return NewRsc(f, f.totalLen)
}

func NewFile(baseName string) *File {
	file, _ := openFile(baseName)
	if file == nil {
		return nil
	}
	return &File{
		baseName: baseName,
		totalLen: getFileSize(baseName),

		fileMutex: sync.RWMutex{},
		file:      file,
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
