package song

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
)

func (ps *StatefulSong) CompleteInfoIfEmpty(title, group string, duration int) {
	provider := third_parties.GetProviderById(ps.songId)
	if provider != nil {
		provider.ModifyInfoPlaceholder(ps.songId, func(i types.GeneralVideoInfo) types.GeneralVideoInfo {
			return i.CompleteIfEmpty(title, group, duration)
		})
	}
}

func (ps *StatefulSong) completionLoop(provider third_parties.ResourceProvider, infoReady chan<- struct{}) {
	handle := provider.Info(ps.songId)
	defer handle.Release()

	snap := handle.Snapshot()
	ps.info = snap.Data
	close(infoReady)

	if snap.Status.Valid() {
		return
	}

	ch := handle.Subscribe()
	defer ch.Close()

	snap = handle.Snapshot()

	for {
		ps.info = snap.Data
		ps.notifyInfoChange()
		if snap.Status.Valid() {
			persistence.UpdateSavedTitle(ps.songId, snap.Data.Title)
			return
		}

		select {
		case snap = <-ch.Channel:
			continue
		case <-ps.stopCh:
			return
		}
	}
}
