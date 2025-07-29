package tui

import (
	"fmt"
	"log"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
)

type ItemTui struct {
	pt        *progress.Tracker
	ps        *song.PreloadedSong
	plt       *PlayListTui
	IsPlaying bool
	StopCh    chan struct{}
}

func NewSongTui(ps *song.PreloadedSong, plt *PlayListTui) *ItemTui {
	return &ItemTui{
		ps:     ps,
		plt:    plt,
		StopCh: make(chan struct{}),
	}
}

func (it *ItemTui) RenderLoop() {
	ch := it.ps.SubscribeEvent()
	for {
		select {
		case <-it.StopCh:
			ch.Close()
			return
		case event := <-ch.Channel:
			switch event {
			case song.ProgressChange:
				it.plt.stdoutMutex.Lock()
				if it.pt != nil {
					it.pt.SetValue(it.ps.DownloadedSize)
				}
				it.plt.stdoutMutex.Unlock()
			case song.StatusChange:
				switch it.ps.GetPreloadStatus() {
				case song.Downloading:
					if it.pt == nil {
						it.pt = &progress.Tracker{
							Message: fmt.Sprintf("Downloading %s", it.ps.GetInfo().Title),
							Total:   int64(it.ps.TotalSize),
							Units:   progress.UnitsBytes,
						}
						it.plt.pw.AppendTracker(it.pt)
					}
				case song.Downloaded:
					if it.pt != nil {
						it.pt.MarkAsDone()
						it.pt = nil
					}
				case song.Failed:
					if it.pt != nil {
						it.pt.MarkAsErrored()
						log.Printf("Preload %s error: %s\n", it.ps.GetSongId(), it.ps.PreloadError.Error())
						it.pt = nil
					}
				case song.Removed:
					it.plt.removeFromMap(it.ps.ID)
					return
				}
				it.plt.Print()
			case song.TimeChange:
				it.IsPlaying = it.ps.GetTimeInfo().IsPlaying
				//fmt.Printf("Song %s at %s\n", it.ps.GetInfo().Title, it.ps.GetTimeInfo().Text)
			}
		}
	}
}
