package entry

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/legacy_file"
)

type UrlBasedEntry struct {
	BaseEntry

	resolvedUrl string
	resolver    UrlResolver

	checkingMutex sync.Mutex

	remoteModTime time.Time
	remoteSize    int64
	remoteEtag    string
}

func NewUrlBasedEntry(id string, resolver UrlResolver) *UrlBasedEntry {
	return &UrlBasedEntry{
		BaseEntry: ConstructBaseEntry(id, requesting.GetClient(requesting.NoProxy)),
		resolver:  resolver,
	}
}

func (e *UrlBasedEntry) resolveRemoteMedia(ctx context.Context) error {
	resolver := e.resolver
	var info *RemoteVideoInfo
	var err error
	var lastUrl string

	for {
		if resolver == nil {
			if err != nil {
				return err
			}
			return ErrNotSupported
		}

		info, err = resolver.Resolve(e.logger, ctx)
		if err != nil {
			resolver = resolver.Next(lastUrl)
			continue
		}
		lastUrl = info.FinalUrl

		// check if it's a real video, at least 1MB
		if info.TotalSize > 1024*1024 {
			break
		} else {
			resolver = resolver.Next(info.FinalUrl)
		}
	}

	if e.remoteModTime.IsZero() {
		if info.LastModified.IsZero() && info.Etag == "" {
			e.logger.WarnLn("We cannot get the modified time or etag of this file on the server, so it's not possible to check if the cache is expired")
		}
		e.remoteModTime = info.LastModified
	}

	e.remoteSize = info.TotalSize
	e.resolvedUrl = info.FinalUrl
	e.remoteEtag = info.Etag
	if info.Client != nil {
		e.client = info.Client
	}
	e.logger.InfoLn(e.id, "resolved to", info.FinalUrl, "size:", e.remoteSize, "modified time:", e.remoteModTime.Local().String(), "etag:", e.remoteEtag)

	return nil
}

func (e *UrlBasedEntry) checkWorkingFile(ctx context.Context) error {
	e.checkingMutex.Lock()
	defer e.checkingMutex.Unlock()

	if e.fileUpgrading.Load() {
		return ErrUpgrading
	}

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
		if errors.Is(err, legacy_file.ErrLegacyDeprecated) {
			e.logger.WarnLn("Legacy file format detected, we will re-download it completely")
			e.Upgrade()
			return ErrUpgrading
		}
		return err
	}
	if e.remoteEtag != "" {
		e.setEtag(e.remoteEtag)
	}

	return e.updateMeta()
}

func (e *UrlBasedEntry) ModTime() time.Time {
	e.acquireFileRLock()
	defer e.releaseFileRLock()

	if err := e.checkWorkingFile(context.Background()); err != nil {
		return unixEpochTime
	}

	return e.workingFile.ModTime()
}

func (e *UrlBasedEntry) TotalLen() (int64, error) {
	e.acquireFileRLock()
	defer e.releaseFileRLock()

	if err := e.checkWorkingFile(context.Background()); err != nil {
		return 0, err
	}

	return e.workingFile.TotalLen(), nil
}
func (e *UrlBasedEntry) GetDownloadStream(ctx context.Context) (io.ReadCloser, error) {
	e.acquireFileRLock()
	defer e.releaseFileRLock()

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
	e.acquireFileRLock()
	defer e.releaseFileRLock()

	if err := e.checkWorkingFile(ctx); err != nil {
		return nil, err
	}

	return e.getReadSeeker(ctx)
}

func (e *UrlBasedEntry) IsComplete() bool {
	e.acquireFileRLock()
	defer e.releaseFileRLock()

	if err := e.checkWorkingFile(context.Background()); err != nil {
		return false
	}
	return e.checkIfCompleteAndSync()
}
