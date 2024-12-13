package song

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
)

type DownloadStatus string

const (
	// Initial is the initial state of every song
	Initial DownloadStatus = "initial"
	// Pending is when the song is waiting for the download to start,
	// either because previous songs are still downloading
	// or it's queue-jumped by a higher priority song
	Pending DownloadStatus = "pending"
	// Requesting is when the song is requesting the download
	Requesting DownloadStatus = "requesting"
	// Downloading is when the song is downloading
	Downloading DownloadStatus = "downloading"
	// Downloaded is when the song is downloaded to the disk
	Downloaded DownloadStatus = "downloaded"
	// Failed is when the song failed to download, will be retried
	Failed DownloadStatus = "failed"
	// Removed is when the song is removed from the playlist
	Removed DownloadStatus = "removed"
)

type PlayStatus string

const (
	Queued  PlayStatus = "queued"
	Playing PlayStatus = "playing"
	Ended   PlayStatus = "ended"
)

// StateMachine is the state machine for a song
type StateMachine struct {
	DownloadStatus DownloadStatus
	PlayStatus     PlayStatus

	PreloadedSong *PreloadedSong

	// waiter
	songWaiting utils.FinishingBroadcaster

	// locks
	timeMutex sync.Mutex
}

func NewSongStateMachine() *StateMachine {
	return &StateMachine{
		DownloadStatus: Initial,
		PlayStatus:     Queued,
		PreloadedSong:  nil,
		songWaiting:    utils.FinishingBroadcaster{},
		timeMutex:      sync.Mutex{},
	}
}

func (sm *StateMachine) IsDownloadLoopStarted() bool {
	return sm.DownloadStatus == Pending || sm.DownloadStatus == Requesting || sm.DownloadStatus == Downloading
}
func (sm *StateMachine) IsPlayingLoopStarted() bool {
	return sm.PlayStatus == Playing
}

func (sm *StateMachine) WaitForCompleteSong() error {
	if sm.DownloadStatus == Downloaded {
		return nil
	}
	if sm.IsDownloadLoopStarted() {
		sm.songWaiting.WaitForFinishing()
		return nil
	}
	return sm.StartDownloadLoop()
}

func (sm *StateMachine) StartDownloadLoop() error {
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

func (sm *StateMachine) PlaySongStartFrom(offset float64) {
	if sm.PlayStatus == Ended {
		return
	}

	sm.timeMutex.Lock()
	sm.PreloadedSong.TimePassed = offset
	sm.timeMutex.Unlock()

	if sm.PlayStatus == Queued {
		go sm.StartPlayingLoop()
	} else {
		sm.PreloadedSong.notifySubscribers(TimeChange)
	}
}

func (sm *StateMachine) StartPlayingLoop() {
	sm.PlayStatus = Playing
	sm.PreloadedSong.notifySubscribers(TimeChange)
	for {
		if sm.PlayStatus != Playing {
			break
		}

		sm.timeMutex.Lock()
		nextTime := math.Floor(sm.PreloadedSong.TimePassed+0.1) + 1.0
		deltaSeconds := nextTime - sm.PreloadedSong.TimePassed
		sm.PreloadedSong.TimePassed = nextTime
		sm.timeMutex.Unlock()

		<-time.After(time.Duration(deltaSeconds) * time.Second)

		if nextTime >= sm.PreloadedSong.Duration {
			sm.PlayStatus = Ended
			break
		} else {
			sm.PreloadedSong.notifySubscribers(TimeChange)
		}
	}
	sm.PreloadedSong.notifySubscribers(TimeChange)
}

func (sm *StateMachine) RemoveFromList() {
	sm.DownloadStatus = Removed
	if sm.PlayStatus == Playing {
		sm.PlayStatus = Ended
	}
	sm.PreloadedSong.notifySubscribers(StatusChange)
}
