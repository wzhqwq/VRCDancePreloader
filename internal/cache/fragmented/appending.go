package fragmented

import (
	"errors"
	"github.com/samber/lo"
	"io"
	"log"
)

// only be called by Append
func (f *File) checkAppend(bytes []byte) (int, error) {
	f.fragmentsMutex.RLock()
	defer f.fragmentsMutex.RUnlock()

	cursor := f.downloadingFragment.End()
	if cursor >= f.totalLen {
		// It can't be
		log.Printf("file %s is oversized, may be we have damanged the video file TAT", f.baseName)
		return 0, io.EOF
	}

	downloadingIndex := lo.IndexOf(f.fragments, f.downloadingFragment)
	if downloadingIndex == -1 {
		return 0, errors.New("fragment lost")
	}

	// trim if needed
	if downloadingIndex+1 < len(f.fragments) {
		nextFrag := f.fragments[downloadingIndex+1]
		if cursor >= nextFrag.start {
			// current fragment is done, seek new position
			return 0, io.EOF
		}
		return min(len(bytes), int(nextFrag.start-cursor)), nil
	} else {
		if len(bytes) > int(f.totalLen-cursor) {
			// It can't be
			log.Printf("file %s is oversized, may be we have damanged the video file TAT", f.baseName)
			return int(f.totalLen - cursor), nil
		}
		return len(bytes), nil
	}
}

func (f *File) Append(bytes []byte) (int, error) {
	if f.downloadingFragment != f.activeFragment {
		// force re-downloading active fragment
		return 0, io.EOF
	}

	n, err := f.checkAppend(bytes)
	if err != nil {
		if errors.Is(err, io.EOF) {
			f.mergeInLoop(f.downloadingFragment)
		}
		return 0, err
	}

	err = f.incompleteFile.AppendTo(f.downloadingFragment, bytes[:n])
	if err != nil {
		return 0, err
	}

	if n < len(bytes) {
		// This means current fragment is finished if this n bytes are written
		// We can also merge current fragment
		f.mergeInLoop(f.downloadingFragment)
		// and return EOF according to the document of go
		// which also force the downloader to restart with new offset
		return n, io.EOF
	}
	if f.downloadingFragment.End() == f.totalLen {
		// We have reached the end, back to the first fragment if there's no other request
		f.backToFirst()
		return n, io.EOF
	}

	f.em.NotifySubscribers(f.downloadingFragment)

	//f.fragmentsMutex.RLock()
	//f.printFragments()
	//f.fragmentsMutex.RUnlock()

	return n, nil
}
