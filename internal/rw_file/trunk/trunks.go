package trunk

func trunkRangeToFragment(start, length int) *Fragment {
	return &Fragment{
		Start:  int64(start) * bytesPerTrunk,
		Length: int64(length) * bytesPerTrunk,
	}
}

func (f *File) ToFragments() []*Fragment {
	fragments := make([]*Fragment, 0, len(f.trunks))
	startIndex := -1
	for i, b := range f.trunks {
		if b == 0 {
			if startIndex != -1 {
				fragments = append(fragments, trunkRangeToFragment(startIndex, i-startIndex))
				startIndex = -1
			}
			continue
		} else {
			if startIndex == -1 {
				startIndex = i
			}
		}
	}
	if startIndex != -1 {
		fragments = append(fragments, trunkRangeToFragment(startIndex, len(f.trunks)-startIndex))
	}
	if len(fragments) == 0 {
		return []*Fragment{
			NewFragment(0, 0),
		}
	}
	return fragments
}

// trunkIndexRange is the half open bitmap range a fragment covers, clamped to
// the bitmap.
//
// The clamp is what keeps a write past capacity from indexing out of range. It
// is a separate function so the clamp can be tested without a file on disk, and
// so FillTrunks stays a plain loop. Init rejects oversized files up front, so in
// practice the clamp is the second line of defense, not the first.
func (f *File) trunkIndexRange(frag *Fragment) (start, end int64) {
	start = (frag.Start + bytesPerTrunk - 1) / bytesPerTrunk
	end = frag.End() / bytesPerTrunk

	if end > int64(len(f.trunks)) {
		end = int64(len(f.trunks))
	}

	return start, end
}

func (f *File) FillTrunks(frag *Fragment) {
	fillStart, fillEnd := f.trunkIndexRange(frag)
	trunksChanged := false

	for i := fillStart; i < fillEnd; i++ {
		if f.trunks[i] == 0 {
			trunksChanged = true
			f.trunks[i] = 1
		}
	}

	if trunksChanged {
		f.writeTrunks()
	}
}
