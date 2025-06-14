package playlist

import (
	"sync"
)

var newListSubscribers []chan *PlayList

func SubscribeNewListEvent() chan *PlayList {
	channel := make(chan *PlayList, 10)
	newListSubscribers = append(newListSubscribers, channel)
	if currentPlaylist != nil {
		channel <- currentPlaylist
	}
	return channel
}
func notifyNewList(pl *PlayList) {
	for _, sub := range newListSubscribers {
		select {
		case sub <- pl:
		default:
			<-sub
			sub <- pl
		}
	}
}

type ChangeType string

const (
	ItemsChange ChangeType = "items"
	RoomChange  ChangeType = "room"
	Stopped     ChangeType = "stopped"
)

type EventManager struct {
	sync.Mutex
	ChangeSubscribers []chan ChangeType
}

func NewEventManager() *EventManager {
	return &EventManager{
		ChangeSubscribers: []chan ChangeType{},
	}
}

func (pl *PlayList) SubscribeChangeEvent() chan ChangeType {
	pl.em.Lock()
	defer pl.em.Unlock()
	channel := make(chan ChangeType, 10)
	pl.em.ChangeSubscribers = append(pl.em.ChangeSubscribers, channel)
	return channel
}

func (pl *PlayList) notifyChange(changeType ChangeType) {
	pl.em.Lock()
	defer pl.em.Unlock()
	for _, sub := range pl.em.ChangeSubscribers {
		select {
		case sub <- changeType:
		default:
		}
	}
}
