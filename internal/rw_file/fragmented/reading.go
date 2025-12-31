package fragmented

import (
	"context"
	"io"
	"time"
)

func (f *File) checkRange(offset, length int64) bool {
	f.fragmentsMutex.RLock()
	defer f.fragmentsMutex.RUnlock()

	for _, frag := range f.fragments {
		if frag.Includes(offset, length) {
			return true
		}
	}
	return false
}

func (f *File) RequestRange(offset, length int64, ctx context.Context) error {
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
			if frag.Includes(offset, length) {
				return nil
			}
		case <-ctx.Done():
			return io.ErrClosedPipe
		case <-time.After(time.Second * 10):
			logger.WarnLn("Timeout warning: requested file", f.File.Name(), "range offset", offset, "length", length)
			logger.InfoLn("It takes 10 seconds and still not available, so this request is forced to close")
			f.printFragments()
			return io.ErrNoProgress
		}
	}
}

func (f *File) ReadAt(p []byte, offset int64) (int, error) {
	if f.IsComplete() {
		return f.File.ReadAt(p, offset)
	}

	f.fragmentsMutex.RLock()
	defer f.fragmentsMutex.RUnlock()

	for _, frag := range f.fragments {
		if frag.Contains(offset) {
			readLen := min(len(p), int(frag.End()-offset))
			return f.File.ReadAt(p[:readLen], offset)
		}
	}

	return 0, io.EOF
}
