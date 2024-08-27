package playlist

import (
	"log"
	"math"
	"time"

	"github.com/wzhqwq/PyPyDancePreloader/internal/constants"
	"github.com/wzhqwq/PyPyDancePreloader/internal/gui"
)

func (i *PlayItem) UpdateProgress(downloaded int64) {
	i.updateMutex.Lock()
	defer i.updateMutex.Unlock()

	progress := float64(downloaded) / float64(i.Size)
	if math.Abs(progress-i.Progress) < 0.001 {
		return
	}
	i.Progress = progress
	i.DownloadedBytes = downloaded

	i.updateGui()
}
func (i *PlayItem) UpdatePreloadStatus(status constants.Status) {
	i.updateMutex.Lock()
	defer i.updateMutex.Unlock()

	if i.PreloadStatus == status {
		return
	}
	i.PreloadStatus = status

	switch status {
	case constants.Failed:
		i.Detach()
	case constants.Downloaded:
		CriticalUpdate()
	}

	i.updateGui()
}
func (i *PlayItem) UpdatePlayStatus(status constants.Status) {
	i.updateMutex.Lock()
	defer i.updateMutex.Unlock()

	if i.PlayStatus == status {
		return
	}
	i.PlayStatus = status

	switch status {
	case constants.Ended:
		i.Detach()
	case constants.Playing:
		CriticalUpdate()
		go func() {
			for {
				i.updateGui()
				if i.PlayStatus != constants.Playing {
					break
				}
				<-time.After(time.Millisecond * 100)
				i.Now += 0.1
				if i.Now >= float64(i.Duration) {
					i.UpdatePlayStatus(constants.Ended)
					break
				}
			}
		}()
	}

	i.updateGui()
}
func (i *PlayItem) UpdateIndex(index int) {
	i.updateMutex.Lock()
	defer i.updateMutex.Unlock()

	if i.Index == index {
		return
	}
	i.Index = index
	log.Println("Updated index", i.Title, i.Index)

	i.updateGui()
}
func (i *PlayItem) UpdateSize(size int64) {
	i.updateMutex.Lock()
	defer i.updateMutex.Unlock()

	if i.Size == size {
		return
	}
	i.Size = size

	i.updateGui()
}
func (i *PlayItem) updateGui() {
	if !i.dirty && !i.disposed {
		gui.UpdatePlayItem(i)
		i.dirty = true
	}
}
