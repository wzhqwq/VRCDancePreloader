package tui

import (
	"fmt"
	"log"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/wzhqwq/PyPyDancePreloader/internal/song"
)

type SongTui struct {
	pt        *progress.Tracker
	ps        *song.PreloadedSong
	plt       *PlayListTui
	IsPlaying bool
	StopCh    chan struct{}
}

func (st *SongTui) RenderLoop() {
	ch := st.ps.SubscribeEvent()
	for {
		select {
		case <-st.StopCh:
			return
		case event := <-ch:
			switch event {
			case song.ProgressChange:
				st.plt.stdoutMutex.Lock()
				if st.pt != nil {
					st.pt.SetValue(st.ps.DownloadedSize)
				}
				st.plt.stdoutMutex.Unlock()
			case song.StatusChange:
				switch st.ps.GetPreloadStatus() {
				case song.Downloading:
					if st.pt == nil {
						st.pt = &progress.Tracker{
							Message: fmt.Sprintf("Downloading %s", st.ps.GetInfo().Title),
							Total:   int64(st.ps.TotalSize),
							Units:   progress.UnitsBytes,
						}
						st.plt.pw.AppendTracker(st.pt)
					}
				case song.Downloaded:
					if st.pt != nil {
						st.pt.MarkAsDone()
						st.pt = nil
					}
				case song.Failed:
					if st.pt != nil {
						st.pt.MarkAsErrored()
						log.Printf("Preload %s error: %s\n", st.ps.GetId(), st.ps.PreloadError.Error())
						st.pt = nil
					}
				case song.Removed:
					st.plt.removeFromMap(st.ps.GetId())
					return
				}
				st.plt.Print()
			case song.TimeChange:
				st.IsPlaying = st.ps.GetTimeInfo().IsPlaying
				//fmt.Printf("Song %s at %s\n", st.ps.GetInfo().Title, st.ps.GetTimeInfo().Text)
			}
		}
	}
}
