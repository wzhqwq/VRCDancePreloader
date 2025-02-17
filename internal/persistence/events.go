package persistence

import (
	"github.com/samber/lo"
	"sync"
)

type FavoriteEventManager struct {
	sync.Mutex
	Subscribers []chan string
}

func NewFavoriteEventManager() *FavoriteEventManager {
	return &FavoriteEventManager{
		Subscribers: make([]chan string, 0),
	}
}

func (f *Favorites) SubscribeEvent() chan string {
	f.em.Lock()
	defer f.em.Unlock()
	channel := make(chan string, 10)
	f.em.Subscribers = append(f.em.Subscribers, channel)
	return channel
}

func (f *Favorites) notifySubscribers(id string) {
	f.em.Lock()
	defer f.em.Unlock()
	f.em.Subscribers = lo.Filter(f.em.Subscribers, func(c chan string, _ int) bool {
		select {
		case c <- id:
			return true
		default:
			return false
		}
	})
}
