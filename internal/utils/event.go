package utils

import (
	"sync"
	"weak"

	"github.com/samber/lo"
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
			return s.send(payload)
		}

		return false
	})
}

type EventSubscriber[T any] struct {
	closed      bool
	closedMutex sync.RWMutex
	Channel     chan T
}

func (es *EventSubscriber[T]) Close() {
	es.closedMutex.Lock()
	defer es.closedMutex.Unlock()

	if !es.closed {
		close(es.Channel)
		es.closed = true
	}
}

func (es *EventSubscriber[T]) send(payload T) bool {
	es.closedMutex.RLock()
	defer es.closedMutex.RUnlock()

	if es.closed {
		return false
	}
	select {
	case es.Channel <- payload:
	default:
	}
	return true
}

func PipeEvent[In any, Out any](sub *EventSubscriber[In], mapFilter func(payload In) (Out, bool)) *EventSubscriber[Out] {
	channel := make(chan Out, 10)
	newSub := &EventSubscriber[Out]{
		Channel: channel,
	}

	go func() {
		defer sub.Close()
		for payload := range sub.Channel {
			if newPayload, ok := mapFilter(payload); ok {
				if !newSub.send(newPayload) {
					break
				}
			}
		}
	}()

	return newSub
}
