package entry

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/legacy_file"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var ErrNotSupported = errors.New("video is not currently supported")

type CDNEntry interface {
	types.CDNResource
	types.CDNFile

	EnsureOpen()
	Close() error
}

type BaseCDNEntry struct {
	id string

	workingFileMutex sync.RWMutex

	workingFile types.DeferredReadableFile

	logger utils.LoggerImpl

	meta *persistence.CacheMeta

	// custom
	openFileFn func() types.DeferredReadableFile

	// optional
	etag *Etag

	upgradeFn func()
}

func (e *BaseCDNEntry) Logger() utils.LoggerImpl {
	return e.logger
}

// extendable operations (please wrap with mutex by yourself and check workingFile first!!)

func (e *BaseCDNEntry) closeFile() error {
	err := e.workingFile.Close()
	if err == nil || errors.Is(err, os.ErrClosed) {
		e.workingFile = nil
		return nil
	}

	return err
}

func (e *BaseCDNEntry) Init(size int64, lastModified time.Time, etag string) error {
	e.workingFileMutex.Lock()
	defer e.workingFileMutex.Unlock()

	for {
		err := e.workingFile.Init(size, lastModified)
		if err != nil {
			if errors.Is(err, legacy_file.ErrLegacyDeprecated) {
				e.logger.WarnLn("Legacy file format detected, we will re-download it completely")

				if e.upgradeFn != nil {
					e.upgradeFn()
					continue
				}

				return errors.New("legacy file upgrade failed")
			}
			return err
		}
		break
	}

	if etag != "" && e.etag != nil {
		e.etag.Set(etag)
	}

	return e.updateMeta()
}

func (e *BaseCDNEntry) ReconcileRemoteInfo(info *types.RemoteHttpResourceInfo) {
	initNeeded := false

	e.workingFileMutex.RLock()
	if e.workingFile == nil {
		return
	}
	defer func() {
		e.workingFileMutex.RUnlock()
		if initNeeded {
			err := e.Init(info.TotalSize, info.LastModified, info.Etag)
			if err != nil {
				e.logger.ErrorLn("Failed to initialize cache", e.id, "url:", info.FinalUrl)
			}
		}
	}()

	if e.workingFile.TotalLen() == 0 {
		initNeeded = true
		return
	}
	if info.Etag != "" && e.Etag() == info.Etag {
		// not changed
		return
	}
	if info.Etag != "" || (!info.LastModified.IsZero() && info.LastModified.After(e.workingFile.ModTime())) {
		if e.workingFile.GetDownloadedBytes() > 0 {
			// local cache is expired
			e.logger.WarnLn("Local cache expired so we will re-download it completely")
		}
		initNeeded = true
		return
	}

	if info.LastModified.IsZero() && info.Etag == "" {
		e.logger.WarnLn("We cannot get the modified time or etag of this file on the server, so it's not possible to check if the cache is expired")
	}
}

func (e *BaseCDNEntry) AcquireFile() (types.DeferredReadableFile, error) {
	e.workingFileMutex.RLock()

	if e.workingFile == nil {
		e.workingFileMutex.RUnlock()
		return nil, io.ErrClosedPipe
	}

	return e.workingFile, nil
}

func (e *BaseCDNEntry) ReleaseFile() {
	e.workingFileMutex.RUnlock()
}

// adapters

func (e *BaseCDNEntry) EnsureOpen() {
	e.workingFileMutex.Lock()
	defer e.workingFileMutex.Unlock()

	if e.workingFile != nil {
		return
	}

	e.workingFile = e.openFileFn()
	e.syncWithFS()
}

func (e *BaseCDNEntry) Close() error {
	e.workingFileMutex.Lock()
	defer e.workingFileMutex.Unlock()

	if e.workingFile == nil {
		return nil
	}

	err := e.closeFile()

	// From this point on, the file object must not be reused, even if the
	// underlying Close operation reported a flush or filesystem error.
	e.workingFile = nil

	return err
}

func (e *BaseCDNEntry) Etag() string {
	if e.etag != nil {
		return e.etag.Read()
	}
	return ""
}

func (e *BaseCDNEntry) MarkComplete() {
	e.meta.SetPartial(false)
}

func (e *BaseCDNEntry) UpdateReqRangeStart(start int64) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.workingFile != nil {
		e.workingFile.NotifyRequestStart(start)
	}
}

func (e *BaseCDNEntry) GetResource(ctx context.Context) (io.ReadSeeker, int64, time.Time, error) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.workingFile == nil {
		return nil, 0, time.Time{}, io.ErrClosedPipe
	}

	r := e.workingFile.RequestRs(ctx)
	l := e.workingFile.TotalLen()
	t := e.workingFile.ModTime()

	if r == nil || l == 0 {
		return nil, 0, time.Time{}, errors.New("failed to download this video")
	}

	return r, l, t, nil
}

var _ types.CDNFile = &BaseCDNEntry{}
