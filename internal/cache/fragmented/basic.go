package fragmented

import (
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache/rw_file"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// continue the current fragment if the starting point is not more than 100KB beyond the fragment
const continueThreshold = int64(1024 * 100)

type File struct {
	fragments []*Fragment
	baseName  string
	totalLen  int64

	incompleteFile *trunkFile
	completeFile   *os.File

	em *utils.EventManager[*Fragment]

	Complete atomic.Bool

	activeFragment      *Fragment
	downloadingFragment *Fragment

	fragmentsMutex  sync.RWMutex
	activeFragMutex sync.RWMutex
	fileMutex       sync.RWMutex
}

func NewFile(baseName string) *File {
	f := &File{
		baseName: baseName,

		em: utils.NewEventManager[*Fragment](),

		Complete: atomic.Bool{},

		fragmentsMutex:  sync.RWMutex{},
		activeFragMutex: sync.RWMutex{},
	}
	f.LoadFragments()
	return f
}

func (f *File) Close() error {
	f.fileMutex.RLock()
	defer f.fileMutex.RUnlock()

	if f.completeFile != nil {
		return f.completeFile.Close()
	}
	if f.incompleteFile != nil {
		return f.incompleteFile.Close()
	}
	return nil
}
func (f *File) GetDownloadedBytes() int64 {
	f.fragmentsMutex.RLock()
	defer f.fragmentsMutex.RUnlock()

	return lo.SumBy(f.fragments, func(frag *Fragment) int64 {
		return frag.length
	})
}

func (f *File) GetTotalLength(getter func() int64) int64 {
	if f.totalLen == 0 {
		f.totalLen = getter()
		// TODO
		if f.incompleteFile != nil {
			f.incompleteFile.Init(f.totalLen, time.Unix(0, 0))
		}
	}
	return f.totalLen
}

func (f *File) RequestRsc() io.ReadSeekCloser {
	if f.totalLen == 0 {
		return nil
	}
	return rw_file.NewRsc(f, f.totalLen)
}

func (f *File) IsComplete() bool {
	return f.Complete.Load()
}
