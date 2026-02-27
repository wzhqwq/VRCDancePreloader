package entry

import (
	"context"
	"io"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type UrlBasedEntry struct {
	BaseEntry

	resolvedUrl       string
	initialInfoGetter func(ctx context.Context) (*RemoteVideoInfo, error)

	remoteModTime time.Time
	remoteSize    int64
	remoteEtag    string
}

func NewUrlBasedEntry(id string, client *requesting.ClientProvider, initialInfoGetter func(ctx context.Context) (*RemoteVideoInfo, error)) *UrlBasedEntry {
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
	e.referer = ""
	if !info.LastModified.IsZero() {
		// BiliBili provide creation time through API
		// DuDuFitDance CDN does not response with Last-Modified, but provide "publishTime" in manifest
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
		if info.LastModified.IsZero() {
			e.logger.WarnLn("We cannot get the modified time of this file on the server, so it's not possible to check if the cache is expired")
		}
		e.remoteModTime = info.LastModified
	}

	e.remoteSize = info.TotalSize
	e.resolvedUrl = url
	e.remoteEtag = info.Etag
	e.logger.InfoLn(e.id, "resolved to", url, "size:", e.remoteSize, "modified time:", e.remoteModTime.Local().String(), "etag:", e.remoteEtag)

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

	if e.resolvedUrl == "" {
		// make sure that we have recorded Last-Modified and url
		err := e.resolveRemoteMedia(ctx)
		if err != nil {
			return err
		}
	}

	if e.workingFile.TotalLen() == 0 {
		return e.init()
	}
	if e.remoteEtag != "" && e.etag == e.remoteEtag {
		// not changed
		return nil
	}
	if e.remoteEtag != "" || (!e.remoteModTime.IsZero() && e.remoteModTime.After(localModTime)) {
		if e.workingFile.GetDownloadedBytes() > 0 {
			// local cache is expired
			e.logger.WarnLn("Local cache expired so we will re-download it completely")
		}
		return e.init()
	}

	return nil
}

func (e *UrlBasedEntry) init() error {
	err := e.workingFile.Init(e.remoteSize, e.remoteModTime)
	if err != nil {
		return err
	}
	if e.remoteEtag != "" {
		e.setEtag(e.remoteEtag)
	}

	size, created := e.workingFile.Stat()
	return e.meta.UpdateInfo(size, e.remoteModTime, created)
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

func (e *UrlBasedEntry) IsComplete() bool {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if err := e.checkWorkingFile(context.Background()); err != nil {
		return false
	}
	return e.checkIfCompleteAndSync()
}
