package song

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var idIncrement int64 = 0

func constructBasicStatefulSong() *StatefulSong {
	idIncrement++
	ret := &StatefulSong{
		sm: NewSongStateMachine(),

		ID: idIncrement,

		stopCh: make(chan struct{}),

		em:     utils.NewEventManager[ChangeType](),
		lazyEm: utils.NewEventManager[ChangeType](),
	}
	ret.sm.ps = ret

	return ret
}

func CreateStatefulSongByInternalId(id string) *StatefulSong {
	ret := constructBasicStatefulSong()

	provider := third_parties.GetProviderById(id)
	if provider != nil {
		ready := make(chan struct{})
		ret.sm.Go(func() {
			ret.completionLoop(provider, ready)
		})
		<-ready
	}

	return ret
}

func CreateUnknownSong() *StatefulSong {
	ret := constructBasicStatefulSong()

	ret.Unknown = true
	ret.sm.DownloadStatus = NotAvailable

	return ret
}
