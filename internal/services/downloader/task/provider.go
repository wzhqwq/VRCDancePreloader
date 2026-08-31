package task

import (
	"context"
	"io"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type StreamInfo struct {
	Rc     io.ReadCloser
	Length int64

	RangeAvailable bool
}

type RemoteProvider interface {
	WaitResolving(ctx context.Context, beforeWait func(status interactive.RemoteStatus)) (int64, error)
	GetDownloadStream(offset int64, ctx context.Context) (StreamInfo, error)
}

type LocalProvider interface {
	io.Writer

	Open() error
	Close()

	IsComplete() bool
	IsForceResolving() bool
	DownloadedSize() int64

	CurrentCursor() (int64, error)
	SeekStart() error
}
