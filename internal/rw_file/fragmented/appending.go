package fragmented

import (
	"errors"

	"github.com/samber/lo"
)

var ErrEndOfFragment = errors.New("end of fragment")
var ErrFragmentLost = errors.New("fatal: fragment lost")
var ErrBadFragment = errors.New("fatal: bad fragment")

// only be called by Append
func (f *File) checkAppend(bytes []byte) (int, error) {
	f.fragmentsMutex.RLock()
	defer f.fragmentsMutex.RUnlock()

	fullSize := f.File.FullSize

	cursor := f.downloadingFragment.End()
	if cursor >= fullSize {
		// It can't be
		logger.ErrorLnf("file %s is oversized, may be we have damaged the video file TAT", f.File.Name())
		return 0, ErrBadFragment
	}

	downloadingIndex := lo.IndexOf(f.fragments, f.downloadingFragment)
	if downloadingIndex == -1 {
		return 0, ErrFragmentLost
	}

	// trim if needed
	if downloadingIndex+1 < len(f.fragments) {
		nextFrag := f.fragments[downloadingIndex+1]
		if cursor >= nextFrag.Start {
			// current fragment is done, seek new position
			return 0, ErrEndOfFragment
		}
		return min(len(bytes), int(nextFrag.Start-cursor)), nil
	}

	if len(bytes) > int(fullSize-cursor) {
		// It can't be
		logger.ErrorLnf("file %s is oversized, may be we have damaged the video file TAT", f.File.Name())
		return int(fullSize - cursor), nil
	}
	return len(bytes), nil
}

func (f *File) Append(bytes []byte) (int, error) {
	if f.downloadingFragment != f.activeFragment {
		// force re-downloading active fragment
		return 0, ErrEndOfFragment
	}

	n, err := f.checkAppend(bytes)
	if err != nil {
		if errors.Is(err, ErrEndOfFragment) {
			f.mergeInLoop(f.downloadingFragment)
		}
		return 0, err
	}

	err = f.File.AppendTo(f.downloadingFragment, bytes[:n])
	if err != nil {
		return 0, err
	}

	if n < len(bytes) {
		// This means current fragment is finished if this n bytes are written
		// We can also merge current fragment
		f.mergeInLoop(f.downloadingFragment)
		// and force the downloader to restart with new offset
		return n, ErrEndOfFragment
	}
	if f.downloadingFragment.End() == f.File.FullSize {
		// We have reached the end, back to the first fragment if there's no other request
		f.backToFirst()
		return n, ErrEndOfFragment
	}

	f.em.NotifySubscribers(f.downloadingFragment)

	//f.fragmentsMutex.RLock()
	//f.printFragments()
	//f.fragmentsMutex.RUnlock()

	return n, nil
}
