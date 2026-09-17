package task

import (
	"context"
	"errors"
	"testing"
	"time"
)

// T2 — Task level cancellation and restart combinations.
//
// These tests drive the real Download() loop against a scriptable
// RemoteProvider, so they cover the interaction that run_control_test.go can
// only assert in isolation: what the loop does after an attempt is aborted.

const testTimeout = 5 * time.Second

// startDownload runs Download() in the background and returns a channel that is
// closed when it returns.
func startDownload(t *testing.T, task *Task) <-chan struct{} {
	t.Helper()

	done := make(chan struct{})
	go func() {
		defer close(done)
		task.Download()
	}()
	return done
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(testTimeout):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func waitClosed(t *testing.T, done <-chan struct{}, what string) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatalf("timed out waiting for %s", what)
	}
}

// A Cancel() issued before Download() means the task never starts an attempt.
// Restart() afterwards must be a no-op as well.
func TestTaskCancelBeforeDownload(t *testing.T) {
	remote := newFakeRemote()
	local := &fakeLocal{}
	task := NewTask("test", remote, local)

	task.Cancel()
	task.Restart()

	waitClosed(t, startDownload(t, task), "Download to return")

	resolve, stream := remote.counts()
	if resolve != 0 || stream != 0 {
		t.Fatalf("cancelled task issued requests: resolve=%d stream=%d", resolve, stream)
	}
}

// Aborting an attempt must not end the task: the loop has to start a fresh
// attempt with a live context. This is the regression that used to turn a
// single connection timeout into an infinite restart loop with zero progress.
func TestTaskCloseConnectionRestartsWithLiveContext(t *testing.T) {
	remote := newFakeRemote()
	remote.blockStream = make(chan struct{})
	local := &fakeLocal{}
	task := NewTask("test", remote, local)

	done := startDownload(t, task)

	// Attempt 1 is in flight.
	waitFor(t, remote.entered, "the first GetDownloadStream call")

	task.CloseConnection()

	// Attempt 2 must arrive, which is only possible if the loop installed a
	// fresh context after the abort.
	waitFor(t, remote.entered, "a second GetDownloadStream call after CloseConnection")

	_, stream := remote.counts()
	if stream < 2 {
		t.Fatalf("stream calls = %d, want at least 2", stream)
	}

	// The second attempt's context must be alive: that is what lets the task
	// make progress instead of failing on a poisoned context forever.
	errs := remote.ctxErrs()
	if len(errs) < 2 {
		t.Fatalf("recorded %d contexts, want at least 2", len(errs))
	}
	if errs[0] != nil {
		t.Fatalf("first attempt context was already cancelled: %v", errs[0])
	}
	if errs[1] != nil {
		t.Fatalf("second attempt context was already cancelled: %v "+
			"(the loop reused the poisoned context)", errs[1])
	}

	task.Cancel()
	waitClosed(t, done, "Download to return after Cancel")
}

// Cancel() during an in-flight attempt aborts it with ErrCanceled and the loop
// terminates instead of retrying.
func TestTaskCancelDuringAttempt(t *testing.T) {
	remote := newFakeRemote()
	remote.blockStream = make(chan struct{})
	local := &fakeLocal{}
	task := NewTask("test", remote, local)

	done := startDownload(t, task)

	waitFor(t, remote.entered, "the first GetDownloadStream call")

	task.Cancel()
	waitClosed(t, done, "Download to return after Cancel")

	if !errors.Is(task.Error, ErrCanceled) {
		t.Fatalf("task error = %v, want ErrCanceled", task.Error)
	}

	_, stream := remote.counts()
	if stream != 1 {
		t.Fatalf("stream calls = %d, want 1: a cancelled task must not retry", stream)
	}

	// Restart after Cancel is a no-op, and Download cannot be re-entered.
	task.Restart()
	waitClosed(t, startDownload(t, task), "the second Download to return")

	_, stream = remote.counts()
	if stream != 1 {
		t.Fatalf("stream calls = %d after a restart on a cancelled task", stream)
	}
}

// Download() is not re-entrant: a second call while the loop is running returns
// immediately without starting another attempt.
func TestTaskDownloadIsNotReentrant(t *testing.T) {
	remote := newFakeRemote()
	remote.blockStream = make(chan struct{})
	local := &fakeLocal{}
	task := NewTask("test", remote, local)

	done := startDownload(t, task)
	waitFor(t, remote.entered, "the first GetDownloadStream call")

	// The loop holds the run slot, so this returns at once.
	waitClosed(t, startDownload(t, task), "the overlapping Download to return")

	_, stream := remote.counts()
	if stream != 1 {
		t.Fatalf("stream calls = %d, want 1: overlapping Download started an attempt", stream)
	}

	task.Cancel()
	waitClosed(t, done, "Download to return after Cancel")
}

// A task whose file is already complete finishes without touching the network.
func TestTaskAlreadyDownloaded(t *testing.T) {
	remote := newFakeRemote()
	local := &fakeLocal{complete: true}
	task := NewTask("test", remote, local)

	waitClosed(t, startDownload(t, task), "Download to return")

	resolve, stream := remote.counts()
	if resolve != 0 || stream != 0 {
		t.Fatalf("complete task issued requests: resolve=%d stream=%d", resolve, stream)
	}
	if task.State != TaskCompleted {
		t.Fatalf("state = %v, want TaskCompleted", task.State)
	}
	if task.Error != nil {
		t.Fatalf("error = %v, want nil", task.Error)
	}
}

// Closing the connection while an attempt is parked keeps the attempt scope as
// the authority: the abort arrives through the context, not through the traffic
// control.
func TestTaskCloseConnectionUsesAttemptScope(t *testing.T) {
	remote := newFakeRemote()
	remote.blockStream = make(chan struct{})
	local := &fakeLocal{}
	task := NewTask("test", remote, local)

	if !task.run.tryBegin() {
		t.Fatal("tryBegin must succeed")
	}
	defer task.run.end()

	ctx, cancel := task.run.beginAttempt()
	defer task.run.endAttempt(cancel)

	task.CloseConnection()

	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		<-ctx.Done()
	}()

	select {
	case <-waitDone:
	case <-time.After(testTimeout):
		t.Fatal("CloseConnection did not abort the in-flight attempt")
	}

	if !errors.Is(context.Cause(ctx), ErrConnectionTimeoutClosed) {
		t.Fatalf("cause = %v, want ErrConnectionTimeoutClosed", context.Cause(ctx))
	}
}
