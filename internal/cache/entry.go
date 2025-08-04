package cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache/fragmented"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache/rw_file"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var unixEpochTime = time.Unix(0, 0)

type Entry interface {
	io.Writer
	Open()
	Release()
	Active() bool
	TotalLen() int64
	DownloadedSize() int64
	GetReadSeeker(ctx context.Context) io.ReadSeeker
	GetDownloadStream() (io.ReadCloser, error)
	Close() error
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
func (e *BaseEntry) openFile() {
	e.workingFileMutex.Lock()
	defer e.workingFileMutex.Unlock()

	if e.workingFile == nil {
		if enableFragmented {
			e.workingFile = fragmented.NewFile(e.getVideoName())
		} else {
			e.workingFile = rw_file.NewFile(e.getVideoName())
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

func (e *BaseEntry) requestHttpResInfo(url string) (int64, string) {
	res, err := e.client.Head(url)
	if err != nil {
		return 0, url
	}
	if res.StatusCode >= 400 {
		return 0, url
	}
	if location := res.Header.Get("Location"); location != "" {
		url = location
	}
	return res.ContentLength, url
}
func (e *BaseEntry) requestHttpResBody(url string, offset int64) (io.ReadCloser, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	res, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}

	if offset > 0 {
		if res.StatusCode == 416 {
			return nil, nil
		}
		if res.StatusCode != http.StatusPartialContent {
			return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
		}
		return res.Body, nil
	} else {
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
		}
		return res.Body, nil
	}
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

func (e *BaseEntry) GetReadSeeker(ctx context.Context) io.ReadSeeker {
	e.workingFileMutex.RLock()
	defer e.workingFileMutex.RUnlock()

	if e.workingFile == nil {
		return nil
	}

	r := e.workingFile.RequestRsc()

	go func() {
		<-ctx.Done()
		r.Close()
	}()

	return r
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
