package legacy_file

import (
	"context"
	"io"
)

func (f *File) RequestRange(offset, length int64, ctx context.Context) error {
	if f.IsComplete() {
		return nil
	}

	ch := f.em.SubscribeEvent()
	defer ch.Close()

	for {
		select {
		case l := <-ch.Channel:
			if l >= offset+length {
				return nil
			}
		case <-ctx.Done():
			return io.ErrClosedPipe
		}
	}
}

func (f *File) ReadAt(p []byte, off int64) (n int, err error) {
	f.fileMutex.RLock()
	defer f.fileMutex.RUnlock()

	return f.file.ReadAt(p, off)
}
