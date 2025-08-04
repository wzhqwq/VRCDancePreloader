package fragmented

import (
	"io"
)

func (f *File) checkRange(offset, length int64) bool {
	f.fragmentsMutex.RLock()
	defer f.fragmentsMutex.RUnlock()

	for _, frag := range f.fragments {
		if frag.start <= offset && offset+length < frag.start+frag.length {
			return true
		}
	}
	return false
}

func (f *File) RequestRange(offset, length int64, closeCh chan struct{}) error {
	if f.IsComplete() {
		return nil
	}

	ch := f.em.SubscribeEvent()
	defer ch.Close()

	if f.checkRange(offset, length) {
		return nil
	}

	for {
		select {
		case frag := <-ch.Channel:
			if frag.start <= offset && offset+length < frag.start+frag.length {
				return nil
			}
		case <-closeCh:
			return io.ErrClosedPipe
		}
	}
}

func (f *File) ReadAt(p []byte, offset int64) (int, error) {
	f.fileMutex.RLock()
	defer f.fileMutex.RUnlock()

	if f.completeFile != nil {
		return f.completeFile.ReadAt(p, offset)
	}
	if f.Complete.Load() {
		return f.incompleteFile.ReadAt(p, offset)
	}

	f.fragmentsMutex.RLock()
	defer f.fragmentsMutex.RUnlock()

	for _, frag := range f.fragments {
		if frag.start <= offset && offset < frag.start+frag.length {
			readLen := min(len(p), int(frag.start+frag.length-offset))
			return f.incompleteFile.ReadAt(p[:readLen], offset)
		}
	}

	return 0, io.EOF
}
