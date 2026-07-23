package types

import (
	"context"
	"io"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type CDNFile interface {
	AcquireFile() (DeferredReadableFile, error)
	ReleaseFile()

	Etag() string
	ReconcileRemoteInfo(info *RemoteHttpResourceInfo)

	MarkComplete()

	Logger() utils.LoggerImpl
}

type CDNResource interface {
	GetResource(ctx context.Context) (io.ReadSeeker, int64, time.Time, error)
	UpdateReqRangeStart(start int64)
}

type CDNFileSession interface {
	CDNFile
	CDNResource

	Open(id string, loggers ...utils.LoggerImpl) error
	Close(loggers ...utils.LoggerImpl)

	IsForceExpiration() bool
}
