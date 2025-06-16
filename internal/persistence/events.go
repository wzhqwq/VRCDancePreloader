package persistence

import "github.com/wzhqwq/VRCDancePreloader/internal/utils"

func (f *LocalSongs) SubscribeEvent() *utils.EventSubscriber[string] {
	return f.em.SubscribeEvent()
}

func (f *LocalSongs) notifySubscribers(id string) {
	f.em.NotifySubscribers(id)
}

func (a *AllowList) SubscribeEvent() *utils.EventSubscriber[string] {
	return a.em.SubscribeEvent()
}

func (a *AllowList) notifySubscribers(id string) {
	a.em.NotifySubscribers(id)
}
