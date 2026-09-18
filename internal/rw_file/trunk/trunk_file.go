package trunk

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_fs"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// 16KB per trunk
const bytesPerTrunk = 1024 * 16

// 256MB capacity.
//
// It is enough for a dance video, and it is a hard limit of this backend: the
// bitmap has exactly one bit per bytesPerTrunk, so a larger file cannot be
// indexed at all. Init refuses one with ErrContentTooLarge instead of writing
// past the end of the bitmap.
//
// This comment used to promise a fall back to the legacy cache. No such fallback
// was ever implemented, which is what made the overflow look handled. Lifting the
// limit belongs to a future file format version rather than to an automatic
// downgrade.
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

func NewTrunkFile(baseName string, cacheFs *cache_fs.CacheFS) *File {
	name := baseName + ".vrcdp"

	f, ok := cacheFs.Get(name)
	if !ok {
		var err error
		f, err = cacheFs.Create(name)
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

// ErrContentTooLarge reports a file that does not fit the trunk bitmap: the
// bitmap has one bit per bytesPerTrunk and is sized by capacity, so a bigger
// file cannot be indexed at all. Callers are expected to fall back to a backend
// that does not need a bitmap.
var ErrContentTooLarge = errors.New("content is larger than the trunk cache can index")

// MaxSize is the largest file the trunk bitmap can index. It is exported so that
// callers can pick a different backend before opening a file, instead of
// discovering the limit from ErrContentTooLarge.
func MaxSize() int64 { return capacity }

func (f *File) Init(contentLength int64, lastModified time.Time) error {
	// Refuse an oversized file up front. Without this the fixed size bitmap is
	// silently too small and every write past capacity indexes out of range
	// (see FillTrunks).
	if contentLength > capacity {
		return fmt.Errorf("%w: %d bytes, the limit is %d", ErrContentTooLarge, contentLength, capacity)
	}

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
	if stat, err := f.file.Stat(); err == nil {
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
