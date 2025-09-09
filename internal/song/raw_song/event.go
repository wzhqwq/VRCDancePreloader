package raw_song

import "github.com/wzhqwq/VRCDancePreloader/internal/utils"

var em = utils.NewEventManager[string]()

func SubscribeSongListChange() *utils.EventSubscriber[string] {
	return em.SubscribeEvent()
}
