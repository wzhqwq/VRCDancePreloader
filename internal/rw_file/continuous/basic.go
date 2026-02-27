package continuous

import (
	"context"
	"io"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/trunk"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type File struct {
	rw_file.BaseFile

	fragment *trunk.Fragment

	em *utils.EventManager[int64]
}

func (f *File) Init(contentLength int64, lastModified time.Time) error {
	err := f.File.Init(contentLength, lastModified)
	if err != nil {
		return err
	}

	f.fragment.Length = 0

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
func (f *File) GetDownloadOffset() int64 {
	return f.fragment.Length
}
func (f *File) GetDownloadedBytes() int64 {
	return f.fragment.Length
}

func (f *File) RequestRs(ctx context.Context) io.ReadSeeker {
	if f.File.FullSize == 0 {
		return nil
	}
	return rw_file.NewRSWithContext(f, f.File.FullSize, ctx)
}

func NewFile(baseName string) *File {
	f := &File{
		BaseFile: rw_file.ConstructBaseFile(baseName),

		em: utils.NewEventManager[int64](),
	}

	if f.File == nil {
		return nil
	}

	f.fragment = f.File.ToFragments()[0]

	return f
}
