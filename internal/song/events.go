package song

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type ChangeType string

const (
	StatusChange   ChangeType = "status"
	ProgressChange ChangeType = "progress"
	TimeChange     ChangeType = "time"
)

func (ps *PreloadedSong) SubscribeEvent() *utils.EventSubscriber[ChangeType] {
	return ps.em.SubscribeEvent()
}

func (ps *PreloadedSong) notifySubscribers(changeType ChangeType) {
	ps.em.NotifySubscribers(changeType)
}
