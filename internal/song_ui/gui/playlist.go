package gui

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/containers"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
)

type PlayListGui struct {
	pl *playlist.PlayList

	Container *container.Scroll

	items   []*ItemGui
	itemMap map[string]*ItemGui

	StopCh   chan struct{}
	changeCh chan playlist.ChangeType

	list *containers.DynamicList

	mapMutex sync.Mutex
}

func NewPlayListGui(pl *playlist.PlayList) *PlayListGui {
	list := containers.NewDynamicList(playItemMinWidth)
	scroll := container.NewVScroll(list)
	scroll.SetMinSize(fyne.NewSize(playItemMinWidth+theme.Padding(), 400))

	return &PlayListGui{
		Container: scroll,

		items:   make([]*ItemGui, 0),
		itemMap: make(map[string]*ItemGui),
		list:    list,

		pl: pl,

		StopCh:   make(chan struct{}),
		changeCh: pl.SubscribeChangeEvent(),
	}
}

func (plg *PlayListGui) RenderLoop() {
	plg.refreshItems()

	for {
		select {
		case <-plg.StopCh:
			for _, item := range plg.items {
				item.StopCh <- struct{}{}
			}
			return
		case change := <-plg.changeCh:
			switch change {
			case playlist.ItemsChange:
				plg.refreshItems()
			}
		}
	}
}

func (plg *PlayListGui) refreshItems() {
	plg.mapMutex.Lock()
	defer func() {
		plg.mapMutex.Unlock()
		plg.list.SetOrder(lo.Map(plg.items, func(item *ItemGui, _ int) string {
			return item.ps.GetId()
		}))
		plg.Container.Refresh()
	}()

	plg.items = lo.Map(plg.pl.Items, func(ps *song.PreloadedSong, _ int) *ItemGui {
		if item, ok := plg.itemMap[ps.GetId()]; ok {
			return item
		}
		newGui := NewItemGui(ps, plg)
		plg.itemMap[ps.GetId()] = newGui
		plg.list.AddItem(newGui.listItem)
		go func() {
			time.Sleep(200 * time.Millisecond)
			newGui.SlideIn()
			newGui.RenderLoop()
		}()
		return newGui
	})
}

func (plg *PlayListGui) removeFromMap(id string) {
	plg.mapMutex.Lock()
	defer plg.mapMutex.Unlock()
	plg.list.RemoveItem(id)
	delete(plg.itemMap, id)
}
