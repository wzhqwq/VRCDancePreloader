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
	sync.Mutex
	pw      progress.Writer
	pl      *playlist.PlayList
	items   []*SongTui
	StopCh  chan struct{}
	itemMap map[string]*SongTui
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
	}
}

func (plt *PlayListTui) NewSongTui(ps *song.PreloadedSong) *SongTui {
	return &SongTui{
		ps:     ps,
		plt:    plt,
		StopCh: make(chan struct{}),
	}
}

func (st *PlayListTui) RenderLoop() {
	changeCh := st.pl.SubscribeChangeEvent()
	newItemCh := st.pl.SubscribeNewItemEvent()
	for {
		select {
		case <-st.StopCh:
			for _, item := range st.items {
				item.StopCh <- struct{}{}
			}
			return
		case change := <-changeCh:
			switch change {
			case playlist.ItemsChange:
				st.items = lo.Map(st.pl.Items, func(ps *song.PreloadedSong, _ int) *SongTui {
					if item, ok := st.itemMap[ps.GetInfo().ID]; ok {
						return item
					}
					// Never reach here
					log.Println("An unexpected behavior: new item in playlist but not in itemMap")
					return nil
				})
				st.Print()
			}
		case newItem := <-newItemCh:
			newTui := st.NewSongTui(newItem)
			st.itemMap[newItem.GetInfo().ID] = newTui
			go newTui.RenderLoop()
		}
	}
}

func (plt *PlayListTui) Print() {
	plt.Lock()
	defer plt.Unlock()

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{i18n.T("key_id"), i18n.T("key_title"), i18n.T("key_status")})
	t.AppendRows(lo.Map(plt.items, func(item *SongTui, _ int) table.Row {
		info := item.ps.GetInfo()
		return table.Row{
			info.ID,
			info.Title,
			item.ps.GetStatusInfo().Status,
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
				st.plt.Lock()
				st.pt.SetValue(int64(st.ps.DownloadedSize))
				st.plt.Unlock()
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
					st.pt.MarkAsDone()
					st.pt = nil
				case song.Failed:
					st.pt.MarkAsErrored()
					st.pt = nil
				case song.Removed:
					st.plt.itemMap[st.ps.GetInfo().ID] = nil
					return
				}
				st.plt.Print()
			case song.TimeChange:
				st.IsPlaying = st.ps.GetTimeInfo().IsPlaying
			}
		}
	}
}

func Start() {
	ch := playlist.SubscribeNewListEvent()
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
			}
		}
	}()
}
func Stop() {
	stopCh <- struct{}{}
}
