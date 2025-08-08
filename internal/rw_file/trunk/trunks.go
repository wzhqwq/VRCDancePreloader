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

func (f *File) FillTrunks(frag *Fragment) {
	fillStart := (frag.Start + bytesPerTrunk - 1) / bytesPerTrunk
	fillEnd := frag.End() / bytesPerTrunk
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
