package playlist

import (
	"fyne.io/fyne/v2/widget"
	"sync"
	"time"
	"weak"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/containers"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
)

type ListGui struct {
	widget.BaseWidget

	pl *playlist.PlayList

	StopCh   chan struct{}
	changeCh chan playlist.ChangeType
}

func NewListGui(pl *playlist.PlayList) *ListGui {
	g := &ListGui{
		pl: pl,

		StopCh:   make(chan struct{}),
		changeCh: pl.SubscribeChangeEvent(),
	}

	g.ExtendBaseWidget(g)

	return g
}

func (l *ListGui) RenderLoop() {
	fyne.Do(func() {
		l.Refresh()
	})

	for {
		select {
		case <-l.StopCh:
			return
		case change := <-l.changeCh:
			switch change {
			case playlist.ItemsChange:
				fyne.Do(func() {
					l.Refresh()
				})
			}
		}
	}
}

func (l *ListGui) CreateRenderer() fyne.WidgetRenderer {
	dynamicList := containers.NewDynamicList(playItemMinWidth)
	scroll := container.NewVScroll(dynamicList)
	scroll.SetMinSize(fyne.NewSize(playItemMinWidth+theme.Padding(), 400))

	go l.RenderLoop()

	return &listGuiRenderer{
		list: l,

		Container: scroll,

		itemMap: make(map[string]weak.Pointer[ItemGui]),

		dynamicList: dynamicList,
	}
}

type listGuiRenderer struct {
	list *ListGui

	Container *container.Scroll

	items   []*ItemGui
	itemMap map[string]weak.Pointer[ItemGui]

	dynamicList *containers.DynamicList

	mapMutex sync.Mutex
}

func (r *listGuiRenderer) MinSize() fyne.Size {
	return r.Container.MinSize()
}

func (r *listGuiRenderer) Layout(size fyne.Size) {
	r.Container.Resize(size)
	r.Container.Move(fyne.NewPos(0, 0))
}

func (r *listGuiRenderer) Refresh() {
	r.mapMutex.Lock()

	songs := make([]*song.PreloadedSong, len(r.list.pl.Items))
	copy(songs, r.list.pl.Items)

	r.items = lo.Map(songs, func(ps *song.PreloadedSong, _ int) *ItemGui {
		if item, ok := r.itemMap[ps.GetId()]; ok {
			if v := item.Value(); v != nil {
				return v
			}
		}
		newGui := NewItemGui(ps, r.dynamicList)
		r.itemMap[ps.GetId()] = weak.Make(newGui)
		r.dynamicList.AddItem(newGui.listItem)

		go newGui.RenderLoop()

		fyne.Do(func() {
			newGui.Hide()
			go func() {
				time.Sleep(100 * time.Millisecond)
				newGui.SlideIn()
			}()
		})
		return newGui
	})

	r.mapMutex.Unlock()
	r.dynamicList.SetOrder(lo.Map(r.items, func(item *ItemGui, _ int) string {
		return item.ps.GetId()
	}))
	r.Container.Refresh()
}

func (r *listGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.Container}
}

func (r *listGuiRenderer) Destroy() {
	close(r.list.StopCh)
}
