package download

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/fragmented"
)

var ErrCanceled = errors.New("task canceled")
var ErrRestarted = errors.New("task restarted")

func (t *Task) Write(p []byte) (int, error) {
	err := t.BlockIfPending()
	if err != nil {
		return 0, err
	}

	n := len(p)
	t.addBytes(int64(n))

	return n, nil
}

func (t *Task) progressiveDownload(body io.ReadCloser, writer io.Writer) error {
	// Write the body to file, while showing progress of the download
	_, err := io.Copy(writer, io.TeeReader(body, t))
	return err
}

func (t *Task) singleDownload(entry cache.Entry) error {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	go func() {
		// error route for requesting
		select {
		case <-t.CancelCh:
			cancel(ErrCanceled)
		case <-t.RestartCh:
			cancel(ErrRestarted)
		case <-ctx.Done():
		}
	}()

	t.connected = true
	defer func() {
		t.connected = false
	}()

	body, err := entry.GetDownloadStream(ctx)
	if cause := context.Cause(ctx); errors.Is(err, context.Canceled) && cause != nil {
		// canceled by myself
		return cause
	}
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

	cacheEntry, err := cache.OpenCacheEntry(t.ID, logger)
	if err != nil {
		t.Error = err
		logger.WarnLn("Skipped", t.ID, "due to", err)
		return
	}
	defer cache.ReleaseCacheEntry(t.ID, logger)

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
	if errors.Is(t.BlockIfPending(), ErrCanceled) {
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

startRequest:
	t.TotalSize, err = cacheEntry.TotalLen()
	if err != nil {
		if errors.Is(err, requesting.ErrClientChanged) {
			logger.InfoLn("Restarted", t.ID, "reason:", err.Error())
			goto startRequest
		}

		t.Error = err
		logger.ErrorLn("Failed to get total size of", t.ID, ":", err)
		if errors.Is(err, cache.ErrThrottle) {
			t.manager.slowDown()
		}
		return
	}

startTask:
	err = t.BlockIfPending()

	if err == nil {
		err = t.singleDownload(cacheEntry)
		if err == nil || cacheEntry.IsComplete() {
			logger.InfoLn("Downloaded", t.ID)
			t.markAsDone()
			return
		}
	}

	if errors.Is(err, ErrCanceled) {
		goto canceled
	}
	if errors.Is(err, io.EOF) ||
		errors.Is(err, fragmented.ErrEndOfFragment) ||
		errors.Is(err, ErrRestarted) ||
		errors.Is(err, requesting.ErrClientChanged) {

		logger.InfoLn("Restarted", t.ID, "reason:", err.Error())
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
