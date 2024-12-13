package song

import "sync"

type SongChangeType string

const (
	StatusChange   SongChangeType = "status"
	ProgressChange SongChangeType = "progress"
	TimeChange     SongChangeType = "time"
)

type EventManager struct {
	sync.Mutex
	Subscribers []chan SongChangeType
}

func NewEventManager() *EventManager {
	return &EventManager{
		Subscribers: make([]chan SongChangeType, 0),
	}
}

func (ps *PreloadedSong) SubscribeEvent() chan SongChangeType {
	ps.em.Lock()
	defer ps.em.Unlock()
	channel := make(chan SongChangeType, 10)
	ps.em.Subscribers = append(ps.em.Subscribers, channel)
	return channel
}

func (ps *PreloadedSong) notifySubscribers(changeType SongChangeType) {
	ps.em.Lock()
	defer ps.em.Unlock()
	for _, sub := range ps.em.Subscribers {
		select {
		case sub <- changeType:
		default:
		}
	}
}
