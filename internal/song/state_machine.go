package song

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// StateMachine is the state machine for a song
type StateMachine struct {
	DownloadStatus DownloadStatus
	PlayStatus     PlayStatus

	ps *PreloadedSong
	ce cache.Entry

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
		syncTimeCh:     make(chan time.Duration, 1),
	}

	return sm
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

	if sm.DownloadStatus == Initial {
		// Call OpenCacheEntry to increase the reference count
		// We will release it in RemoveFromList
		entry, err := cache.OpenCacheEntry(sm.ps.GetSongId(), "[ActiveSong]")
		if err != nil {
			sm.DownloadStatus = NotAvailable
			sm.ps.notifyStatusChange()
			return
		}
		sm.ce = entry
	}

	if !sm.IsDownloadLoopStarted() {
		sm.DownloadStatus = Pending
		sm.ps.notifyStatusChange()

		ds := download.Download(sm.ps.GetSongId())
		if ds == nil {
			sm.DownloadStatus = NotAvailable
			sm.ps.notifyStatusChange()
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

func (sm *StateMachine) SwitchDownloadStatus(s DownloadStatus) {
	if sm.DownloadStatus == s {
		return
	}
	sm.DownloadStatus = s
	sm.ps.notifyStatusChange()
}

func (sm *StateMachine) StartDownloadLoop(ds *download.State) {
	sm.completeSongWg.Add(1)
	defer sm.completeSongWg.Done()

	sm.ps.PreloadError = nil

	lazy := utils.NewLazy(func() {
		sm.ps.notifyLazySubscribers(ProgressChange)
	})

	for {
		select {
		case <-ds.StateCh:
			if ds.Done {
				sm.DownloadStatus = Downloaded
				sm.ps.TotalSize = ds.TotalSize
				sm.ps.DownloadedSize = ds.DownloadedSize
				sm.ps.notifySubscribers(ProgressChange)
				sm.ps.notifyStatusChange()
				return
			}
			if ds.Error != nil {
				if errors.Is(ds.Error, cache.ErrNotSupported) {
					sm.DownloadStatus = NotAvailable
					sm.ps.notifyStatusChange()
					download.CancelDownload(sm.ps.GetSongId())
					return
				}
				if errors.Is(ds.Error, download.ErrCanceled) {
					return
				}
				sm.DownloadStatus = Failed
				sm.ps.PreloadError = ds.Error
				sm.ps.notifyStatusChange()
				download.Retry(ds)
				continue
			} else {
				sm.ps.PreloadError = nil
			}
			if ds.Pending {
				sm.SwitchDownloadStatus(Pending)
				continue
			}
			if ds.Cooling {
				sm.SwitchDownloadStatus(CoolingDown)
				continue
			}
			if ds.Requesting {
				sm.SwitchDownloadStatus(Requesting)
				continue
			}
			// Otherwise, it's downloading
			if sm.DownloadStatus == Removed {
				return
			}

			sm.SwitchDownloadStatus(Downloading)

			sm.ps.TotalSize = ds.TotalSize
			sm.ps.DownloadedSize = ds.DownloadedSize
			sm.ps.notifySubscribers(ProgressChange)
			lazy.Change()
		case <-lazy.WaitUpdate():
			lazy.Update()
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
	}
}

func (sm *StateMachine) CancelPlayingLoop() {
	if sm.DownloadStatus == Removed {
		return
	}
	if sm.PlayStatus != Queued {
		sm.PlayStatus = Queued
		sm.ps.notifyTimeChange(false)
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
		routine := false
		select {
		case sm.ps.TimePassed = <-sm.syncTimeCh:
			startTime = time.Now().Add(-sm.ps.TimePassed)
		case <-time.After(delta):
			sm.ps.TimePassed = nextTime
			routine = true
		}

		if nextTime >= sm.ps.Duration {
			sm.PlayStatus = Ended
			sm.ps.AddToHistory()
			break
		} else {
			sm.ps.notifyTimeChange(routine)
		}
	}
	sm.ps.notifyTimeChange(false)
}

func (sm *StateMachine) RemoveFromList() {
	sm.DownloadStatus = Removed
	if sm.PlayStatus == Playing {
		sm.PlayStatus = Ended
		if sm.ps.TimePassed > 20*time.Second {
			sm.ps.AddToHistory()
		}
	}
	sm.ps.notifyStatusChange()
	download.CancelDownload(sm.ps.GetSongId())
	if sm.ce != nil {
		cache.ReleaseCacheEntry(sm.ps.GetSongId(), "[RemovedSong]")
		sm.ce = nil
	}
}
