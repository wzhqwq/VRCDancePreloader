package song

import (
	"fmt"
	"math"
	"time"

	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
)

type SongDownloadStatus string

const (
	// Initial is the initial state of every song
	Initial SongDownloadStatus = "initial"
	// Pending is when the song is waiting for the download to start,
	// either because previous songs are still downloading
	// or it's queue-jumped by a higher priority song
	Pending SongDownloadStatus = "pending"
	// Requesting is when the song is requesting the download
	Requesting SongDownloadStatus = "requesting"
	// Downloading is when the song is downloading
	Downloading SongDownloadStatus = "downloading"
	// Downloaded is when the song is downloaded to the disk
	Downloaded SongDownloadStatus = "downloaded"
	// Failed is when the song failed to download, will be retried
	Failed SongDownloadStatus = "failed"
	// Removed is when the song is removed from the playlist
	Removed SongDownloadStatus = "removed"
)

type SongPlayStatus string

const (
	Queued  SongPlayStatus = "queued"
	Playing SongPlayStatus = "playing"
	Ended   SongPlayStatus = "ended"
)

// SongStateMachine is the state machine for a song
type SongStateMachine struct {
	DownloadStatus SongDownloadStatus
	PlayStatus     SongPlayStatus

	PreloadedSong *PreloadedSong

	// waiter
	songWaiting utils.FinishingBroadcaster
}

func (sm *SongStateMachine) IsDownloadLoopStarted() bool {
	return sm.DownloadStatus == Pending || sm.DownloadStatus == Requesting || sm.DownloadStatus == Downloading
}
func (sm *SongStateMachine) IsPlayingLoopStarted() bool {
	return sm.PlayStatus == Playing
}

func (sm *SongStateMachine) WaitForCompleteSong() error {
	if sm.DownloadStatus == Downloaded {
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
				sm.DownloadStatus = Downloaded
				sm.PreloadedSong.notifySubscribers(StatusChange)
				return nil
			}
			if ds.Error != nil {
				sm.DownloadStatus = Failed
				sm.PreloadedSong.notifySubscribers(StatusChange)
				return ds.Error
			}
			if ds.Pending && sm.DownloadStatus != Pending {
				sm.DownloadStatus = Pending
				sm.PreloadedSong.notifySubscribers(StatusChange)
				continue
			}
			if ds.TotalSize == 0 && sm.DownloadStatus != Requesting {
				sm.DownloadStatus = Requesting
				sm.PreloadedSong.notifySubscribers(StatusChange)
				continue
			}
			// Otherwise, it's downloading
			if sm.DownloadStatus == Removed {
				cache.CancelDownload(sm.PreloadedSong.GetId())
				return fmt.Errorf("download removed")
			}
			if sm.DownloadStatus != Downloading {
				sm.DownloadStatus = Downloading
				sm.PreloadedSong.notifySubscribers(StatusChange)
			}
			sm.PreloadedSong.TotalSize = ds.TotalSize
			sm.PreloadedSong.DownloadedSize = ds.DownloadedSize
			sm.PreloadedSong.notifySubscribers(ProgressChange)
		}
	}
}

func (sm *SongStateMachine) PlaySongStartFrom(offset float64) {
	if sm.PlayStatus == Ended {
		return
	}
	sm.PreloadedSong.TimePassed = offset
	if sm.PlayStatus == Queued {
		go sm.StartPlayingLoop()
	} else {
		sm.PreloadedSong.notifySubscribers(TimeChange)
	}
}

func (sm *SongStateMachine) StartPlayingLoop() {
	sm.PlayStatus = Playing
	sm.PreloadedSong.notifySubscribers(TimeChange)
	for {
		if sm.PlayStatus != Playing {
			break
		}
		nextTime := math.Floor(sm.PreloadedSong.TimePassed+0.1) + 1.0
		<-time.After(time.Duration(nextTime-sm.PreloadedSong.TimePassed) * time.Second)

		if nextTime >= sm.PreloadedSong.Duration {
			sm.PlayStatus = Ended
			break
		} else {
			sm.PreloadedSong.TimePassed = nextTime
			sm.PreloadedSong.notifySubscribers(TimeChange)
		}
	}
	sm.PreloadedSong.notifySubscribers(TimeChange)
}

func (sm *SongStateMachine) RemoveFromList() {
	sm.DownloadStatus = Removed
	if sm.PlayStatus == Playing {
		sm.PlayStatus = Ended
	}
	sm.PreloadedSong.notifySubscribers(StatusChange)
}
