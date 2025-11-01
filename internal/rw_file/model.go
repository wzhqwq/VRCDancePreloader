package rw_file

import (
	"context"
	"io"
	"time"
)

type DeferredReadableFile interface {
	Init(contentLength int64, lastModified time.Time)
	TotalLen() int64
	ModTime() time.Time

	Append(bytes []byte) (int, error)
	ReadAt(p []byte, off int64) (int, error)
	Close() error

	RequestRs(ctx context.Context) io.ReadSeeker

	NotifyRequestStart(start int64)
	MarkDownloading()

	GetDownloadOffset() int64
	GetDownloadedBytes() int64
	IsComplete() bool
}

type DeferredReader interface {
	RequestRange(offset, length int64, ctx context.Context) error
	ReadAt(p []byte, off int64) (int, error)
}
