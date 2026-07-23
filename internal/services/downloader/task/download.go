package task

import (
	"context"
	"errors"
	"io"

	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/fragmented"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewLogger("Download Task")

var ErrCanceled = errors.New("task canceled")
var ErrRestarted = errors.New("task restarted")

func unwrapError(err error, ctx context.Context) error {
	if errors.Is(err, context.Canceled) {
		return ctx.Err()
	}
	return err
}

func (t *Task) Write(p []byte) (int, error) {
	if err := t.waitPending(); err != nil {
		return 0, err
	}

	n := len(p)
	t.addBytes(int64(n))

	return n, nil
}

func (t *Task) progressiveDownload(body io.ReadCloser) error {
	// Write the body to file, while showing progress of the download
	_, err := io.Copy(t.Local, io.TeeReader(body, t))
	return err
}

func (t *Task) singleDownload(ctx context.Context) error {
	t.connected = true
	defer func() {
		t.connected = false
	}()

	t.setState(TaskRequested)
	offset, err := t.Local.CurrentCursor()
	if err != nil {
		return err
	}

	stream, err := t.Remote.GetDownloadStream(offset, ctx)

	if cause := context.Cause(ctx); errors.Is(err, context.Canceled) && cause != nil {
		// canceled by myself
		return cause
	}
	if err != nil {
		return err
	}
	if stream.Rc == nil {
		// already downloaded, save fragments
		return nil
	}
	defer stream.Rc.Close()

	if !stream.RangeAvailable {
		err = t.Local.SeekStart()
		if err != nil {
			return err
		}
	}

	if t.TotalSize == 0 {
		t.TotalSize = stream.Length
	}
	t.DownloadedSize = t.Local.DownloadedSize()
	t.resetEta()

	// Notify about the total size and that the request header is done
	t.setState(TaskDownloading)

	// Copy the body to the file, which will also update the download progress
	return t.progressiveDownload(stream.Rc)
}

func (t *Task) singleResolve(ctx context.Context) error {
	totalLen, err := t.Remote.WaitResolving(ctx, func() {
		t.setState(TaskResolving)
	})
	if err != nil {
		return err
	}

	t.TotalSize = totalLen
	return nil
}

func (t *Task) markAsDone() {
	t.DownloadedSize = t.TotalSize
	t.setState(TaskCompleted)
}

func (t *Task) Download() {
	if !t.downloading.CompareAndSwap(false, true) {
		return
	}
	defer func() {
		t.downloading.Store(false)
	}()

	err := t.Local.Open()
	if err != nil {
		t.setError(err)
		logger.WarnLn(t.ID, "is not downloaded because", err.Error())
		return
	}
	defer t.Local.Close()

	// Check if file is already downloaded
	if !t.Local.IsForceResolving() && t.Local.IsComplete() {
		logger.InfoLn("Already downloaded", t.ID)
		t.TotalSize = t.Local.DownloadedSize()
		t.markAsDone()
		return
	}

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	t.cancelFn = cancel

	if errors.Is(t.waitSchedule(), ErrCanceled) {
		goto canceled
	}

	t.setState(TaskInitial)

	if errors.Is(t.waitPending(), ErrCanceled) {
		goto canceled
	}

	for {
		err = unwrapError(t.singleResolve(ctx), ctx)
		if err != nil {
			if errors.Is(err, requesting.ErrClientChanged) {
				continue
			}
			if errors.Is(err, ErrCanceled) {
				goto canceled
			}
			t.setError(err)
			logger.ErrorLn("Failed to resolve download task", t.ID)
		}
		break
	}

	// Check again
	if t.Local.IsComplete() {
		logger.InfoLn("Already downloaded", t.ID)
		t.TotalSize = t.Local.DownloadedSize()
		t.markAsDone()
		return
	}

	for {
		if errors.Is(t.waitPending(), ErrCanceled) {
			goto canceled
		}

		err = unwrapError(t.singleDownload(ctx), ctx)
		if err == nil || t.Local.IsComplete() {
			logger.InfoLn("Downloaded", t.ID)
			t.markAsDone()
			return
		}

		if errors.Is(err, ErrCanceled) {
			goto canceled
		}

		if errors.Is(err, io.EOF) ||
			errors.Is(err, fragmented.ErrEndOfFragment) ||
			errors.Is(err, ErrRestarted) ||
			errors.Is(err, requesting.ErrClientChanged) {

			logger.InfoLn("Restarted", t.ID, "reason:", err.Error())
			continue
		}

		t.setError(err)
		logger.ErrorLn("Downloading error:", err.Error(), t.ID)
		return
	}

canceled:
	t.setError(ErrCanceled)
	logger.InfoLn("Canceled download task", t.ID)
	return
}
