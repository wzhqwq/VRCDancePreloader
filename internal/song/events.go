package song

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type ChangeType string

const (
	StatusChange    ChangeType = "status"
	ProgressChange  ChangeType = "progress"
	TimeChange      ChangeType = "time"
	BasicInfoChange ChangeType = "basic"
)

func (ps *StatefulSong) SubscribeEvent(lazy bool) *utils.EventSubscriber[ChangeType] {
	if lazy {
		return ps.lazyEm.SubscribeEvent()
	} else {
		return ps.em.SubscribeEvent()
	}
}

func (ps *StatefulSong) notifySubscribers(changeType ChangeType) {
	ps.em.NotifySubscribers(changeType)
}
func (ps *StatefulSong) notifyLazySubscribers(changeType ChangeType) {
	ps.lazyEm.NotifySubscribers(changeType)
}

func (ps *StatefulSong) notifyStatusChange() {
	ps.notifySubscribers(StatusChange)
	ps.notifyLazySubscribers(StatusChange)
}

func (ps *StatefulSong) notifyInfoChange() {
	ps.notifySubscribers(BasicInfoChange)
	ps.notifyLazySubscribers(BasicInfoChange)
}

func (ps *StatefulSong) notifyTimeChange(routine bool) {
	ps.notifySubscribers(TimeChange)
	if !routine {
		ps.notifyLazySubscribers(TimeChange)
	}
}
