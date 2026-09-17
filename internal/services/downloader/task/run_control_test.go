package task

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// T1 — the state transition table of runControl.
//
// runControl owns the re-entrancy guard, the permanent cancellation flag and
// the in-flight attempt scope. These tests pin the transitions that the
// download loop relies on; the assertions about "what the loop does" live in
// task_control_test.go.

func TestRunControlReentrancy(t *testing.T) {
	var r runControl

	if !r.tryBegin() {
		t.Fatal("first tryBegin must succeed")
	}
	if r.tryBegin() {
		t.Fatal("second tryBegin must fail while the loop is running")
	}

	r.end()

	if !r.tryBegin() {
		t.Fatal("tryBegin must succeed again after end")
	}
	r.end()
}

func TestRunControlCancelIsIrreversible(t *testing.T) {
	var r runControl

	if !r.tryBegin() {
		t.Fatal("tryBegin must succeed")
	}

	if !r.cancelAll() {
		t.Fatal("first cancelAll must report that it cancelled the task")
	}
	if r.cancelAll() {
		t.Fatal("second cancelAll must not report a new cancellation")
	}
	if !r.cancelled() {
		t.Fatal("cancelled must be true after cancelAll")
	}

	r.end()

	if r.tryBegin() {
		t.Fatal("tryBegin must fail after cancelAll, even after end")
	}
	if !r.cancelled() {
		t.Fatal("end must not clear the done phase")
	}
}

func TestRunControlCancelBeforeBegin(t *testing.T) {
	var r runControl

	r.cancelAll()

	ctx, cancel := r.beginAttempt()
	defer cancel(nil)

	if ctx.Err() == nil {
		t.Fatal("an attempt installed after cancelAll must already be cancelled")
	}
	if !errors.Is(context.Cause(ctx), ErrCanceled) {
		t.Fatalf("cause = %v, want ErrCanceled", context.Cause(ctx))
	}
}

func TestRunControlAbortAttempt(t *testing.T) {
	var r runControl

	// No attempt in flight: aborting is a no-op.
	r.abortAttempt(ErrRestarted)

	ctx, cancel := r.beginAttempt()

	r.abortAttempt(ErrConnectionTimeoutClosed)

	if !errors.Is(context.Cause(ctx), ErrConnectionTimeoutClosed) {
		t.Fatalf("cause = %v, want ErrConnectionTimeoutClosed", context.Cause(ctx))
	}

	r.endAttempt(cancel)

	// The scope was released, so a later abort must not reach the old context.
	r.abortAttempt(ErrRestarted)
	if !errors.Is(context.Cause(ctx), ErrConnectionTimeoutClosed) {
		t.Fatalf("cause changed after endAttempt: %v", context.Cause(ctx))
	}
}

func TestRunControlAbortAttemptNeverOverridesFirstCause(t *testing.T) {
	var r runControl

	ctx, cancel := r.beginAttempt()

	r.abortAttempt(ErrRestarted)
	r.abortAttempt(ErrConnectionTimeoutClosed)

	if !errors.Is(context.Cause(ctx), ErrRestarted) {
		t.Fatalf("cause = %v, want the first cause ErrRestarted", context.Cause(ctx))
	}

	r.endAttempt(cancel)
}

// TestRunControlCancelRacesBeginAttempt pins the invariant that made the
// previous implementation fragile by hand: whatever the interleaving, an
// attempt scope installed concurrently with Cancel() must end up cancelled.
// Without the re-check inside beginAttempt, a Cancel landing between the loop's
// cancelled check and the installation of the new scope would be lost, and the
// fresh attempt would run to completion after the task was cancelled.
func TestRunControlCancelRacesBeginAttempt(t *testing.T) {
	for i := 0; i < 200; i++ {
		var r runControl

		if !r.tryBegin() {
			t.Fatal("tryBegin must succeed")
		}

		start := make(chan struct{})
		var wg sync.WaitGroup

		var (
			ctx    context.Context
			cancel context.CancelCauseFunc
		)

		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			r.cancelAll()
		}()
		go func() {
			defer wg.Done()
			<-start
			ctx, cancel = r.beginAttempt()
		}()

		close(start)
		wg.Wait()

		if ctx == nil {
			t.Fatal("beginAttempt did not run")
		}
		if ctx.Err() == nil {
			t.Fatalf("iteration %d: the attempt scope survived a concurrent Cancel", i)
		}
		if !errors.Is(context.Cause(ctx), ErrCanceled) {
			t.Fatalf("iteration %d: cause = %v, want ErrCanceled", i, context.Cause(ctx))
		}

		cancel(nil)
		r.end()
	}
}

// TestRunControlAbortRacesEndAttempt documents the tolerated race: a Restart
// landing exactly as an attempt finishes may be dropped, which is correct
// because the loop is about to decide what to do next anyway. The only
// requirement is that nothing panics and no scope is left installed.
func TestRunControlAbortRacesEndAttempt(t *testing.T) {
	for i := 0; i < 200; i++ {
		var r runControl

		_, cancel := r.beginAttempt()

		start := make(chan struct{})
		var wg sync.WaitGroup

		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			r.abortAttempt(ErrRestarted)
		}()
		go func() {
			defer wg.Done()
			<-start
			r.endAttempt(cancel)
		}()

		close(start)
		wg.Wait()

		r.mu.Lock()
		leftover := r.cancel
		r.mu.Unlock()

		if leftover != nil {
			t.Fatalf("iteration %d: cancel function was left installed", i)
		}
	}
}

// TestRunControlTimeoutKeepsTaskAlive guards the single most important
// semantic: aborting an attempt (Restart / CloseConnection) must never put the
// task into the done phase, because the download loop has to be able to start
// a fresh attempt afterwards.
func TestRunControlTimeoutKeepsTaskAlive(t *testing.T) {
	var r runControl

	if !r.tryBegin() {
		t.Fatal("tryBegin must succeed")
	}

	ctx, cancel := r.beginAttempt()
	r.abortAttempt(ErrConnectionTimeoutClosed)
	r.endAttempt(cancel)

	if r.cancelled() {
		t.Fatal("aborting an attempt must not cancel the task")
	}
	if !errors.Is(context.Cause(ctx), ErrConnectionTimeoutClosed) {
		t.Fatalf("cause = %v, want ErrConnectionTimeoutClosed", context.Cause(ctx))
	}

	// A fresh attempt must be usable, i.e. its context is not cancelled.
	ctx2, cancel2 := r.beginAttempt()
	defer cancel2(nil)

	if err := ctx2.Err(); err != nil {
		t.Fatalf("fresh attempt context must be alive, got %v", err)
	}
}
