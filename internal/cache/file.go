package cache

import (
	"github.com/samber/lo"
	"io"
	"os"
	"sync"
)

type ReadableWritingFile struct {
	sync.Mutex
	//utils.LoggingMutex
	file    *os.File
	size    int64
	readers []*rsc
}

func (rw *ReadableWritingFile) Append(bytes []byte) error {
	rw.Lock()
	defer rw.Unlock()

	_, err := rw.file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	_, err = rw.file.Write(bytes)
	if err != nil {
		return err
	}
	rw.size += int64(len(bytes))

	rw.readers = lo.Filter(rw.readers, func(item *rsc, index int) bool {
		return !rw.readers[index].Closed
	})
	//log.Printf("New size: %d, %d readers", rw.size, len(rw.readers))
	lo.ForEach(rw.readers, func(reader *rsc, index int) {
		for {
			select {
			case reader.availableLengthCh <- rw.size:
				return
			default:
				<-reader.availableLengthCh
			}
		}
	})
	return nil
}
func (rw *ReadableWritingFile) Close() error {
	rw.Lock()
	defer rw.Unlock()

	for _, reader := range rw.readers {
		reader.Close()
	}
	return rw.file.Close()
}
func (rw *ReadableWritingFile) ReadAt(p []byte, off int64) (n int, err error) {
	rw.Lock()
	defer rw.Unlock()

	return rw.file.ReadAt(p, off)
}
func (rw *ReadableWritingFile) RequestRsc(total int64) io.ReadSeekCloser {
	rw.Lock()
	defer rw.Unlock()

	rsc := newRsc(rw, total)

	rw.readers = append(rw.readers, rsc)

	return rsc
}
func (rw *ReadableWritingFile) Rename(to string) error {
	rw.Lock()
	defer rw.Unlock()

	dst, err := openFile(to)
	if err != nil {
		return err
	}
	_, err = rw.file.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	size, err := io.Copy(dst, rw.file)
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

	rw.file.Close()
	os.Remove(rw.file.Name())

	rw.size = size
	rw.file = dst

	return nil
}

func newReadableWritingFile(path string) *ReadableWritingFile {
	file, _ := openFile(path)
	if file == nil {
		return nil
	}
	return &ReadableWritingFile{
		file: file,
		size: getFileSize(path),
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
