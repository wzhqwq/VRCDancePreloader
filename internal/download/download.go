package download

import (
	"errors"
	"io"
	"log"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var ErrCanceled = errors.New("task canceled")

var downloadScheduler = utils.NewScheduler(time.Second*3, time.Second)

type State struct {
	sync.Mutex

	ID string

	TotalSize      int64
	DownloadedSize int64

	Requesting bool
	Done       bool
	Pending    bool
	Cooling    bool
	Error      error

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
			ds.notify()
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
				ds.notify()
				logger.InfoLn("Continue download task", ds.ID)
			}
			return true
		} else {
			if !ds.Pending {
				ds.Pending = true
				ds.notify()
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
	ds.notify()
}

func (ds *State) notify() {
	select {
	case ds.StateCh <- ds:
	default:
	}
}

func (ds *State) progressiveDownload(body io.ReadCloser, writer io.Writer) error {
	// Write the body to file, while showing progress of the download
	_, err := io.Copy(writer, io.TeeReader(body, ds))
	if err != nil {
		return err
	}

	return nil
}

func (ds *State) singleDownload(entry cache.Entry) error {
	body, err := entry.GetDownloadStream()
	if err != nil {
		return err
	}
	if body == nil {
		// already downloaded, save fragments
		return nil
	}
	defer body.Close()

	ds.DownloadedSize = entry.DownloadedSize()
	ds.Requesting = false

	// Notify about the total size and that the request header is done
	ds.notify()

	// Copy the body to the file, which will also update the download progress
	return ds.progressiveDownload(body, entry)
}

func (ds *State) markAsDone() {
	ds.DownloadedSize = ds.TotalSize
	ds.Done = true
	ds.Error = nil
}

func (ds *State) Download(retryDelay bool) {
	// Lock the state so that there is always only one download happening
	ds.Lock()
	defer ds.unlockAndNotify()

	cacheEntry, err := cache.OpenCacheEntry(ds.ID, "[Downloader]")
	if err != nil {
		ds.Error = err
		log.Println("Skipped", ds.ID, "due to", err)
		return
	}
	defer cache.ReleaseCacheEntry(ds.ID, "[Downloader]")

	// Check if file is already downloaded
	if cacheEntry.IsComplete() {
		logger.InfoLn("Already downloaded", ds.ID)
		ds.TotalSize, _ = cacheEntry.TotalLen()
		ds.markAsDone()
		return
	}

	// check if the task is canceled or paused
	ds.Pending = false
	if !ds.BlockIfPending() {
		ds.Error = ErrCanceled
		logger.InfoLn("Canceled download task", ds.ID)
		return
	}

	// delay or cool down before we start downloading
	delay := time.Duration(0)
	if retryDelay {
		delay = downloadScheduler.ReserveWithDelay()
	} else {
		delay = downloadScheduler.Reserve()
	}
	if delay > 0 {
		ds.Cooling = true
		ds.notify()

		select {
		case <-ds.CancelCh:
			ds.Error = ErrCanceled
			logger.InfoLn("Canceled download task", ds.ID)
			return
		case <-time.After(delay):
		}
	}
	ds.Cooling = false

	ds.Error = nil
	ds.Requesting = true
	ds.notify()

	ds.TotalSize, err = cacheEntry.TotalLen()
	if err != nil {
		ds.Error = err
		logger.ErrorLn("Failed to get total size of", ds.ID, ":", err)
		if errors.Is(err, cache.ErrThrottle) {
			slowDown()
		}
		return
	}

	for {
		if !ds.BlockIfPending() {
			ds.Error = ErrCanceled
			logger.InfoLn("Canceled download task", ds.ID)
			return
		}

		err = ds.singleDownload(cacheEntry)
		if err == nil || cacheEntry.IsComplete() {
			logger.InfoLn("Downloaded", ds.ID)
			ds.markAsDone()
			return
		}

		if !errors.Is(err, io.EOF) {
			ds.Error = err

			if errors.Is(err, ErrCanceled) {
				// canceled task
				logger.InfoLn("Canceled download task", ds.ID)
			} else {
				logger.ErrorLn("Downloading error:", err.Error(), ds.ID)
				if errors.Is(err, cache.ErrThrottle) {
					slowDown()
				}
			}
			return
		}

		logger.InfoLn("Switch to another offset", ds.ID)
	}
}

func Download(id string) *State {
	ds := dm.CreateOrGetState(id)
	if ds == nil {
		return nil
	}
	go func() {
		ds.Download(false)
		dm.UpdatePriorities()
	}()

	return ds
}

func Retry(ds *State) {
	go func() {
		ds.Download(true)
		dm.UpdatePriorities()
	}()
}

func slowDown() {
	downloadScheduler.Throttle()
	go func() {
		// PyPyDance seems to ban you for 3 minute if you have been requesting so fast
		<-time.After(time.Minute * 3)
		downloadScheduler.ReleaseOneThrottle()
	}()
}

func SubscribeCoolDownInterval() *utils.EventSubscriber[time.Duration] {
	return downloadScheduler.SubscribeIntervalEvent()
}
