package fragmented

import (
	"context"
	"io"
	"sync"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/trunk"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// continue the current fragment if the starting point is not more than 100KB beyond the fragment
const continueThreshold = int64(1024 * 100)

var logger = utils.NewLogger("Fragmented File")

type File struct {
	rw_file.BaseFile

	fragments []*trunk.Fragment

	em *utils.EventManager[*trunk.Fragment]

	activeFragment      *trunk.Fragment
	downloadingFragment *trunk.Fragment

	fragmentsMutex  sync.RWMutex
	activeFragMutex sync.RWMutex

	lastestRequestStart int64
}

func (f *File) Clear() error {
	// file won't be cleared if the file is active
	// so we can safely reset fragments
	f.File.ClearTrunks()
	f.fragments = f.File.ToFragments()
	f.activeFragment = f.fragments[0]
	return nil
}

func NewFile(baseName string) *File {
	f := &File{
		BaseFile: rw_file.ConstructBaseFile(baseName),

		em: utils.NewEventManager[*trunk.Fragment](),
	}

	if f.File == nil {
		return nil
	}

	f.LoadFragments()
	return f
}

func (f *File) RequestRs(ctx context.Context) io.ReadSeeker {
	if f.File.FullSize == 0 {
		return nil
	}
	return rw_file.NewRSWithContext(f, f.File.FullSize, ctx)
}

func (f *File) GetDownloadedBytes() int64 {
	f.fragmentsMutex.RLock()
	defer f.fragmentsMutex.RUnlock()

	return lo.SumBy(f.fragments, func(frag *trunk.Fragment) int64 {
		return frag.Length
	})
}
