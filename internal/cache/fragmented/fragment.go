package fragmented

import (
	"github.com/samber/lo"
	"log"
	"os"
	"slices"
)

type Fragment struct {
	start  int64
	length int64
}

func newFragment(start int64, length int64) *Fragment {
	return &Fragment{
		start:  start,
		length: length,
	}
}

func (f *Fragment) End() int64 {
	return f.start + f.length
}

func (f *File) LoadFragments() {
	if stat, err := os.Stat(f.baseName); err == nil {
		file, err := os.OpenFile(f.baseName, os.O_RDONLY, 0666)
		if err != nil {
			log.Printf("open file %s failed: %v\n", f.baseName, err)
		} else {
			f.completeFile = file
			f.Complete.Swap(true)
			f.totalLen = stat.Size()
			return
		}
	}

	f.incompleteFile = newTrunkFile(f.baseName)
	f.fragments = f.incompleteFile.ToFragments()
	f.backToFirst()
}

func (f *File) setActive(frag *Fragment) {
	f.mergeInLoop(frag)
	if f.totalLen > 0 && frag.End() >= f.totalLen {
		frag = f.fragments[0]
	}
	f.activeFragMutex.Lock()
	f.activeFragment = frag
	f.activeFragMutex.Unlock()
}

func (f *File) backToFirst() {
	f.mergeForward(f.fragments[0])
	f.checkComplete()

	f.activeFragMutex.Lock()
	if f.activeFragment == f.downloadingFragment {
		// to prevent overriding unprocessed request
		f.activeFragment = f.fragments[0]
	}
	f.activeFragMutex.Unlock()
}

func (f *File) printFragments() {
	for _, frag := range f.fragments {
		notation := ""
		if frag == f.activeFragment {
			notation = " <-"
		}
		log.Printf("%d - %d %s", frag.start, frag.start+frag.length, notation)
	}
}

func (f *File) addFragment(start int64) *Fragment {
	newFrag := newFragment(start, 0)

	f.fragmentsMutex.Lock()
	f.fragments = append(f.fragments, newFrag)
	slices.SortFunc(f.fragments, func(a, b *Fragment) int {
		return int(a.start - b.start)
	})
	//f.printFragments()
	f.fragmentsMutex.Unlock()

	return newFrag
}

func (f *File) mergeForward(currentFrag *Fragment) {
	f.fragmentsMutex.Lock()
	defer func() {
		f.incompleteFile.FillTrunks(currentFrag)
		f.em.NotifySubscribers(currentFrag)
		//f.printFragments()
		f.fragmentsMutex.Unlock()
	}()

	index := lo.IndexOf(f.fragments, currentFrag)
	if index == -1 {
		return
	}

	if index+1 < len(f.fragments) {
		prefix := f.fragments[:index+1]
		merging := f.fragments[index+1:]

		for i, frag := range merging {
			if currentFrag.End() < frag.start {
				f.fragments = append(prefix, merging[i:]...)
				return
			}
			currentFrag.length = frag.End() - currentFrag.start
		}

		f.fragments = append(prefix)
	}
}

func (f *File) mergeInLoop(frag *Fragment) {
	f.mergeForward(frag)
	if f.totalLen == 0 || frag.End() < f.totalLen {
		return
	}
	// suffix downloaded, start from prefix
	f.mergeForward(f.fragments[0])
	f.checkComplete()
}

func (f *File) checkComplete() {
	if f.totalLen == 0 || len(f.fragments) > 1 || f.fragments[0].End() < f.totalLen {
		return
	}

	if !f.Complete.CompareAndSwap(false, true) {
		return
	}

	go func() {
		f.fileMutex.Lock()
		defer f.fileMutex.Unlock()

		err := f.incompleteFile.SaveAs(f.baseName)
		if err != nil {
			log.Printf("save file %s failed: %v\n", f.baseName, err)
			return
		}

		file, err := os.OpenFile(f.baseName, os.O_RDONLY, 0666)
		if err != nil {
			log.Printf("open file %s failed: %v\n", f.baseName, err)
		} else {
			f.completeFile = file
			f.incompleteFile.ScheduleRemove()
			f.incompleteFile = nil
		}
	}()
}
