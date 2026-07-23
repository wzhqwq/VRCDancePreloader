package utils

import (
	"sync"

	"github.com/samber/lo"
)

type eventSink[T any] interface {
	send(T) bool
}

type EventManager[T any] struct {
	sync.Mutex
	subscribers []eventSink[T]
}

func NewEventManager[T any]() *EventManager[T] {
	return &EventManager[T]{}
}

func (em *EventManager[T]) SubscribeEvent() *EventSubscriber[T] {
	em.Lock()
	defer em.Unlock()

	sub := &EventSubscriber[T]{
		Channel: make(chan T, 10),
	}
	em.subscribers = append(em.subscribers, sub)
	return sub
}

func (em *EventManager[T]) NotifySubscribers(payload T) {
	em.Lock()
	defer em.Unlock()
	em.subscribers = lo.Filter(em.subscribers, func(p eventSink[T], _ int) bool {
		return p.send(payload)
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

type mappedEventSink[In, Out any] struct {
	target    *EventSubscriber[Out]
	mapFilter func(In) (Out, bool)
}

func (s *mappedEventSink[In, Out]) send(payload In) bool {
	data, ok := s.mapFilter(payload)
	if !ok {
		return false
	}
	return s.target.send(data)
}

func PipeEvent[In, Out any](
	em *EventManager[In],
	mapFilter func(In) (Out, bool),
) *EventSubscriber[Out] {
	target := &EventSubscriber[Out]{
		Channel: make(chan Out, 10),
	}

	pipe := &mappedEventSink[In, Out]{
		target:    target,
		mapFilter: mapFilter,
	}
	em.subscribers = append(em.subscribers, pipe)

	return target
}

func PipeSubEvent[In any, Out any](sub *EventSubscriber[In], mapFilter func(payload In) (Out, bool)) *EventSubscriber[Out] {
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

func Pipe2Event[In1 any, In2 any, Out any](
	sub1 *EventSubscriber[In1],
	sub2 *EventSubscriber[In2],
	mapFilter1 func(payload In1) (Out, bool),
	mapFilter2 func(payload In2) (Out, bool),
) *EventSubscriber[Out] {
	channel := make(chan Out, 10)
	newSub := &EventSubscriber[Out]{
		Channel: channel,
	}

	go func() {
		defer sub1.Close()
		defer sub2.Close()

		for {
			select {
			case payload, ok := <-sub1.Channel:
				if !ok {
					return
				}
				if newPayload, ok := mapFilter1(payload); ok {
					if !newSub.send(newPayload) {
						break
					}
				}
			case payload, ok := <-sub2.Channel:
				if !ok {
					return
				}
				if newPayload, ok := mapFilter2(payload); ok {
					if !newSub.send(newPayload) {
						break
					}
				}
			}
		}
	}()

	return newSub
}
