package trunk

import (
	"os"
	"sync"
	"time"

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
	dlf, err := os.OpenFile(baseName+".vrcdp", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		logger.ErrorLn("Failed to open cache file:", err)
		return nil
	}

	f := &File{
		file:         dlf,
		trunks:       make([]byte, numTrunks),
		LastModified: time.Unix(0, 0),
	}
	if !f.tryRead() {
		if !f.tryCreate() {
			return nil
		}
	}
	return f
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

func (f *File) ClearTrunks() {
	// remove complete flag
	f.Completed = false
	f.writeStates()
	// fill zeros
	for i := 0; i < numTrunks; i++ {
		f.trunks[i] = 0
	}
	f.writeTrunks()
}

func (f *File) Init(contentLength int64, lastModified time.Time) {
	f.FullSize = contentLength
	f.LastModified = lastModified

	err := f.file.Truncate(bodyOffset + contentLength)
	if err != nil {
		logger.ErrorLn("Failed to truncate trunk file:", err)
	}

	f.writeFullSize()
	f.writeLastModifiedTime()
}

func (f *File) MarkCompleted() {
	f.Completed = true
	f.writeStates()
}
func (f *File) IsSuffix(frag *Fragment) bool {
	return f.FullSize > 0 && frag.End() >= f.FullSize
}
