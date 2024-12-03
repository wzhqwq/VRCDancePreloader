package utils

import (
	"sync"
)

type SingleWaiter struct {
	finishCh chan struct{}
}

type FinishingBroadcaster struct {
	sync.Mutex
	waiters  []*SingleWaiter
	finished bool
}

func NewBroadcaster() *FinishingBroadcaster {
	return &FinishingBroadcaster{
		waiters: make([]*SingleWaiter, 0),
	}
}

func (fb *FinishingBroadcaster) WaitForFinishing() {
	fb.Lock()
	if fb.finished {
		fb.Unlock()
		return
	}

	w := &SingleWaiter{
		finishCh: make(chan struct{}, 10),
	}
	fb.waiters = append(fb.waiters, w)
	fb.Unlock()

	<-w.finishCh
}

func (fb *FinishingBroadcaster) Finish() {
	fb.Lock()
	defer fb.Unlock()

	for _, w := range fb.waiters {
		w.finishCh <- struct{}{}
		close(w.finishCh)
	}
	fb.waiters = make([]*SingleWaiter, 0)
	fb.finished = true
}
