package task

import (
	"context"
	"errors"
	"time"
)

const hangingConnectionTimeout = time.Second * 30

type TrafficControl interface {
	WaitPending(beforeWait func(eta time.Time)) error
	WaitScheduled(beforeWait func()) error
	ScheduledTime() time.Time
	Cancel()
}

type nopTrafficControl struct {
	canceled bool
}

func (n *nopTrafficControl) Cancel() {
	n.canceled = true
}

var _ TrafficControl = (*nopTrafficControl)(nil)

func (n *nopTrafficControl) WaitPending(_ func(eta time.Time)) error {
	if n.canceled {
		return ErrCanceled
	}
	return nil
}

func (n *nopTrafficControl) WaitScheduled(_ func()) error {
	if n.canceled {
		return ErrCanceled
	}
	return nil
}

func (n *nopTrafficControl) ScheduledTime() time.Time {
	return time.Time{}
}

func (t *Task) waitPending() error {
	// scooped timeout
	ctx, cancel := context.WithTimeout(context.Background(), hangingConnectionTimeout)
	defer cancel()

	waited := false
	lastState := t.State

	err := t.Traffic.WaitPending(func(eta time.Time) {
		waited = true
		t.setState(TaskPending)

		t.resetEta()

		if t.connected {
			if !eta.IsZero() && eta.Sub(time.Now()) > hangingConnectionTimeout {
				// close download stream if it won't resume in 30s
				t.Restart()
			} else {
				// close download stream after 30s
				go func() {
					<-ctx.Done()
					if errors.Is(ctx.Err(), context.DeadlineExceeded) {
						t.Restart()
					}
				}()
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

func (t *Task) waitSchedule() error {
	waited := false
	lastState := t.State

	err := t.Traffic.WaitScheduled(func() {
		waited = true
		t.setState(TaskWaitScheduled)
	})
	if err != nil {
		return err
	}

	if waited {
		t.setState(lastState)
	}

	return nil
}
