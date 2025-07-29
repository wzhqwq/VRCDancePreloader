package song

import (
	"fmt"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"sync"
	"time"
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
	completeSongWg sync.WaitGroup

	// channels
	syncTimeCh chan time.Duration

	// locks
	timeMutex          sync.Mutex
	startDownloadMutex sync.Mutex
}

func NewSongStateMachine() *StateMachine {
	sm := &StateMachine{
		DownloadStatus: Initial,
		PlayStatus:     Queued,
		ps:             nil,
		syncTimeCh:     make(chan time.Duration, 1),
		completeSongWg: sync.WaitGroup{},
		timeMutex:      sync.Mutex{},
	}

	return sm
}

func (sm *StateMachine) IsDownloadLoopStarted() bool {
	return sm.DownloadStatus == Pending || sm.DownloadStatus == Requesting || sm.DownloadStatus == Downloading
}
func (sm *StateMachine) IsDownloadNeeded() bool {
	return sm.DownloadStatus != Downloaded && sm.DownloadStatus != Removed && sm.DownloadStatus != NotAvailable
}
func (sm *StateMachine) CanPreload() bool {
	return sm.DownloadStatus != NotAvailable && (sm.DownloadStatus == Initial || sm.DownloadStatus == Failed)
}
func (sm *StateMachine) IsPlayingLoopStarted() bool {
	return sm.PlayStatus == Playing
}

func (sm *StateMachine) DownloadInstantly(waitComplete bool) error {
	sm.StartDownload()
	sm.Prioritize()
	if waitComplete {
		sm.completeSongWg.Wait()
	}

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
	sm.startDownloadMutex.Lock()
	defer sm.startDownloadMutex.Unlock()

	if !sm.IsDownloadNeeded() {
		return
	}
	if !sm.IsDownloadLoopStarted() {
		sm.DownloadStatus = Pending
		sm.ps.notifySubscribers(StatusChange)

		ds := download.Download(sm.ps.GetSongId())
		if ds == nil {
			sm.DownloadStatus = NotAvailable
			sm.ps.notifySubscribers(StatusChange)
			return
		}
		go sm.StartDownloadLoop(ds)
	}
}
func (sm *StateMachine) Prioritize() {
	if sm.IsDownloadLoopStarted() {
		download.Prioritize(sm.ps.GetSongId())
	}
}

func (sm *StateMachine) StartDownloadLoop(ds *download.State) {
	sm.completeSongWg.Add(1)
	defer sm.completeSongWg.Done()

	sm.ps.PreloadError = nil

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

func (sm *StateMachine) PlaySongStartFrom(offset time.Duration) {
	if sm.PlayStatus == Ended {
		return
	}

	sm.syncTimeCh <- offset

	if sm.PlayStatus == Queued {
		go sm.StartPlayingLoop()
	} else {
		sm.ps.notifySubscribers(TimeChange)
	}
}

func (sm *StateMachine) CancelPlayingLoop() {
	if sm.DownloadStatus == Removed {
		return
	}
	if sm.PlayStatus != Queued {
		sm.PlayStatus = Queued
		sm.ps.notifySubscribers(TimeChange)
	}
}

func (sm *StateMachine) StartPlayingLoop() {
	sm.PlayStatus = Playing
	startTime := time.Now()
	for {
		if sm.PlayStatus != Playing {
			break
		}

		realTimePassed := time.Since(startTime)
		nextTime := (sm.ps.TimePassed/time.Second + 1) * time.Second
		delta := nextTime - realTimePassed
		select {
		case sm.ps.TimePassed = <-sm.syncTimeCh:
			startTime = time.Now().Add(-sm.ps.TimePassed)
		case <-time.After(delta):
			sm.ps.TimePassed = nextTime
		}

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
		if sm.ps.TimePassed > 20*time.Second {
			sm.ps.AddToHistory()
		}
	}
	sm.ps.notifySubscribers(StatusChange)
	download.CancelDownload(sm.ps.GetSongId())
	cache.CloseCacheEntry(sm.ps.GetSongId())
}
