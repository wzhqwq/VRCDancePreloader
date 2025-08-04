package fragmented

func (f *File) findAvailableFragment(start int64) *Fragment {
	f.fragmentsMutex.RLock()
	defer f.fragmentsMutex.RUnlock()
	f.activeFragMutex.RLock()
	defer f.activeFragMutex.RUnlock()

	for _, frag := range f.fragments {
		if frag.start <= start {
			offset := start - frag.start

			if f.activeFragment == frag {
				if offset < frag.length+continueThreshold {
					// continue the current fragment
					return f.activeFragment
				}
			} else {
				if offset < frag.length {
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
	if f.Complete.Load() {
		return
	}
	f.activeOrCreateFragment(start)
}

func (f *File) GetDownloadOffset() int64 {
	return f.downloadingFragment.start + f.downloadingFragment.length
}

func (f *File) MarkDownloading() {
	// It never races with Append, so we don't need to protect downloadingFragment
	f.downloadingFragment = f.activeFragment
}
