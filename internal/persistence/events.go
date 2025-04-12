package persistence

import "github.com/wzhqwq/VRCDancePreloader/internal/utils"

func (f *LocalSongs) SubscribeEvent() *utils.StringEventSubscriber {
	return f.em.SubscribeEvent()
}

func (f *LocalSongs) notifySubscribers(id string) {
	f.em.NotifySubscribers(id)
}

func (a *AllowList) SubscribeEvent() *utils.StringEventSubscriber {
	return a.em.SubscribeEvent()
}

func (a *AllowList) notifySubscribers(id string) {
	a.em.NotifySubscribers(id)
}
