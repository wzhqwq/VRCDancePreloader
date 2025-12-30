package download

import (
	"errors"
	"io"
	"log"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
)

var ErrCanceled = errors.New("task canceled")
var ErrRestarted = errors.New("task restarted")

func (t *Task) Write(p []byte) (int, error) {
	select {
	case <-t.CancelCh:
		return 0, ErrCanceled
	case <-t.RestartCh:
		// force close current network connection and then continue downloading
		return 0, ErrRestarted
	default:
		if t.BlockIfPending() {
			n := len(p)
			t.addBytes(int64(n))
			return n, nil
		}

		return 0, ErrCanceled
	}
}

func (t *Task) progressiveDownload(body io.ReadCloser, writer io.Writer) error {
	// Write the body to file, while showing progress of the download
	_, err := io.Copy(writer, io.TeeReader(body, t))
	if err != nil {
		return err
	}

	return nil
}

func (t *Task) singleDownload(entry cache.Entry) error {
	body, err := entry.GetDownloadStream()
	if err != nil {
		return err
	}
	if body == nil {
		// already downloaded, save fragments
		return nil
	}
	defer body.Close()

	t.DownloadedSize = entry.DownloadedSize()
	t.Requesting = false
	t.resetEta()

	// Notify about the total size and that the request header is done
	t.notifyStateChange()

	// Copy the body to the file, which will also update the download progress
	return t.progressiveDownload(body, entry)
}

func (t *Task) markAsDone() {
	t.DownloadedSize = t.TotalSize
	t.Done = true
	t.Error = nil
}

func (t *Task) Download(retryDelay bool) {
	// Lock the state so that there is always only one download happening
	t.Lock()
	defer t.unlockAndNotifyStateChange()

	cacheEntry, err := cache.OpenCacheEntry(t.ID, "[Downloader]")
	if err != nil {
		t.Error = err
		log.Println("Skipped", t.ID, "due to", err)
		return
	}
	defer cache.ReleaseCacheEntry(t.ID, "[Downloader]")

	// Check if file is already downloaded
	if cacheEntry.IsComplete() {
		logger.InfoLn("Already downloaded", t.ID)
		t.TotalSize, _ = cacheEntry.TotalLen()
		t.markAsDone()
		return
	}

	t.Error = nil
	t.Pending = false
	var delay time.Duration

	// check if the task is canceled or paused
	if !t.BlockIfPending() {
		goto canceled
	}

	// delay or cool down before we start downloading
	delay = t.manager.getDelay(retryDelay)
	if delay > 0 {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			if retryDelay {
				select {
				case <-t.CancelCh:
					return
				case <-time.After(t.manager.scheduler.Delay()):
				}
			}
			t.Cooling = true
			t.notifyStateChange()
		}()

		select {
		case <-t.CancelCh:
			goto canceled
		case <-time.After(delay):
		}
		wg.Wait()
	}
	t.Cooling = false

	t.Requesting = true
	t.notifyStateChange()

	t.TotalSize, err = cacheEntry.TotalLen()
	if err != nil {
		t.Error = err
		logger.ErrorLn("Failed to get total size of", t.ID, ":", err)
		if errors.Is(err, cache.ErrThrottle) {
			t.manager.slowDown()
		}
		return
	}

	if !t.BlockIfPending() {
		goto canceled
	}

startTask:
	err = t.singleDownload(cacheEntry)
	if err == nil || cacheEntry.IsComplete() {
		logger.InfoLn("Downloaded", t.ID)
		t.markAsDone()
		return
	}

	if errors.Is(err, ErrCanceled) {
		goto canceled
	}
	if errors.Is(err, io.EOF) || errors.Is(err, ErrRestarted) {
		logger.InfoLn("Switch to another offset", t.ID)
		t.Requesting = true
		goto startTask
	}

	t.Error = err
	logger.ErrorLn("Downloading error:", err.Error(), t.ID)
	if errors.Is(err, cache.ErrThrottle) {
		t.manager.slowDown()
	}
	return

canceled:
	t.Error = ErrCanceled
	logger.InfoLn("Canceled download task", t.ID)
	return
}
