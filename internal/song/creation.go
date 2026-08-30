package song

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var idIncrement int64 = 0

func constructBasicStatefulSong(id string) *StatefulSong {
	idIncrement++
	ret := &StatefulSong{
		ID: idIncrement,

		stopCh: make(chan struct{}),

		em:     utils.NewEventManager[ChangeType](),
		lazyEm: utils.NewEventManager[ChangeType](),
	}
	ret.sm = NewSongStateMachine(ret, id)

	return ret
}

func CreateStatefulSongByInternalId(id string) *StatefulSong {
	ret := constructBasicStatefulSong(id)
	ret.songId = id

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
	ret := constructBasicStatefulSong("unknown")

	ret.Unknown = true
	ret.sm.DownloadStatus = NotAvailable

	return ret
}
