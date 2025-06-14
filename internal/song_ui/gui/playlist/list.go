package playlist

import (
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
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

	itemChanged bool
	roomChanged bool
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
	for {
		select {
		case <-l.StopCh:
			return
		case change := <-l.changeCh:
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

	roomName := canvas.NewText(l.pl.RoomName, theme.Color(theme.ColorNamePlaceHolder))
	roomName.TextSize = 18
	roomName.TextStyle.Bold = true

	emptyTip := canvas.NewText(i18n.T("tip_playlist_empty"), theme.Color(theme.ColorNamePlaceHolder))
	emptyTip.TextSize = 12

	r := &listGuiRenderer{
		list: l,

		Container: scroll,

		RoomName: roomName,
		EmptyTip: emptyTip,

		itemMap: make(map[string]weak.Pointer[ItemGui]),

		dynamicList: dynamicList,
	}

	r.updateItems()

	go l.RenderLoop()

	return r
}

var playlistTopHeight = float32(30)

type listGuiRenderer struct {
	list *ListGui

	Container *container.Scroll

	RoomName *canvas.Text
	EmptyTip *canvas.Text

	items   []*ItemGui
	itemMap map[string]weak.Pointer[ItemGui]

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
}

func (r *listGuiRenderer) updateItems() {
	songs := r.list.pl.GetItemsSnapshot()

	r.items = lo.Map(songs, func(ps *song.PreloadedSong, _ int) *ItemGui {
		if item, ok := r.itemMap[ps.GetId()]; ok {
			if v := item.Value(); v != nil {
				if v.Staled {
					v.Replaced = true
				} else {
					return v
				}
			}
		}
		newGui := NewItemGui(ps, r.dynamicList)
		r.itemMap[ps.GetId()] = weak.Make(newGui)
		r.dynamicList.AddItem(newGui.listItem)

		newGui.Hide()
		go func() {
			time.Sleep(100 * time.Millisecond)
			newGui.SlideIn()
		}()
		return newGui
	})

	if len(r.items) == 0 {
		r.EmptyTip.Show()
		r.RoomName.TextSize = 18
	} else {
		r.EmptyTip.Hide()
		r.RoomName.TextSize = 14
	}

	r.dynamicList.SetOrder(lo.Map(r.items, func(item *ItemGui, _ int) string {
		return item.ps.GetId()
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
		r.RoomName.Text = r.list.pl.RoomName
	}

	canvas.Refresh(r.list)
}

func (r *listGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.Container,
		r.RoomName,
		r.EmptyTip,
	}
}

func (r *listGuiRenderer) Destroy() {
	close(r.list.StopCh)
}
