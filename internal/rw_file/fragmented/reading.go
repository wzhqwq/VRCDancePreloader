package fragmented

import (
	"context"
	"io"
	"log"
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
			log.Println("Timeout warning: requested file", f.File.Name(), "offset", offset, "length", length)
			log.Println("It takes 10 seconds and still not available, we should close this request to prevent memory leak")
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
