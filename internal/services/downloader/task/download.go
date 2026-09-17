package task

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/rw_file/fragmented"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

var logger = utils.NewLogger("Download Task")

var ErrCanceled = errors.New("task canceled")
var ErrRestarted = errors.New("task restarted")
var ErrConnectionTimeoutClosed = errors.New("connection timed out")

// minBodyRequestInterval is the minimum interval between two body requests of
// the same task. It keeps a restart loop (for example a connection that is
// closed right after the response headers) from re-issuing requests at full
// speed.
const minBodyRequestInterval = time.Millisecond * 100

// unwrapError restores the cancellation cause of a
// context.WithCancelCause-based context.
//
// ctx.Err() deliberately collapses every cancellation into
// context.Canceled, which makes the callers' errors.Is(err, ErrCanceled)
// checks fail: Cancel, Restart and CloseConnection all pass a specific cause.
// context.Cause(ctx) keeps that reason.
//
// When the task context itself is not canceled the cancellation comes from
// another context (the remote manager cancelling a fetch, the HTTP client
// being reconfigured, ...). In that case the original error must be preserved
// instead of being replaced by a nil ctx.Err(), which the resolve loop would
// mistake for a successful resolution.
func unwrapError(err error, ctx context.Context) error {
	if errors.Is(err, context.Canceled) {
		if cause := context.Cause(ctx); cause != nil {
			return cause
		}
	}
	return err
}

// waitBodyRequestInterval paces consecutive body requests of this task and
// returns early when ctx is done, so that Cancel stays responsive.
//
// lastBodyRequest is only ever touched by the download goroutine (Download is
// serialized by the downloading flag), so it needs no lock.
func (t *Task) waitBodyRequestInterval(ctx context.Context) error {
	// Check the context first, like wait() does: a cancelled task must not issue
	// another request just because the pacing window has already elapsed. The
	// no-queue traffic control does not veto on its own, so this is the check
	// that keeps "cancelled" from turning into "one more request".
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}

	elapsed := time.Since(t.lastBodyRequest)
	if elapsed >= minBodyRequestInterval {
		return nil
	}

	delay := minBodyRequestInterval - elapsed
	logger.DebugLn(t.ID, "delays", delay, "before the next body request")

	select {
	case <-ctx.Done():
		return context.Cause(ctx)
	case <-time.After(delay):
		return nil
	}
}

func (t *Task) Write(p []byte) (int, error) {
	if err := t.waitPending(true); err != nil {
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
	if err := t.waitPending(false); err != nil {
		return err
	}

	if err := t.waitBodyRequestInterval(ctx); err != nil {
		return err
	}
	t.lastBodyRequest = time.Now()

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

func wait(ctx context.Context, until time.Time) error {
	// Check the context first: an already canceled task must not proceed, even
	// when the requested deadline is already in the past.
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}

	d := time.Until(until)
	if d > 0 {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-time.After(d):
			return nil
		}
	}

	return nil
}

func (t *Task) singleResolve(ctx context.Context) error {
	if err := wait(ctx, t.ResolverStatus.RetryAfter); err != nil {
		return err
	}
	if err := wait(ctx, t.ResolverStatus.CooldownUntil); err != nil {
		return err
	}
	if err := t.waitPending(false); err != nil {
		return err
	}

	totalLen, err := t.Remote.WaitResolving(ctx, func(status interactive.RemoteStatus) {
		t.ResolverStatus = status
		if status.Fetching() {
			t.setState(TaskResolving)
		} else {
			t.setState(TaskResolvingFailed)
		}
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
	if !t.run.tryBegin() {
		return
	}
	defer t.run.end()

	if err := t.Local.Open(); err != nil {
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

	t.setState(TaskInitial)

	// Every attempt installs its own cancellation scope, so that a Restart or a
	// connection timeout only aborts the current attempt instead of poisoning
	// the task permanently. See the comment on runControl in run_control.go.
	for {
		if t.run.cancelled() {
			goto canceled
		}

		ctx, cancel := t.run.beginAttempt()
		err := unwrapError(t.singleResolve(ctx), ctx)
		t.run.endAttempt(cancel)

		if err == nil {
			break
		}

		if errors.Is(err, ErrCanceled) {
			goto canceled
		}
		if errors.Is(err, interactive.ErrUnrecoverable) {
			t.setError(err)
			return
		}
		logger.ErrorLn("Failed to resolve download task", t.ID, err.Error())
	}

	// Check again
	if t.Local.IsComplete() {
		logger.InfoLn("Already downloaded", t.ID)
		t.TotalSize = t.Local.DownloadedSize()
		t.markAsDone()
		return
	}

	for {
		if t.run.cancelled() {
			goto canceled
		}

		ctx, cancel := t.run.beginAttempt()
		err := unwrapError(t.singleDownload(ctx), ctx)
		t.run.endAttempt(cancel)

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
			errors.Is(err, ErrConnectionTimeoutClosed) ||
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
