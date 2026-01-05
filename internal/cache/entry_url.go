package cache

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type UrlBasedEntry struct {
	BaseEntry

	resolvedUrl       string
	initialInfoGetter func(ctx context.Context) (*RemoteVideoInfo, error)

	remoteModTime time.Time
}

func newUrlBasedEntry(id string, client *http.Client, initialInfoGetter func(ctx context.Context) (*RemoteVideoInfo, error)) *UrlBasedEntry {
	return &UrlBasedEntry{
		BaseEntry:         ConstructBaseEntry(id, client),
		initialInfoGetter: initialInfoGetter,
	}
}

func (e *UrlBasedEntry) resolveRemoteMedia(ctx context.Context) error {
	info, err := e.initialInfoGetter(ctx)
	if err != nil {
		return err
	}

	url := info.FinalUrl
	if !info.LastModified.IsZero() {
		// BiliBili provide creation time through API
		e.remoteModTime = info.LastModified
	}

	for {
		if _, ok := utils.CheckYoutubeURL(url); ok {
			// TODO youtube handler
			return ErrNotSupported
		}
		info, err = e.requestHttpResInfo(url, ctx)
		if err != nil {
			return err
		}
		changed := info.FinalUrl != url
		url = info.FinalUrl

		// check if it's a real video, at least 1MB
		if info.TotalSize > 1024*1024 {
			break
		} else if !changed {
			// not redirected and not a video
			return ErrNotSupported
		}
	}

	if e.remoteModTime.IsZero() {
		e.remoteModTime = info.LastModified
	}

	e.workingFile.UpdateRemoteInfo(info.TotalSize, e.remoteModTime)

	e.resolvedUrl = url
	e.logger.InfoLn(e.id, "resolved to", url)
	return nil
}

func (e *UrlBasedEntry) checkWorkingFile(ctx context.Context) error {
	if e.workingFile == nil {
		return io.ErrClosedPipe
	}

	if e.workingFile.IsComplete() && !forceExpirationCheck {
		// skip check unless we need to check Last-Modified
		return nil
	}

	localModTime := e.workingFile.ModTime()

	if e.remoteModTime.IsZero() || e.resolvedUrl == "" {
		// make sure that we have recorded Last-Modified and url
		err := e.resolveRemoteMedia(ctx)
		if err != nil {
			return err
		}
	}

	if !localModTime.IsZero() && !e.remoteModTime.IsZero() && e.remoteModTime.After(localModTime) {
		// local cache is expired
		e.logger.WarnLn("Local cache expired so we will re-download it completely")
		err := e.workingFile.Clear()
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *UrlBasedEntry) ModTime() time.Time {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if err := e.checkWorkingFile(context.Background()); err != nil {
		return unixEpochTime
	}

	return e.workingFile.ModTime()
}

func (e *UrlBasedEntry) TotalLen() (int64, error) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if err := e.checkWorkingFile(context.Background()); err != nil {
		return 0, err
	}

	return e.workingFile.TotalLen(), nil
}
func (e *UrlBasedEntry) GetDownloadStream(ctx context.Context) (io.ReadCloser, error) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if err := e.checkWorkingFile(ctx); err != nil {
		return nil, err
	}

	e.workingFile.MarkDownloading()
	offset := e.workingFile.GetDownloadOffset()

	e.logger.InfoLnf("Download %s start from %d, (total %d)", e.id, offset, e.workingFile.TotalLen())

	return e.requestHttpResBody(e.resolvedUrl, offset, ctx)
}
func (e *UrlBasedEntry) Reset() {
	e.resolvedUrl = ""
}
func (e *UrlBasedEntry) GetReadSeeker(ctx context.Context) (io.ReadSeeker, error) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if err := e.checkWorkingFile(ctx); err != nil {
		return nil, err
	}

	return e.getReadSeeker(ctx)
}
