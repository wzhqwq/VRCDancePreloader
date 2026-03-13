package entry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache/cache_fs"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/continuous"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/fragmented"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/legacy_file"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var unixEpochTime = time.Time{}

var ErrThrottle = errors.New("too many requests, slow down")
var ErrNotSupported = errors.New("video is not currently supported")
var ErrUpgrading = errors.New("video cache is upgrading")

type Entry interface {
	io.Writer

	// Open this entry. It's a heavy method with locks
	Open()

	// Release this entry. It's a lightweight method with no locks
	Release()

	// Close this entry. It's a heavy method with locks
	Close() error

	Active() bool
	TotalLen() (int64, error)
	DownloadedSize() int64
	GetReadSeeker(ctx context.Context) (io.ReadSeeker, error)
	GetDownloadStream(ctx context.Context) (io.ReadCloser, error)
	IsComplete() bool
	ModTime() time.Time
	Etag() string
	UpdateReqRangeStart(start int64)
}

type BaseEntry struct {
	id     string
	client *requesting.ClientProvider

	referer string
	etag    string

	openCount atomic.Int32

	workingFileMutex sync.RWMutex
	fileUpgrading    atomic.Bool
	fileClosing      atomic.Bool

	workingFile rw_file.DeferredReadableFile

	logger *utils.CustomLogger

	baseName string

	meta *persistence.CacheMeta
}

func ConstructBaseEntry(id string, client *requesting.ClientProvider) BaseEntry {
	return BaseEntry{
		id:     id,
		client: client,
		logger: utils.NewLogger("Cached " + id),

		baseName: "video$" + id,
	}
}

func (e *BaseEntry) checkLegacy() bool {
	if cache_fs.Exists(e.baseName + ".vrcdp") {
		return false
	}
	if cache_fs.Exists(e.baseName+".mp4") || cache_fs.Exists(e.baseName+".mp4.dl") {
		return true
	}

	return false
}

// extendable operations (please wrap with mutex by yourself and check workingFile first!!)

func (e *BaseEntry) openFile() {
	if e.checkLegacy() {
		e.workingFile = legacy_file.NewFile(e.baseName)
	}

	if e.workingFile == nil {
		switch fileFormat {
		case 1:
			e.workingFile = continuous.NewFile(e.baseName)
		case 2:
			e.workingFile = fragmented.NewFile(e.baseName)
		}
	}

	e.syncWithFS()
}

func (e *BaseEntry) closeFile() error {
	err := e.workingFile.Close()
	if err == nil || errors.Is(err, os.ErrClosed) {
		e.workingFile = nil
	}

	return err
}

func (e *BaseEntry) upgradeFile() {
	e.workingFileMutex.Lock()
	defer func() {
		e.fileUpgrading.Store(false)
		e.workingFileMutex.Unlock()
	}()

	if e.workingFile == nil {
		e.logger.WarnLn("Try to upgrade a closed file")
		return
	}
	if _, ok := e.workingFile.(*legacy_file.File); !ok {
		e.logger.WarnLn("Try to upgrade a non-legacy file")
		return
	}

	err := e.workingFile.Close()
	if err != nil {
		e.logger.ErrorLn("Failed to close working file", err)
		return
	}
	e.workingFile = nil

	err = cache_fs.DeleteWithoutExt("video$" + e.id)
	if err != nil {
		e.logger.ErrorLn("Failed to delete old file", err)
	}

	e.openFile()
}

func (e *BaseEntry) Upgrade() {
	e.fileUpgrading.Store(true)
	go e.upgradeFile()
}

func (e *BaseEntry) acquireFileRLock() {
	// avoid starving upgrading and closing lock
	for e.fileUpgrading.Load() {
		runtime.Gosched()
	}
	for e.fileClosing.Load() {
		runtime.Gosched()
	}
	e.workingFileMutex.RLock()
}

func (e *BaseEntry) releaseFileRLock() {
	e.workingFileMutex.RUnlock()
}

func (e *BaseEntry) getReadSeeker(ctx context.Context) (io.ReadSeeker, error) {
	r := e.workingFile.RequestRs(ctx)
	if r == nil {
		return nil, errors.New("failed to download this video")
	}

	return r, nil
}

// http utils

func (e *BaseEntry) requestHttpResBody(url string, offset int64, ctx context.Context) (io.ReadCloser, error) {
	e.logger.InfoLn("Request body", url, offset)
	req, err := e.client.NewGetRequest(url, ctx)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	requesting.SetupHeader(req, e.referer)
	res, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusTooManyRequests {
		return nil, ErrThrottle
	}
	if res.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		return nil, nil
	}
	if res.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	return res.Body, nil
}

// adapters

func (e *BaseEntry) Open() {
	e.openCount.Add(1)

	e.workingFileMutex.Lock()
	defer e.workingFileMutex.Unlock()

	if e.workingFile == nil {
		e.openFile()
	}
}

func (e *BaseEntry) Release() {
	e.openCount.Add(-1)
	if e.openCount.Load() <= 0 {
		go func() {
			<-time.After(time.Second)
			if e.openCount.Load() <= 0 {
				err := e.Close()
				if err != nil {
					e.logger.ErrorLn("failed to close file", e.id, err)
				}
			}
		}()
	}
}

func (e *BaseEntry) Active() bool {
	return e.openCount.Load() > 0
}

func (e *BaseEntry) Close() error {
	e.fileClosing.Store(true)
	e.workingFileMutex.Lock()
	defer func() {
		e.fileClosing.Store(false)
		e.workingFileMutex.Unlock()
	}()

	if e.workingFile == nil {
		return nil
	}
	return e.closeFile()
}

func (e *BaseEntry) IsComplete() bool {
	e.acquireFileRLock()
	defer e.releaseFileRLock()

	if e.workingFile == nil {
		return false
	}
	return e.checkIfCompleteAndSync()
}

func (e *BaseEntry) DownloadedSize() int64 {
	e.acquireFileRLock()
	defer e.releaseFileRLock()

	if e.workingFile == nil {
		return 0
	}
	return e.workingFile.GetDownloadedBytes()
}

func (e *BaseEntry) Etag() string {
	return e.etag
}

func (e *BaseEntry) Write(bytes []byte) (int, error) {
	e.acquireFileRLock()
	defer e.releaseFileRLock()

	if e.workingFile == nil {
		return 0, io.ErrClosedPipe
	}
	return e.workingFile.Append(bytes)
}

func (e *BaseEntry) UpdateReqRangeStart(start int64) {
	e.acquireFileRLock()
	defer e.releaseFileRLock()

	if e.workingFile != nil {
		e.workingFile.NotifyRequestStart(start)
	}
}
