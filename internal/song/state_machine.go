package song

import (
	"errors"
	"sync"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader/task"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// StateMachine is the state machine for a song
type StateMachine struct {
	DownloadStatus DownloadStatus
	PlayStatus     PlayStatus

	ps *StatefulSong

	session types.CDNFileSession
	task    *downloader.ManagedTask

	// waiter
	completeSongWg sync.WaitGroup

	// channels
	syncTimeCh chan time.Duration

	// locks
	timeMutex sync.Mutex
	taskMutex sync.Mutex

	wg sync.WaitGroup
}

func NewSongStateMachine() *StateMachine {
	sm := &StateMachine{
		DownloadStatus: Initial,
		PlayStatus:     Queued,

		syncTimeCh: make(chan time.Duration, 1),
	}

	return sm
}

func (sm *StateMachine) Go(fn func()) {
	sm.wg.Go(fn)
}

func (sm *StateMachine) BindCache(createFn func() (types.CDNFileSession, error)) bool {
	if sm.session != nil {
		return true
	}

	session, err := createFn()
	if err != nil {
		return false
	}

	sm.session = session
	err = session.Open(sm.ps.SongId(), activeSongLogger)
	if err != nil {
		sm.DownloadStatus = NotAvailable
		sm.ps.notifyStatusChange()
		return false
	}

	return true
}

func (sm *StateMachine) BindTask(createFn func(session types.CDNFileSession) *downloader.ManagedTask) {
	sm.taskMutex.Lock()
	defer sm.taskMutex.Unlock()

	if !sm.IsDownloadNeeded() {
		return
	}

	if !sm.IsDownloadLoopStarted() {
		t := createFn(sm.session)
		if t == nil {
			return
		}
		sm.task = t

		sm.SwitchDownloadStatus(Pending)

		sm.Go(sm.StartDownloadLoop)
	}
}

func (sm *StateMachine) SwitchDownloadStatus(s DownloadStatus) {
	if sm.DownloadStatus == s {
		return
	}
	sm.DownloadStatus = s
	sm.ps.notifyStatusChange()
}

func (sm *StateMachine) StartDownloadLoop() {
	sm.completeSongWg.Add(1)
	defer sm.completeSongWg.Done()

	sm.ps.PreloadError = nil

	lazy := utils.NewLazy(func() {
		sm.ps.notifyLazySubscribers(ProgressChange)
	})

	ch := sm.task.SubscribeChanges()
	defer ch.Close()

	for {
		select {
		case change := <-ch.Channel:
			switch change {
			case task.State:
				if sm.task.Error != nil {
					if errors.Is(sm.task.Error, task.ErrCanceled) {
						return
					}

					sm.DownloadStatus = Failed
					sm.ps.PreloadError = sm.task.Error
					sm.ps.notifyStatusChange()
					sm.task.Retry()
				} else {
					sm.ps.PreloadError = nil

					switch sm.task.State {
					case task.TaskCompleted:
						sm.ps.TotalSize = sm.task.TotalSize
						sm.ps.DownloadedSize = sm.task.DownloadedSize
						sm.SwitchDownloadStatus(Downloaded)
						sm.ps.notifySubscribers(ProgressChange)
					case task.TaskPending:
						sm.SwitchDownloadStatus(Pending)
					case task.TaskWaitScheduled:
						sm.SwitchDownloadStatus(CoolingDown)
					case task.TaskRequested:
						sm.SwitchDownloadStatus(Requesting)
					case task.TaskDownloading:
						sm.ps.TotalSize = sm.task.TotalSize
						sm.SwitchDownloadStatus(Downloading)
					}
				}
			case task.Progress:
				sm.ps.DownloadedSize = sm.task.DownloadedSize
				sm.ps.notifySubscribers(ProgressChange)
				lazy.Change()
			}
		case <-lazy.WaitUpdate():
			lazy.Update()
		}
	}
}

func (sm *StateMachine) PlaySongAndSync(offset time.Duration) {
	if sm.PlayStatus == Ended {
		return
	}

	sm.syncTimeCh <- offset

	queued := sm.PlayStatus == Queued
	sm.PlayStatus = SyncPlaying
	if queued {
		sm.Go(sm.StartPlayingLoop)
	}
}

func (sm *StateMachine) PlaySong() {
	if sm.PlayStatus == Ended {
		return
	}

	queued := sm.PlayStatus == Queued
	sm.PlayStatus = Playing
	if queued {
		sm.Go(sm.StartPlayingLoop)
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
	sm.ps.notifyTimeChange(false)
	startTime := time.Now()
	for {
		if !sm.IsPlaying() {
			break
		}

		realTimePassed := time.Since(startTime)
		nextTime := (sm.ps.TimePassed + time.Second) / time.Second * time.Second
		delta := nextTime - realTimePassed
		routine := false
		select {
		case sm.ps.TimePassed = <-sm.syncTimeCh:
			startTime = time.Now().Add(-sm.ps.TimePassed)
		case <-time.After(delta):
			sm.ps.TimePassed = nextTime
			routine = true
		}

		if sm.ps.info.Duration > 0 && nextTime >= sm.ps.info.Duration {
			sm.PlayStatus = Ended
			sm.ps.AddToHistory()
			break
		} else if sm.PlayStatus == SyncPlaying {
			sm.ps.notifyTimeChange(routine)
		}
	}
	sm.ps.notifyTimeChange(false)
}

func (sm *StateMachine) RemoveFromList() {
	sm.DownloadStatus = Removed
	if sm.IsPlaying() {
		sm.PlayStatus = Ended
		if sm.ps.TimePassed > 20*time.Second {
			sm.ps.AddToHistory()
		}
	}
	sm.ps.notifyStatusChange()
	sm.CancelTask()
	if sm.session != nil {
		sm.session.Close(removedSongLogger)
		sm.session = nil
	}
	sm.wg.Wait()
}

func (sm *StateMachine) CancelTask() {
	if sm.task != nil {
		sm.task.Cancel()
		sm.task = nil
	}
}
