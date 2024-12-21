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

	ps *PreloadedSong

	// waiter
	songWaiting utils.FinishingBroadcaster

	// locks
	timeMutex sync.Mutex
}

func NewSongStateMachine() *StateMachine {
	return &StateMachine{
		DownloadStatus: Initial,
		PlayStatus:     Queued,
		ps:             nil,
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
	sm.StartDownload()
	sm.Prioritize()
	sm.songWaiting.WaitForFinishing()
	if sm.DownloadStatus == Downloaded {
		return nil
	}
	if sm.DownloadStatus == Removed {
		return fmt.Errorf("download removed")
	}
	return sm.ps.PreloadError
}
func (sm *StateMachine) SubscribePartialDownload(closeCh chan struct{}) (int64, chan int64, error) {
	availableSizeCh := make(chan int64, 10)

	eventCh := sm.ps.SubscribeEvent()
	sm.WaitForSong(closeCh, eventCh)

	switch sm.DownloadStatus {
	case Removed:
		return 0, nil, fmt.Errorf("download removed")
	case Failed:
		return 0, nil, sm.ps.PreloadError
	}

	availableSizeCh <- sm.ps.DownloadedSize
	go func() {
		for {
			select {
			case event := <-eventCh:
				switch event {
				case StatusChange:
					switch sm.DownloadStatus {
					case Downloaded:
						availableSizeCh <- sm.ps.DownloadedSize
						return
					case Removed, Failed:
						availableSizeCh <- -1
						return
					}
				case ProgressChange:
					availableSizeCh <- sm.ps.DownloadedSize
				}
			case <-closeCh:
				return
			}
		}
	}()

	return sm.ps.TotalSize, availableSizeCh, nil
}
func (sm *StateMachine) WaitForSong(closeCh chan struct{}, eventCh chan ChangeType) {
	sm.StartDownload()
	sm.Prioritize()
	if sm.DownloadStatus == Pending || sm.DownloadStatus == Requesting {
		for {
			select {
			case <-closeCh:
				return
			case <-eventCh:
				if sm.DownloadStatus != Pending && sm.DownloadStatus != Requesting {
					return
				}
			}
		}
	}
}
func (sm *StateMachine) StartDownload() {
	if sm.DownloadStatus == Downloaded {
		return
	}
	if !sm.IsDownloadLoopStarted() {
		sm.DownloadStatus = Pending
		sm.ps.notifySubscribers(StatusChange)

		ds := cache.Download(sm.ps.GetId(), sm.ps.GetDownloadUrl())
		go sm.StartDownloadLoop(ds)
	}
}
func (sm *StateMachine) Prioritize() {
	if sm.IsPlayingLoopStarted() {
		cache.Prioritize(sm.ps.GetId())
	}
}

func (sm *StateMachine) StartDownloadLoop(ds *cache.DownloadState) {
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
	}
	sm.ps.notifySubscribers(StatusChange)
	cache.CancelDownload(sm.ps.GetId())
}
