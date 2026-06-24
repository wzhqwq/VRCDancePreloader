package playlist

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var newListEm = utils.NewEventManager[*PlayList]()

func SubscribeNewListEvent() *utils.EventSubscriber[*PlayList] {
	return newListEm.SubscribeEvent()
}
func notifyNewList(pl *PlayList) {
	newListEm.NotifySubscribers(pl)
}

type ChangeType string

const (
	ItemsChange ChangeType = "items"
	RoomChange  ChangeType = "room"
	Stopped     ChangeType = "stopped"
)

func (pl *PlayList) SubscribeChangeEvent() *utils.EventSubscriber[ChangeType] {
	return pl.em.SubscribeEvent()
}

func (pl *PlayList) notifyChange(changeType ChangeType) {
	pl.em.NotifySubscribers(changeType)
}

func (pl *PlayList) SubscribeActiveSongEvent() *utils.EventSubscriber[*song.PreloadedSong] {
	return pl.activeEm.SubscribeEvent()
}

func (pl *PlayList) notifyActiveSongChange(song *song.PreloadedSong) {
	pl.activeEm.NotifySubscribers(song)
}
