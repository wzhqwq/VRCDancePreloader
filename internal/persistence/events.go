package persistence

func (f *LocalSongs) SubscribeEvent() chan string {
	return f.em.SubscribeEvent()
}

func (f *LocalSongs) notifySubscribers(id string) {
	f.em.NotifySubscribers(id)
}

func (a *AllowList) SubscribeEvent() chan string {
	return a.em.SubscribeEvent()
}

func (a *AllowList) notifySubscribers(id string) {
	a.em.NotifySubscribers(id)
}
