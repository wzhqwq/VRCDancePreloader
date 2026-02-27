package trunk

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache/cache_fs"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// 16KB per trunk
const bytesPerTrunk = 1024 * 16

// 256MB capacity
// It's enough for a dance video. If a file exceed this size, we will fall back to legacy cache
const capacity = 1024 * 1024 * 256

var logger = utils.NewLogger("Cache File")

type File struct {
	file         *os.File
	trunks       []byte
	LastModified time.Time
	FullSize     int64

	// states
	Completed bool

	readerWg sync.WaitGroup
	a        sync.Once
}

func NewTrunkFile(baseName string) *File {
	name := baseName + ".vrcdp"

	f, ok := cache_fs.Get(name)
	if !ok {
		var err error
		f, err = cache_fs.Create(name)
		if err != nil {
			logger.ErrorLn("Failed to open cache file:", err)
			return nil
		}
	}

	tf := &File{
		file:         f,
		trunks:       make([]byte, numTrunks),
		LastModified: time.Time{},
	}
	if !tf.tryRead() {
		if !tf.tryCreate() {
			return nil
		}
	}
	return tf
}

func (f *File) AppendTo(frag *Fragment, data []byte) error {
	offset := bodyOffset + frag.End()

	n, err := f.file.WriteAt(data, offset)
	if err != nil {
		return err
	}

	frag.Length += int64(n)

	f.FillTrunks(frag)

	return nil
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	f.readerWg.Add(1)
	defer f.readerWg.Done()

	offset := bodyOffset + off
	return f.file.ReadAt(p, offset)
}

func (f *File) Name() string {
	return f.file.Name()
}

func (f *File) Close() error {
	f.readerWg.Wait()
	return f.file.Close()
}

var ErrWriteFailed = errors.New("failed to write changes to disk")

func (f *File) Init(contentLength int64, lastModified time.Time) error {
	f.FullSize = contentLength
	f.LastModified = lastModified

	// remove complete flag
	f.Completed = false
	if !f.writeStates() {
		return ErrWriteFailed
	}

	// fill zeros
	for i := 0; i < numTrunks; i++ {
		f.trunks[i] = 0
	}
	if !f.writeTrunks() {
		return ErrWriteFailed
	}

	// write full size
	if !f.writeFullSize() {
		return ErrWriteFailed
	}
	// write last modified time
	if !f.writeLastModifiedTime() {
		return ErrWriteFailed
	}

	// resize
	err := f.file.Truncate(bodyOffset + contentLength)
	if err != nil {
		return fmt.Errorf("failed to truncate trunk file: %v", err)
	}

	return nil
}

func (f *File) Stat() (int64, time.Time) {
	var created time.Time
	if stat, err := f.file.Stat(); err != nil {
		if attr, ok := stat.Sys().(*syscall.Win32FileAttributeData); ok {
			created = time.Unix(0, attr.CreationTime.Nanoseconds())
		}
	}
	return bodyOffset + f.FullSize, created
}

func (f *File) MarkCompleted() {
	f.Completed = true
	f.writeStates()
}
func (f *File) IsSuffix(frag *Fragment) bool {
	return f.FullSize > 0 && frag.End() >= f.FullSize
}
