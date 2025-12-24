package tui

import (
	"sync"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
)

type PlayListTui struct {
	pw progress.Writer
	pl *playlist.PlayList
	t  *SongTable

	items   []*ItemTui
	StopCh  chan struct{}
	itemMap map[int64]*ItemTui

	stdoutMutex sync.Mutex
	mapMutex    sync.Mutex
}

func NewPlayListTui(pl *playlist.PlayList) *PlayListTui {
	return &PlayListTui{
		pw: progress.NewWriter(),
		pl: pl,
		t:  NewSongTable(),

		items:   make([]*ItemTui, 0),
		StopCh:  make(chan struct{}),
		itemMap: make(map[int64]*ItemTui),

		stdoutMutex: sync.Mutex{},
		mapMutex:    sync.Mutex{},
	}
}

func (plt *PlayListTui) RenderLoop() {
	plt.refreshItems()
	changeCh := plt.pl.SubscribeChangeEvent()
	defer changeCh.Close()

	go plt.pw.Render()
	defer plt.pw.Stop()

	for {
		select {
		case <-plt.StopCh:
			for _, item := range plt.items {
				close(item.StopCh)
			}
			return
		case change := <-changeCh.Channel:
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

	songs := plt.pl.GetItemsSnapshot()

	plt.items = lo.Map(songs, func(ps *song.PreloadedSong, _ int) *ItemTui {
		if item, ok := plt.itemMap[ps.ID]; ok {
			return item
		}
		newTui := NewSongTui(ps, plt)
		plt.itemMap[ps.ID] = newTui
		go newTui.RenderLoop()
		return newTui
	})
}

func (plt *PlayListTui) removeFromMap(id int64) {
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
