package song

import "sync"

type ChangeType string

const (
	StatusChange   ChangeType = "status"
	ProgressChange ChangeType = "progress"
	TimeChange     ChangeType = "time"
)

type EventManager struct {
	sync.Mutex
	Subscribers []chan ChangeType
}

func NewEventManager() *EventManager {
	return &EventManager{
		Subscribers: make([]chan ChangeType, 0),
	}
}

func (ps *PreloadedSong) SubscribeEvent() chan ChangeType {
	ps.em.Lock()
	defer ps.em.Unlock()
	channel := make(chan ChangeType, 10)
	ps.em.Subscribers = append(ps.em.Subscribers, channel)
	return channel
}

func (ps *PreloadedSong) notifySubscribers(changeType ChangeType) {
	ps.em.Lock()
	defer ps.em.Unlock()
	for _, sub := range ps.em.Subscribers {
		select {
		case sub <- changeType:
		default:
		}
	}
}
