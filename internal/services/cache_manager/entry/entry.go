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

	// initialized is closed as soon as workingFile became usable, i.e. as soon
	// as it carries a non-zero total length. That happens either right after
	// opening an existing cache file with a valid header, or after Init() has
	// consumed the resolved remote info.
	//
	// It exists because "the resolve info is available in the RemoteManager"
	// and "this cache file has consumed it" are two different moments, and
	// GetResource in between would report a bogus download failure.
	//
	// Guarded by workingFileMutex, and rebuilt on every open generation (see
	// resetInitializedLocked) so that a stale "ready" can never be observed
	// after the entry has been closed.
	initialized chan struct{}
}

// resetInitializedLocked rebuilds the readiness signal for the current open
// generation. Caller must hold workingFileMutex.
func (e *BaseCDNEntry) resetInitializedLocked() {
	ch := make(chan struct{})

	if e.workingFile != nil && e.workingFile.TotalLen() > 0 {
		// A cache file that already carries a valid header is usable without
		// resolving the remote info again.
		close(ch)
	}

	e.initialized = ch
}

// markInitializedLocked closes the readiness signal. It is idempotent and safe
// to call on an entry that was never opened. Caller must hold workingFileMutex.
func (e *BaseCDNEntry) markInitializedLocked() {
	if e.initialized == nil {
		e.initialized = make(chan struct{})
	}

	select {
	case <-e.initialized:
		// already closed
	default:
		close(e.initialized)
	}
}

// WaitInitialized blocks until this entry is ready to serve GetResource, i.e.
// until its working file has a non-zero total length, and returns as soon as
// ctx is done. Callers must pass a cancellable context.
func (e *BaseCDNEntry) WaitInitialized(ctx context.Context) error {
	e.workingFileMutex.Lock()

	if e.workingFile != nil && e.workingFile.TotalLen() > 0 {
		e.workingFileMutex.Unlock()
		return nil
	}

	ch := e.initialized
	if ch == nil {
		ch = make(chan struct{})
		e.initialized = ch
	}

	e.workingFileMutex.Unlock()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
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

	// The working file is usable from this point on, so release any waiter that
	// is blocked in WaitInitialized.
	e.markInitializedLocked()

	return e.updateMeta()
}

func (e *BaseCDNEntry) ReconcileRemoteInfo(info *types.RemoteHttpResourceInfo) {
	initNeeded := false

	e.workingFileMutex.RLock()
	if e.workingFile == nil {
		// Dropping the info silently would leave the entry permanently
		// uninitialized, so at least make the condition visible.
		e.logger.WarnLn("Dropped the resolved info of", e.id, "because the cache entry is closed")
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

	// A new open generation invalidates any previous readiness state.
	e.resetInitializedLocked()
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

	// The entry is no longer usable. Rebuild the signal so that a concurrent
	// WaitInitialized cannot observe the stale "ready" of the closed file.
	e.resetInitializedLocked()

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
