package task

import (
	"context"
	"sync"
)

// runPhase is the lifecycle phase of one Task's download loop.
type runPhase uint8

const (
	// runIdle means no Download() is executing.
	runIdle runPhase = iota
	// runActive means exactly one Download() is executing.
	runActive
	// runDone means Cancel() was called. It is irreversible: no further
	// attempt may start.
	runDone
)

// runControl owns the whole cancellation and re-entrancy state of a Task:
//
//   - the re-entrancy guard (only one Download() at a time),
//   - the permanent cancellation flag (Cancel is irreversible),
//   - the cancellation scope of the attempt that is currently in flight.
//
// Restart and CloseConnection only abort the attempt that is currently in
// flight; the download loop is expected to install a fresh scope with
// beginAttempt and try again. A single task wide context cannot express that:
// once canceled it stays canceled, so every following attempt would fail
// immediately without doing any work (and the retry loop would spin at full
// speed).
//
// Keeping the three concerns in one struct under one mutex makes the
// transitions explicit. The previous implementation spread them over two
// atomics plus a separately locked func pointer, which left "Cancel racing
// with the installation of a new attempt" as a real hazard that had to be
// re-checked by hand.
type runControl struct {
	mu     sync.Mutex
	phase  runPhase
	cancel context.CancelCauseFunc
}

// tryBegin claims the download loop for the caller. It reports false when
// another Download() is already running, or when the task has been canceled.
func (r *runControl) tryBegin() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.phase != runIdle {
		return false
	}

	r.phase = runActive
	return true
}

// end releases the loop claimed by tryBegin. A canceled task stays done.
func (r *runControl) end() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.phase == runActive {
		r.phase = runIdle
	}
}

// canceled reports whether Cancel has been called.
func (r *runControl) cancelled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.phase == runDone
}

// beginAttempt installs a fresh cancellation scope for one attempt and returns
// it together with its cancel function. The caller must pass the returned
// cancel function to endAttempt.
//
// The cancellation flag is re-checked while the mutex is held: without that, a
// Cancel landing between the loop's check and the installation of the new
// cancel function would find the previous (already finished) attempt's cancel
// function, and the fresh attempt would run to completion before the loop
// noticed the cancellation.
func (r *runControl) beginAttempt() (context.Context, context.CancelCauseFunc) {
	ctx, cancel := context.WithCancelCause(context.Background())

	r.mu.Lock()
	r.cancel = cancel
	alreadyCancelled := r.phase == runDone
	r.mu.Unlock()

	if alreadyCancelled {
		cancel(ErrCanceled)
	}

	return ctx, cancel
}

// endAttempt releases the scope installed by beginAttempt.
//
// The cancel function is cleared first: a Restart or CloseConnection racing
// with the end of an attempt then observes a nil cancel function and does
// nothing, which is correct because the attempt it wanted to abort has already
// finished (and the loop is about to decide what to do next anyway).
func (r *runControl) endAttempt(cancel context.CancelCauseFunc) {
	r.mu.Lock()
	r.cancel = nil
	r.mu.Unlock()

	cancel(nil)
}

// abortAttempt cancels the attempt that is currently in flight, if any. The
// task itself keeps running: the download loop is expected to retry.
func (r *runControl) abortAttempt(cause error) {
	r.mu.Lock()
	cancel := r.cancel
	r.mu.Unlock()

	if cancel != nil {
		cancel(cause)
	}
}

// cancelAll makes the cancellation permanent and aborts the attempt that is
// currently in flight. It reports whether this call was the one that canceled
// the task, so that callers can run one-shot side effects exactly once.
func (r *runControl) cancelAll() bool {
	r.mu.Lock()

	if r.phase == runDone {
		r.mu.Unlock()
		return false
	}

	r.phase = runDone
	cancel := r.cancel
	r.mu.Unlock()

	if cancel != nil {
		cancel(ErrCanceled)
	}

	return true
}
