package download

import (
	"errors"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"io"
	"sync"
)

var ErrCanceled = errors.New("task canceled")

type State struct {
	sync.Mutex

	ID string

	cacheEntry cache.Entry

	TotalSize      int64
	DownloadedSize int64

	Done    bool
	Pending bool
	Error   error

	StateCh    chan *State
	CancelCh   chan bool
	PriorityCh chan int
}

func (ds *State) Write(p []byte) (int, error) {
	select {
	case <-ds.CancelCh:
		return 0, ErrCanceled
	default:
		if ds.BlockIfPending() {
			n := len(p)
			ds.DownloadedSize += int64(n)
			ds.StateCh <- ds
			return n, nil
		} else {
			return 0, ErrCanceled
		}
	}
}

// BlockIfPending keeps blocked until this task is able to continue or is canceled (returning false)
func (ds *State) BlockIfPending() bool {
	var priority int
	select {
	case priority = <-ds.PriorityCh:
		// continue checking
	default:
		// This means the priority have not been changed since the previous pending check
		// which approved the downloading task to continue
		return true
	}

	for {
		if dm.CanDownload(priority) {
			if ds.Pending {
				ds.Pending = false
				ds.StateCh <- ds
				logger.InfoLn("Continue download task", ds.ID)
			}
			return true
		} else {
			if !ds.Pending {
				ds.Pending = true
				ds.StateCh <- ds
				logger.InfoLnf("Paused download task %s, because its priority is %d", ds.ID, priority)
			}
		}
		select {
		case <-ds.CancelCh:
			return false
		case priority = <-ds.PriorityCh:
			// continue checking
		}
	}
}
func (ds *State) unlockAndNotify() {
	ds.Unlock()
	ds.StateCh <- ds
}

func (ds *State) progressiveDownload(body io.ReadCloser, writer io.Writer) error {
	// Write the body to file, while showing progress of the download
	_, err := io.Copy(writer, io.TeeReader(body, ds))
	if err != nil {
		return err
	}

	return nil
}

func Download(id string) *State {
	ds := dm.CreateOrGetState(id)
	if ds == nil {
		return nil
	}
	go func() {
		// Lock the state so that there is always only one download happening
		ds.Lock()
		defer ds.unlockAndNotify()

		// Clear error every time we start downloading
		ds.Error = nil

		if ds.Done {
			return
		}

		// Otherwise we start downloading, the total size is unknown
		ds.TotalSize = 0
		if !ds.BlockIfPending() {
			logger.InfoLn("Canceled download task", ds.ID)
			return
		}

		entry := ds.cacheEntry
		ds.TotalSize = entry.TotalLen()

		body, err := entry.GetDownloadBody()
		if err != nil {
			ds.Error = err
			logger.ErrorLn("Start Downloading error:", err.Error())
			return
		}
		defer body.Close()

		ds.DownloadedSize = entry.DownloadedSize()

		// Notify about the total size and that the request header is done
		ds.StateCh <- ds

		// Copy the body to the file, which will also update the download progress
		err = ds.progressiveDownload(body, entry)
		if err != nil {
			ds.Error = err
			if errors.Is(err, ErrCanceled) {
				// canceled task
				logger.InfoLn("Canceled download task", ds.ID)
			} else {
				logger.ErrorLn("Downloading error:", err.Error())
			}
			return
		}

		err = entry.Save()
		if err != nil {
			ds.Error = err
			logger.ErrorLn("Saving error:", err.Error())
			return
		}

		// Mark the download as done and update the priorities
		ds.Done = true
		dm.UpdatePriorities()
	}()

	return ds
}
