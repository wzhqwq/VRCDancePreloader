package playlist

import (
	"weak"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/containers"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/song"
)

type ListGui struct {
	widget.BaseWidget

	pl *playlist.PlayList

	itemChanged bool
	roomChanged bool
}

func NewListGui(pl *playlist.PlayList) *ListGui {
	g := &ListGui{
		pl: pl,
	}

	g.ExtendBaseWidget(g)

	return g
}

func (l *ListGui) RenderLoop(stopCh chan struct{}) {
	ch := l.pl.SubscribeChangeEvent()
	defer ch.Close()

	for {
		select {
		case <-stopCh:
			return
		case change := <-ch.Channel:
			switch change {
			case playlist.ItemsChange:
				l.itemChanged = true
				fyne.Do(func() {
					l.Refresh()
				})
			case playlist.RoomChange:
				l.roomChanged = true
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

	roomName := canvas.NewText(l.pl.RoomBrand, theme.Color(theme.ColorNamePlaceHolder))
	roomName.TextSize = 18
	roomName.TextStyle.Bold = true

	emptyTip := canvas.NewText(i18n.T("tip_playlist_empty"), theme.Color(theme.ColorNamePlaceHolder))
	emptyTip.TextSize = 12

	r := &listGuiRenderer{
		list: l,

		stopCh: make(chan struct{}),

		Container:  scroll,
		Background: canvas.NewRectangle(theme.Color(custom_fyne.ColorNameOuterBackground)),

		RoomName: roomName,
		EmptyTip: emptyTip,

		itemMap: make(map[int64]weak.Pointer[ItemGui]),

		dynamicList: dynamicList,
	}

	r.updateItems()

	go l.RenderLoop(r.stopCh)

	return r
}

var playlistTopHeight = float32(30)

type listGuiRenderer struct {
	list *ListGui

	stopCh chan struct{}

	Container  *container.Scroll
	Background *canvas.Rectangle

	RoomName *canvas.Text
	EmptyTip *canvas.Text

	items   []*ItemGui
	itemMap map[int64]weak.Pointer[ItemGui]

	dynamicList *containers.DynamicList
}

func (r *listGuiRenderer) MinSize() fyne.Size {
	return r.Container.MinSize()
}

func (r *listGuiRenderer) Layout(size fyne.Size) {
	p := theme.Padding()

	r.RoomName.Resize(r.RoomName.MinSize())
	if len(r.items) == 0 {
		r.RoomName.Move(fyne.NewPos((size.Width-r.RoomName.MinSize().Width)/2, size.Height/2-r.RoomName.MinSize().Height))

		r.EmptyTip.Resize(r.EmptyTip.MinSize())
		r.EmptyTip.Move(fyne.NewPos((size.Width-r.EmptyTip.MinSize().Width)/2, size.Height/2+theme.Padding()))
	} else {
		r.RoomName.Move(fyne.NewPos(p, p))
	}

	r.Container.Resize(fyne.NewSize(size.Width, size.Height-playlistTopHeight))
	r.Container.Move(fyne.NewPos(0, playlistTopHeight))

	r.Background.Resize(size)
	r.Background.Move(fyne.NewSquareOffsetPos(0))
}

func (r *listGuiRenderer) updateItems() {
	songs := r.list.pl.GetItemsSnapshot()

	r.items = lo.Map(songs, func(ps *song.PreloadedSong, _ int) *ItemGui {
		if item, ok := r.itemMap[ps.ID]; ok {
			if v := item.Value(); v != nil {
				if v.ps == ps {
					return v
				}
			}
		}
		newGui := NewItemGui(ps, r.dynamicList)
		r.itemMap[ps.ID] = weak.Make(newGui)
		r.dynamicList.AddItem(newGui.listItem, true)
		return newGui
	})

	if len(r.items) == 0 {
		r.EmptyTip.Show()
		r.RoomName.TextSize = 18
	} else {
		r.EmptyTip.Hide()
		r.RoomName.TextSize = 14
	}

	r.dynamicList.SetOrder(lo.Map(r.items, func(item *ItemGui, _ int) int64 {
		return item.ps.ID
	}))

	r.Container.Refresh()
}

func (r *listGuiRenderer) Refresh() {
	if r.list.itemChanged {
		r.list.itemChanged = false
		r.updateItems()
	}

	if r.list.roomChanged {
		r.list.roomChanged = false
		r.RoomName.Text = r.list.pl.RoomBrand
	}

	canvas.Refresh(r.list)
}

func (r *listGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.Background,
		r.Container,
		r.RoomName,
		r.EmptyTip,
	}
}

func (r *listGuiRenderer) Destroy() {
	close(r.stopCh)
}
