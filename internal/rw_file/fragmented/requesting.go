package fragmented

import "github.com/wzhqwq/VRCDancePreloader/internal/rw_file/trunk"

func (f *File) findAvailableFragment(start int64) *trunk.Fragment {
	f.fragmentsMutex.RLock()
	defer f.fragmentsMutex.RUnlock()
	f.activeFragMutex.RLock()
	defer f.activeFragMutex.RUnlock()

	for _, frag := range f.fragments {
		if frag.Start <= start {
			offset := start - frag.Start

			if f.activeFragment == frag {
				if offset < frag.Length+continueThreshold {
					// continue the current fragment
					return f.activeFragment
				}
			} else {
				if offset < frag.Length {
					// the starting point is in another fragment
					return frag
				}
			}
		}
	}

	return nil
}

func (f *File) activeOrCreateFragment(start int64) {
	active := f.findAvailableFragment(start)
	if active == nil {
		// add a new fragment and mark as active
		active = f.addFragment(start)
	}
	f.setActive(active)
}

func (f *File) NotifyRequestStart(start int64) {
	if f.File.Completed {
		return
	}
	f.activeOrCreateFragment(start)
}

func (f *File) GetDownloadOffset() int64 {
	return f.downloadingFragment.End()
}

func (f *File) MarkDownloading() {
	// It never races with Append, so we don't need to protect downloadingFragment
	f.downloadingFragment = f.activeFragment
}
