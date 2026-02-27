package persistence

import "github.com/wzhqwq/VRCDancePreloader/internal/utils"

func (f *LocalSongs) SubscribeEvent() *utils.EventSubscriber[string] {
	return f.em.SubscribeEvent()
}

func (f *LocalSongs) notifySubscribers(id string) {
	f.em.NotifySubscribers(id)
}
