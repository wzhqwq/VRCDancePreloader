package utils

import (
	"github.com/samber/lo"
	"sync"
	"weak"
)

type EventManager[T any] struct {
	sync.Mutex
	weakSubscribers []weak.Pointer[EventSubscriber[T]]
}

func NewEventManager[T any]() *EventManager[T] {
	return &EventManager[T]{
		weakSubscribers: []weak.Pointer[EventSubscriber[T]]{},
	}
}

func (em *EventManager[T]) SubscribeEvent() *EventSubscriber[T] {
	em.Lock()
	defer em.Unlock()

	channel := make(chan T, 10)
	sub := &EventSubscriber[T]{
		Channel: channel,
	}
	em.weakSubscribers = append(em.weakSubscribers, weak.Make(sub))
	return sub
}

func (em *EventManager[T]) NotifySubscribers(payload T) {
	em.Lock()
	defer em.Unlock()
	em.weakSubscribers = lo.Filter(em.weakSubscribers, func(p weak.Pointer[EventSubscriber[T]], _ int) bool {
		if s := p.Value(); s != nil {
			if s.closed {
				return false
			} else {
				s.Channel <- payload
				return true
			}
		} else {
			return false
		}
	})
}

type EventSubscriber[T any] struct {
	closed  bool
	Channel chan T
}

func (es *EventSubscriber[T]) Close() {
	if es.closed {
		return
	}
	es.closed = true
	close(es.Channel)
}
