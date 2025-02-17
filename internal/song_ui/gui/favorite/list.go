package favorite

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
)

func NewFavoritesPage() fyne.CanvasObject {
	return NewFavoritesGui().Container
}

type FavoritesGui struct {
	Container *container.Scroll
	List      *fyne.Container

	StopCh   chan struct{}
	changeCh chan string
}

func NewFavoritesGui() *FavoritesGui {
	list := container.NewVBox()
	scroll := container.NewVScroll(list)
	scroll.SetMinSize(fyne.NewSize(300, 400))

	g := &FavoritesGui{
		Container: scroll,
		List:      list,

		StopCh:   make(chan struct{}),
		changeCh: persistence.GetFavorite().SubscribeEvent(),
	}

	go g.RenderLoop()

	return g
}

func (fg *FavoritesGui) RenderLoop() {
	fg.refreshItems()

	for {
		select {
		case <-fg.StopCh:
			return
		case <-fg.changeCh:
			fg.refreshItems()
		}
	}
}

func (fg *FavoritesGui) refreshItems() {
	entries := persistence.GetFavorite().ListFavorites(0, 20, "id", true)
	fg.List.RemoveAll()
	for _, entry := range entries {
		fg.List.Add(NewItemGui(entry).Card)
	}
	fg.List.Refresh()
	fg.Container.Refresh()
}
