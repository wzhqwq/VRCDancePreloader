package utils

import (
	"github.com/samber/lo"
	"sync"
)

type StringEventManager struct {
	sync.Mutex
	subscribers []*StringEventSubscriber
}

func NewStringEventManager() *StringEventManager {
	return &StringEventManager{
		subscribers: make([]*StringEventSubscriber, 0),
	}
}

func (em *StringEventManager) SubscribeEvent() *StringEventSubscriber {
	em.Lock()
	defer em.Unlock()
	channel := make(chan string, 10)
	sub := &StringEventSubscriber{
		closed:  false,
		Channel: channel,
	}
	em.subscribers = append(em.subscribers, sub)
	return sub
}

func (em *StringEventManager) NotifySubscribers(payload string) {
	em.Lock()
	defer em.Unlock()
	em.subscribers = lo.Filter(em.subscribers, func(c *StringEventSubscriber, _ int) bool {
		if c.closed {
			return false
		} else {
			c.Channel <- payload
			return true
		}
	})
}

type StringEventSubscriber struct {
	closed  bool
	Channel chan string
}

func (es *StringEventSubscriber) Close() {
	if es.closed {
		return
	}
	es.closed = true
	close(es.Channel)
}
