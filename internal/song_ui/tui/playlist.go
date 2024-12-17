package tui

import (
	"sync"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/samber/lo"
	"github.com/wzhqwq/PyPyDancePreloader/internal/playlist"
	"github.com/wzhqwq/PyPyDancePreloader/internal/song"
)

type PlayListTui struct {
	pw progress.Writer
	pl *playlist.PlayList
	t  *SongTable

	items   []*SongTui
	StopCh  chan struct{}
	itemMap map[string]*SongTui

	stdoutMutex sync.Mutex
	mapMutex    sync.Mutex
}

func NewPlayListTui(pl *playlist.PlayList) *PlayListTui {
	return &PlayListTui{
		pw: progress.NewWriter(),
		pl: pl,
		t:  NewSongTable(),

		items:   make([]*SongTui, 0),
		StopCh:  make(chan struct{}),
		itemMap: make(map[string]*SongTui),

		stdoutMutex: sync.Mutex{},
		mapMutex:    sync.Mutex{},
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

	plt.t.Print(plt.items)
}
