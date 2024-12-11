package playlist

import (
	"sync"

	"github.com/wzhqwq/PyPyDancePreloader/internal/song"
)

type PlayListChangeType string

const (
	ItemsChange PlayListChangeType = "items"
)

type EventManager struct {
	sync.Mutex
	ChangeSubscribers  []chan PlayListChangeType
	NewItemSubscribers []chan *song.PreloadedSong
}

func (ps *PlayList) SubscribeChangeEvent() chan PlayListChangeType {
	ps.em.Lock()
	defer ps.em.Unlock()
	channel := make(chan PlayListChangeType, 10)
	ps.em.ChangeSubscribers = append(ps.em.ChangeSubscribers, channel)
	return channel
}

func (ps *PlayList) notifyChange(changeType PlayListChangeType) {
	ps.em.Lock()
	defer ps.em.Unlock()
	for _, sub := range ps.em.ChangeSubscribers {
		select {
		case sub <- changeType:
		default:
		}
	}
}

func (ps *PlayList) SubscribeNewItemEvent() chan *song.PreloadedSong {
	ps.em.Lock()
	defer ps.em.Unlock()
	channel := make(chan *song.PreloadedSong, 10)
	ps.em.NewItemSubscribers = append(ps.em.NewItemSubscribers, channel)
	return channel
}

func (ps *PlayList) notifyNewItem(item *song.PreloadedSong) {
	ps.em.Lock()
	defer ps.em.Unlock()
	for _, sub := range ps.em.NewItemSubscribers {
		select {
		case sub <- item:
		default:
		}
	}
}
