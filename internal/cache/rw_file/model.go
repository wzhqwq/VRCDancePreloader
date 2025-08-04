package rw_file

import "io"

type DeferredReadableFile interface {
	Append(bytes []byte) (int, error)
	ReadAt(p []byte, off int64) (int, error)
	Close() error

	RequestRsc() io.ReadSeekCloser

	NotifyRequestStart(start int64)
	MarkDownloading()

	GetDownloadOffset() int64
	GetDownloadedBytes() int64
	GetTotalLength(getter func() int64) int64
	IsComplete() bool
}

type DeferredReader interface {
	RequestRange(offset, length int64, closeCh chan struct{}) error
	ReadAt(p []byte, off int64) (int, error)
}
