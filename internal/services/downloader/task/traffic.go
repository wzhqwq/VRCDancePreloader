package task

import (
	"context"
	"errors"
	"sync"
	"time"
)

const hangingConnectionTimeout = time.Second * 30

// TrafficControl is the injection point for "may this task download right now".
//
// It has two implementations, and both are load bearing:
//
//   - traffic (in the downloader package) is the queue controlled policy: the
//     manager publishes each task's queue position and whether the position is
//     within the parallel limit;
//   - nopTrafficControl is the no queue control policy used by NewTask, so that
//     other modules can reuse the download engine without a queue.
type TrafficControl interface {
	// WaitPending blocks until this task is permitted to download. A non-nil
	// error is the queue's veto, including cancellation.
	//
	// beforeWait is invoked once per wait episode with the estimated resume
	// time (zero value when unknown); the caller uses it to arm the
	// hanging-connection watchdog.
	//
	// The permission is level triggered: every call re-reads the current state,
	// so a task that re-enters the gate after a Restart or a connection timeout
	// is re-evaluated instead of being assumed to be still permitted.
	WaitPending(beforeWait func(resumeAt time.Time)) error

	// NotifyPermit publishes the current queue position and whether this task
	// may download. It is called by whoever controls the queue; implementations
	// without queue control ignore it and always allow.
	NotifyPermit(position int, allowed bool)

	// Cancel is called exactly once, when the task is cancelled for good. It is
	// only needed by an implementation whose wait has to be interrupted;
	// cancellation itself is delivered through the attempt context, which is
	// what actually stops the next request.
	Cancel()
}

// nopTrafficControl is the "no queue control" policy: there is no queue, so
// there is nothing to wait for and nothing to veto. It is deliberately
// stateless.
//
// In particular Cancel does not have to be remembered here. Cancelling a task
// closes its attempt context (Task.Cancel), and every request the download loop
// makes is bound to that context, so a cancelled task stops on its own — see
// the cancellation tests in task_control_test.go. Keeping a flag would only
// duplicate that, and would have to be synchronised to be race free.
type nopTrafficControl struct{}

// Cancel is a no-op: this implementation never blocks, so there is no wait to
// interrupt. See the type comment.
func (n *nopTrafficControl) Cancel() {}

var _ TrafficControl = (*nopTrafficControl)(nil)

// WaitPending returns immediately: without a queue the task is always permitted.
func (n *nopTrafficControl) WaitPending(_ func(eta time.Time)) error {
	return nil
}

// NotifyPermit is a no-op: without a queue there is no permit to publish, and
// the task is always allowed to download.
func (n *nopTrafficControl) NotifyPermit(int, bool) {}

func (t *Task) waitPending(connected bool) error {
	// scooped timeout
	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), hangingConnectionTimeout)
	defer cancel()

	waited := false
	lastState := t.State

	err := t.Traffic.WaitPending(func(eta time.Time) {
		waited = true
		t.setState(TaskPending)

		t.resetEta()

		if connected {
			if !eta.IsZero() && eta.Sub(time.Now()) > hangingConnectionTimeout {
				// close download stream if it won't resume in 30s
				logger.InfoLn(t.ID, "is closed because it won't resume in 30s")
				t.CloseConnection()
			} else {
				// close download stream after 30s
				wg.Go(func() {
					<-ctx.Done()
					if errors.Is(ctx.Err(), context.DeadlineExceeded) {
						logger.InfoLn(t.ID, "is closed because of 30s timeout")
						t.CloseConnection()
					}
				})
			}
		}
	})
	if err != nil {
		return err
	}

	if waited {
		t.setState(lastState)
	}

	return nil
}
