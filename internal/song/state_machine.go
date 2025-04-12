package song

import (
	"fmt"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"math"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
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
	// NotAvailable means the song cannot be cached by now
	NotAvailable DownloadStatus = "na"
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

	ps *PreloadedSong

	// waiter
	songWaiting *utils.FinishingBroadcaster

	// locks
	timeMutex sync.Mutex
}

func NewSongStateMachine() *StateMachine {
	return &StateMachine{
		DownloadStatus: Initial,
		PlayStatus:     Queued,
		ps:             nil,
		songWaiting:    utils.NewBroadcaster(),
		timeMutex:      sync.Mutex{},
	}
}

func (sm *StateMachine) IsDownloadLoopStarted() bool {
	return sm.DownloadStatus == Pending || sm.DownloadStatus == Requesting || sm.DownloadStatus == Downloading
}
func (sm *StateMachine) IsDownloadNeeded() bool {
	return sm.DownloadStatus != Downloaded && sm.DownloadStatus != Removed && sm.DownloadStatus != NotAvailable
}
func (sm *StateMachine) IsPlayingLoopStarted() bool {
	return sm.PlayStatus == Playing
}

func (sm *StateMachine) WaitForCompleteSong() error {
	sm.StartDownload()
	sm.Prioritize()
	sm.songWaiting.WaitForFinishing()

	switch sm.DownloadStatus {
	case Removed:
		return fmt.Errorf("download removed")
	case Failed:
		return sm.ps.PreloadError
	default:
		return nil
	}
}
func (sm *StateMachine) WaitForSong() error {
	sm.StartDownload()
	sm.Prioritize()

	switch sm.DownloadStatus {
	case Removed:
		return fmt.Errorf("download removed")
	case Failed:
		return sm.ps.PreloadError
	default:
		return nil
	}
}
func (sm *StateMachine) StartDownload() {
	if !sm.IsDownloadNeeded() {
		return
	}
	if !sm.IsDownloadLoopStarted() {
		sm.DownloadStatus = Pending
		sm.ps.notifySubscribers(StatusChange)

		ds := download.Download(sm.ps.GetId())
		if ds == nil {
			sm.DownloadStatus = NotAvailable
			sm.ps.notifySubscribers(StatusChange)
			return
		}
		go sm.StartDownloadLoop(ds)
	}
}
func (sm *StateMachine) Prioritize() {
	if sm.IsPlayingLoopStarted() {
		download.Prioritize(sm.ps.GetId())
	}
}

func (sm *StateMachine) StartDownloadLoop(ds *download.DownloadState) {
	defer sm.songWaiting.Finish()
	for {
		select {
		case <-ds.StateCh:
			if ds.Done {
				sm.DownloadStatus = Downloaded
				sm.ps.TotalSize = ds.TotalSize
				sm.ps.DownloadedSize = ds.DownloadedSize
				sm.ps.notifySubscribers(ProgressChange)
				sm.ps.notifySubscribers(StatusChange)
				return
			}
			if ds.Error != nil {
				sm.DownloadStatus = Failed
				sm.ps.PreloadError = ds.Error
				sm.ps.notifySubscribers(StatusChange)
				return
			}
			if ds.Pending && sm.DownloadStatus != Pending {
				sm.DownloadStatus = Pending
				sm.ps.notifySubscribers(StatusChange)
				continue
			}
			if ds.TotalSize == 0 && sm.DownloadStatus != Requesting {
				sm.DownloadStatus = Requesting
				sm.ps.notifySubscribers(StatusChange)
				continue
			}
			// Otherwise, it's downloading
			if sm.DownloadStatus == Removed {
				return
			}
			if sm.DownloadStatus != Downloading {
				sm.DownloadStatus = Downloading
				sm.ps.notifySubscribers(StatusChange)
			}
			sm.ps.TotalSize = ds.TotalSize
			sm.ps.DownloadedSize = ds.DownloadedSize
			sm.ps.notifySubscribers(ProgressChange)
		}
	}
}

func (sm *StateMachine) PlaySongStartFrom(offset float64) {
	if sm.PlayStatus == Ended {
		return
	}

	sm.timeMutex.Lock()
	sm.ps.TimePassed = offset
	sm.timeMutex.Unlock()

	if sm.PlayStatus == Queued {
		go sm.StartPlayingLoop()
	} else {
		sm.ps.notifySubscribers(TimeChange)
	}
}

func (sm *StateMachine) StartPlayingLoop() {
	sm.PlayStatus = Playing
	sm.ps.notifySubscribers(TimeChange)
	for {
		if sm.PlayStatus != Playing {
			break
		}

		sm.timeMutex.Lock()
		nextTime := math.Floor(sm.ps.TimePassed+0.1) + 1.0
		deltaSeconds := nextTime - sm.ps.TimePassed
		sm.ps.TimePassed = nextTime
		sm.timeMutex.Unlock()

		<-time.After(time.Duration(deltaSeconds) * time.Second)

		if nextTime >= sm.ps.Duration {
			sm.PlayStatus = Ended
			sm.ps.AddToHistory()
			break
		} else {
			sm.ps.notifySubscribers(TimeChange)
		}
	}
	sm.ps.notifySubscribers(TimeChange)
}

func (sm *StateMachine) RemoveFromList() {
	sm.DownloadStatus = Removed
	if sm.PlayStatus == Playing {
		sm.PlayStatus = Ended
		sm.ps.AddToHistory()
	}
	sm.ps.notifySubscribers(StatusChange)
	download.CancelDownload(sm.ps.GetId())
	cache.CloseCacheEntry(sm.ps.GetId())
}
