package playlist

import (
	"sync"

	"github.com/wzhqwq/PyPyDancePreloader/internal/song"
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
)

type EventManager struct {
	sync.Mutex
	ChangeSubscribers  []chan ChangeType
	NewItemSubscribers []chan *song.PreloadedSong
}

func NewEventManager() *EventManager {
	return &EventManager{
		ChangeSubscribers:  []chan ChangeType{},
		NewItemSubscribers: []chan *song.PreloadedSong{},
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

func (pl *PlayList) SubscribeNewItemEvent() chan *song.PreloadedSong {
	pl.em.Lock()
	defer pl.em.Unlock()
	channel := make(chan *song.PreloadedSong, 10)
	pl.em.NewItemSubscribers = append(pl.em.NewItemSubscribers, channel)
	return channel
}

func (pl *PlayList) notifyNewItem(item *song.PreloadedSong) {
	pl.em.Lock()
	defer pl.em.Unlock()
	for _, sub := range pl.em.NewItemSubscribers {
		select {
		case sub <- item:
		default:
		}
	}
}
