package tui

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
	"github.com/wzhqwq/PyPyDancePreloader/internal/playlist"
	"github.com/wzhqwq/PyPyDancePreloader/internal/song"
)

var currentTui *PlayListTui
var stopCh chan struct{}

type PlayListTui struct {
	pw      progress.Writer
	pl      *playlist.PlayList
	items   []*SongTui
	StopCh  chan struct{}
	itemMap map[string]*SongTui

	stdoutMutex sync.Mutex
	mapMutex    sync.Mutex

	lastStatus map[string]string
}
type SongTui struct {
	pt        *progress.Tracker
	ps        *song.PreloadedSong
	plt       *PlayListTui
	IsPlaying bool
	StopCh    chan struct{}
}

func NewPlayListTui(pl *playlist.PlayList) *PlayListTui {
	return &PlayListTui{
		pw:      progress.NewWriter(),
		pl:      pl,
		items:   make([]*SongTui, 0),
		StopCh:  make(chan struct{}),
		itemMap: make(map[string]*SongTui),

		stdoutMutex: sync.Mutex{},
		mapMutex:    sync.Mutex{},

		lastStatus: make(map[string]string),
	}
}

func (plt *PlayListTui) NewSongTui(ps *song.PreloadedSong) *SongTui {
	return &SongTui{
		ps:     ps,
		plt:    plt,
		StopCh: make(chan struct{}),
	}
}

func (plt *PlayListTui) RenderLoop() {
	plt.refreshItems()
	changeCh := plt.pl.SubscribeChangeEvent()

	go plt.pw.Render()
	defer plt.pw.Stop()

	for {
		select {
		case <-plt.StopCh:
			for _, item := range plt.items {
				item.StopCh <- struct{}{}
			}
			return
		case change := <-changeCh:
			switch change {
			case playlist.ItemsChange:
				plt.refreshItems()
			}
		}
	}
}

func (plt *PlayListTui) refreshItems() {
	plt.mapMutex.Lock()
	defer func() {
		plt.mapMutex.Unlock()
		plt.Print()
	}()

	plt.items = lo.Map(plt.pl.Items, func(ps *song.PreloadedSong, _ int) *SongTui {
		if item, ok := plt.itemMap[ps.GetId()]; ok {
			return item
		}
		newTui := plt.NewSongTui(ps)
		plt.itemMap[ps.GetId()] = newTui
		go newTui.RenderLoop()
		return newTui
	})
}

func (plt *PlayListTui) removeFromMap(id string) {
	plt.mapMutex.Lock()
	defer plt.mapMutex.Unlock()
	delete(plt.itemMap, id)
}

func (plt *PlayListTui) Print() {
	plt.stdoutMutex.Lock()
	plt.mapMutex.Lock()
	defer plt.mapMutex.Unlock()
	defer plt.stdoutMutex.Unlock()

	allTheSame := true
	statusMap := map[string]string{}
	for _, item := range plt.items {
		status := item.ps.GetStatusInfo().Status
		id := item.ps.GetId()
		statusMap[id] = status
		if lastStatus, ok := plt.lastStatus[id]; !ok || lastStatus != status {
			allTheSame = false
		}
	}
	if allTheSame {
		return
	}
	plt.lastStatus = statusMap

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{i18n.T("key_id"), i18n.T("key_status"), i18n.T("key_title")})
	t.AppendRows(lo.Map(plt.items, func(item *SongTui, _ int) table.Row {
		info := item.ps.GetInfo()
		return table.Row{
			info.ID,
			plt.lastStatus[info.ID],
			info.Title,
		}
	}))
	t.Render()
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

func Start() {
	ch := playlist.SubscribeNewListEvent()
	stopCh = make(chan struct{})
	go func() {
		for {
			select {
			case <-stopCh:
				if currentTui != nil {
					currentTui.StopCh <- struct{}{}
				}
				return
			case pl := <-ch:
				if currentTui != nil {
					currentTui.StopCh <- struct{}{}
				}
				currentTui = NewPlayListTui(pl)
				go currentTui.RenderLoop()
			}
		}
	}()
}
func Stop() {
	stopCh <- struct{}{}
}
