package fragmented

import (
	"slices"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/trunk"
)

func (f *File) LoadFragments() {
	f.fragments = f.File.ToFragments()
	f.backToFirst()
}

func (f *File) setActive(frag *trunk.Fragment) {
	f.mergeInLoop(frag)
	if f.File.IsSuffix(frag) {
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
	logger.DebugLnf("Fragments of %s (total %d):", f.File.Name(), f.TotalLen())
	for _, frag := range f.fragments {
		notation := ""
		if frag == f.activeFragment {
			notation = " *"
		}
		logger.DebugLnf("%d - %d%s", frag.Start, frag.End(), notation)
	}
}

func (f *File) addFragment(start int64) *trunk.Fragment {
	newFrag := trunk.NewFragment(start, 0)

	f.fragmentsMutex.Lock()
	f.fragments = append(f.fragments, newFrag)
	slices.SortFunc(f.fragments, func(a, b *trunk.Fragment) int {
		return int(a.Start - b.Start)
	})
	//f.printFragments()
	f.fragmentsMutex.Unlock()

	return newFrag
}

func (f *File) mergeForward(currentFrag *trunk.Fragment) {
	f.fragmentsMutex.Lock()
	defer func() {
		f.File.FillTrunks(currentFrag)
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
			if currentFrag.End() < frag.Start {
				f.fragments = append(prefix, merging[i:]...)
				return
			}
			currentFrag.Length = frag.End() - currentFrag.Start
		}

		f.fragments = append(prefix)
	}
}

func (f *File) mergeInLoop(frag *trunk.Fragment) {
	f.mergeForward(frag)
	if f.File.IsSuffix(frag) {
		return
	}
	// suffix downloaded, start from prefix
	f.mergeForward(f.fragments[0])
	f.checkComplete()
}

func (f *File) checkComplete() {
	if f.File.Completed {
		return
	}
	if len(f.fragments) == 1 && f.File.IsSuffix(f.fragments[0]) {
		f.File.MarkCompleted()
	}
}
