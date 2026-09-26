package song

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

//var completionLogger = utils.NewLogger("Info Completion")

func (ps *StatefulSong) CompleteInfoIfEmpty(title, group string, duration int) {
	provider := third_parties.GetProviderById(ps.songId)
	if provider != nil {
		provider.ModifyInfoPlaceholder(ps.songId, func(i types.GeneralVideoInfo) types.GeneralVideoInfo {
			//completionLogger.DebugLn("Modify placeholder info of", ps.songId, "using queue info:", title, group, duration)
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

	// published is the snapshot whose Data ps.info already holds. The entry
	// notifies on every attempt and on every phase change, and most of those
	// snapshots carry exactly the info the song already has — publishing each of
	// them is what looked like duplicated output (review/09 §2.4).
	published := snap

	for {
		if infoSnapshotMoved(published, snap) {
			published = snap
			ps.info = snap.Data
			// The instance id is what tells two live StatefulSong objects of the
			// same song apart: they share the provider entry, so they log the same
			// info from their own completionLoop. See review/09 §8.4.
			//completionLogger.DebugLn("Latest info of", ps.songId, "(instance", ps.ID, ") is", ps.info.Title, ps.info.GroupName, ps.info.Duration, "@", snap.Status.Phase)
			ps.notifyInfoChange()
		}

		if snap.Status.Valid() {
			//completionLogger.DebugLn("Info of", ps.songId, "is seen as valid")
			persistence.UpdateSavedTitle(ps.songId, snap.Data.Title)
			return
		}

		select {
		case <-ch.Channel:
			// A notification is only a wake-up, never the value: RemoteManager
			// reads its snapshot under the entry lock and notifies *after*
			// releasing it (interactive/remote.go:329-332), so a snapshot read
			// before a newer one can be delivered after it. Trusting the payload
			// is what made the published info go 3m40s -> 0s -> 3m40s for the same
			// instance (review/09 §8.5); the current state is what gets published.
			snap = handle.Snapshot()
			continue
		case <-ps.stopCh:
			return
		}
	}
}

// infoSnapshotMoved reports whether a snapshot carries info the song does not
// have yet.
//
// Only the data counts. The status bookkeeping is deliberately not part of the
// comparison (neither Err/RetryAfter/RetryAttempts/CooldownUntil, which change on
// every attempt, nor Phase): ps.info does not carry the status, so a phase change
// alone is not news for the song, and counting it would keep this loop as loud as
// it was before — the three seemingly identical lines per song were one entry
// notify per phase, not three loops (review/09 §8.2). Failures are reported by
// the RemoteManager's own log, and "the info became usable" is reported by the
// Valid branch above.
//
// The loop stays subscribed either way: a snapshot can still improve later (a
// queue item completing the placeholder, a fetch that finally succeeds).
func infoSnapshotMoved(previous, current interactive.RemoteSnapshot[types.GeneralVideoInfo]) bool {
	return previous.Data != current.Data ||
		previous.HasData != current.HasData
}
