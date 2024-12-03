package song

import (
	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
)

type SongSMStatus string

const (
	// Initial is the initial state of every song
	Initial SongSMStatus = "initial"
	// Pending is when the song is waiting for the download to start,
	// either because previous songs are still downloading
	// or it's queue-jumped by a higher priority song
	Pending SongSMStatus = "pending"
	// Requesting is when the song is requesting the download
	Requesting SongSMStatus = "requesting"
	// Downloading is when the song is downloading
	Downloading SongSMStatus = "downloading"
	// Downloaded is when the song is downloaded to the disk
	Downloaded SongSMStatus = "downloaded"
	// Failed is when the song failed to download, will be retried
	Failed SongSMStatus = "failed"
	// Removed is when the song is removed from the playlist
	Removed SongSMStatus = "removed"
)

// SongStateMachine is the state machine for a song
type SongStateMachine struct {
	Status SongSMStatus

	PreloadedSong *PreloadedSong

	// waiter
	songWaiting utils.FinishingBroadcaster
}

func (sm *SongStateMachine) IsDownloadLoopStarted() bool {
	return sm.Status == Pending || sm.Status == Requesting || sm.Status == Downloading
}

func (sm *SongStateMachine) WaitForCompleteSong() error {
	if sm.Status == Downloaded {
		return nil
	}
	if sm.IsDownloadLoopStarted() {
		sm.songWaiting.WaitForFinishing()
		return nil
	}
	return sm.StartDownloadLoop()
}

func (sm *SongStateMachine) StartDownloadLoop() error {
	ds := cache.Download(sm.PreloadedSong.GetId(), sm.PreloadedSong.GetDownloadUrl())

	for {
		select {
		case <-ds.StateCh:
			if ds.Done {
				sm.Status = Downloaded
				// TODO: Status update callback
				return nil
			}
			if ds.Error != nil {
				sm.Status = Failed
				// TODO: Status update callback
				return ds.Error
			}
			if ds.Pending && sm.Status != Pending {
				sm.Status = Pending
				// TODO: Status update callback
				continue
			}
			if ds.TotalSize == 0 && sm.Status != Requesting {
				sm.Status = Requesting
				// TODO: Status update callback
				continue
			}
			// Otherwise, it's downloading
			if sm.Status != Downloading {
				sm.Status = Downloading
				// TODO: Status update callback
			}
			sm.PreloadedSong.TotalSize = ds.TotalSize
			sm.PreloadedSong.DownloadedSize = ds.DownloadedSize
			// TODO: Download progress update callback
		}
	}
}

func (sm *SongStateMachine) StartPlayingLoop() {

}
