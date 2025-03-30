package utils

import (
	"github.com/samber/lo"
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

type StringEventManager struct {
	sync.Mutex
	subscribers []chan string
}

func NewStringEventManager() *StringEventManager {
	return &StringEventManager{
		subscribers: make([]chan string, 0),
	}
}

func (em *StringEventManager) SubscribeEvent() chan string {
	em.Lock()
	defer em.Unlock()
	channel := make(chan string, 10)
	em.subscribers = append(em.subscribers, channel)
	return channel
}

func (em *StringEventManager) NotifySubscribers(payload string) {
	em.Lock()
	defer em.Unlock()
	em.subscribers = lo.Filter(em.subscribers, func(c chan string, _ int) bool {
		select {
		case c <- payload:
			return true
		default:
			return false
		}
	})
}
