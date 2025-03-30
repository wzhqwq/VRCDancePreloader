package persistence

func (f *Favorites) SubscribeEvent() chan string {
	return f.em.SubscribeEvent()
}

func (f *Favorites) notifySubscribers(id string) {
	f.em.NotifySubscribers(id)
}

func (a *AllowList) SubscribeEvent() chan string {
	return a.em.SubscribeEvent()
}

func (a *AllowList) notifySubscribers(id string) {
	a.em.NotifySubscribers(id)
}
