package task

import (
	"context"
	"errors"
	"sync"
	"time"
)

const hangingConnectionTimeout = time.Second * 30

type TrafficControl interface {
	WaitPending(beforeWait func(eta time.Time)) error
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
