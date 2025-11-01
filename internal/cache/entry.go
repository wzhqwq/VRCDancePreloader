package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/continuous"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/fragmented"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/legacy_file"
)

var unixEpochTime = time.Unix(0, 0)

var ErrThrottle = errors.New("too many requests, slow down")

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
	GetDownloadStream() (io.ReadCloser, error)
	IsComplete() bool
	ModTime() time.Time
	UpdateReqRangeStart(start int64)
}

type BaseEntry struct {
	id     string
	client *http.Client

	openCount atomic.Int32

	workingFileMutex sync.RWMutex

	workingFile rw_file.DeferredReadableFile
}

func ConstructBaseEntry(id string, client *http.Client) BaseEntry {
	return BaseEntry{
		id:     id,
		client: client,
	}
}

func (e *BaseEntry) getVideoName() string {
	return fmt.Sprintf("%s/%s.mp4", cachePath, e.id)
}

func (e *BaseEntry) checkLegacy() bool {
	if _, err := os.Stat(e.getVideoName()); err == nil {
		return true
	}
	if _, err := os.Stat(e.getVideoName() + ".dl"); err == nil {
		return true
	}
	return false
}

// extendable operations

func (e *BaseEntry) openFile() {
	e.workingFileMutex.Lock()
	defer e.workingFileMutex.Unlock()

	if e.workingFile != nil {
		return
	}

	if e.checkLegacy() {
		e.workingFile = legacy_file.NewFile(e.getVideoName())
	}

	if e.workingFile == nil {
		switch fileFormat {
		case 1:
			e.workingFile = continuous.NewFile(e.getVideoName())
		case 2:
			e.workingFile = fragmented.NewFile(e.getVideoName())
		default:
			e.workingFile = legacy_file.NewFile(e.getVideoName())
		}
	}
}

func (e *BaseEntry) closeFile() error {
	e.workingFileMutex.Lock()
	defer e.workingFileMutex.Unlock()

	if e.workingFile == nil {
		return nil
	}

	err := e.workingFile.Close()
	if err == nil || errors.Is(err, os.ErrClosed) {
		e.workingFile = nil
	}

	return err
}

func (e *BaseEntry) getReadSeeker(ctx context.Context) (io.ReadSeeker, error) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.workingFile == nil {
		return nil, io.ErrClosedPipe
	}

	r := e.workingFile.RequestRs(ctx)
	if r == nil {
		return nil, errors.New("failed to download this video")
	}

	return r, nil
}

// http utils

type RemoteVideoInfo struct {
	FinalUrl     string
	TotalSize    int64
	LastModified time.Time
}

func (e *BaseEntry) requestHttpResInfo(url string, ctx context.Context) (*RemoteVideoInfo, error) {
	log.Println("request info", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	res, err := e.client.Do(req)
	if err != nil {
		log.Println("Failed to get ", url, "reason:", err)
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusTooManyRequests {
		return nil, ErrThrottle
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	lastModified := time.Unix(0, 0)
	if lastModifiedText := res.Header.Get("Last-Modified"); lastModifiedText != "" {
		lastModified, _ = http.ParseTime(lastModifiedText)
	}

	return &RemoteVideoInfo{
		FinalUrl:     res.Request.URL.String(),
		TotalSize:    res.ContentLength,
		LastModified: lastModified,
	}, nil
}
func (e *BaseEntry) requestHttpResBody(url string, offset int64, ctx context.Context) (io.ReadCloser, error) {
	log.Println("request body", url, offset)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
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
	e.openFile()
}

func (e *BaseEntry) Release() {
	e.openCount.Add(-1)
	if e.openCount.Load() <= 0 {
		go func() {
			<-time.After(time.Second)
			if e.openCount.Load() <= 0 {
				e.closeFile()
			}
		}()
	}
}

func (e *BaseEntry) Active() bool {
	return e.openCount.Load() > 0
}

func (e *BaseEntry) Close() error {
	return e.closeFile()
}

func (e *BaseEntry) IsComplete() bool {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.workingFile == nil {
		return false
	}
	return e.workingFile.IsComplete()
}

func (e *BaseEntry) DownloadedSize() int64 {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.workingFile == nil {
		return 0
	}
	return e.workingFile.GetDownloadedBytes()
}

func (e *BaseEntry) ModTime() time.Time {
	// TODO
	return unixEpochTime
}

func (e *BaseEntry) Write(bytes []byte) (int, error) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.workingFile == nil {
		return 0, io.ErrClosedPipe
	}
	return e.workingFile.Append(bytes)
}

func (e *BaseEntry) UpdateReqRangeStart(start int64) {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.workingFile != nil {
		e.workingFile.NotifyRequestStart(start)
	}
}
