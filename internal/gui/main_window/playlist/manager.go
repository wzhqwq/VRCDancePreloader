package playlist

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type Manager struct {
	widget.BaseWidget

	currentList *playlist.PlayList

	listUpdate *utils.EventSubscriber[*playlist.PlayList]
	stopCh     chan struct{}

	listChanged bool
}

func NewPlaylistManager() *Manager {
	m := &Manager{
		listUpdate:  playlist.SubscribeNewListEvent(),
		stopCh:      make(chan struct{}),
		currentList: playlist.GetCurrentPlaylist(),
	}

	m.ExtendBaseWidget(m)

	return m
}

func (m *Manager) RenderLoop() {
	for {
		select {
		case <-m.stopCh:
			return
		case pl := <-m.listUpdate.Channel:
			m.currentList = pl
			m.listChanged = true
			fyne.Do(func() {
				m.Refresh()
			})
		}
	}
}

func (m *Manager) CreateRenderer() fyne.WidgetRenderer {
	go m.RenderLoop()

	var list *ListGui
	if m.currentList != nil {
		list = NewListGui(m.currentList)
	}

	return &managerRender{
		manager: m,
		list:    list,
	}
}

type managerRender struct {
	manager *Manager

	list *ListGui
}

func (r *managerRender) MinSize() fyne.Size {
	if r.list == nil {
		return fyne.NewSize(playItemMinWidth, playItemMinWidth)
	}
	return r.list.MinSize()
}

func (r *managerRender) Layout(size fyne.Size) {
	if r.list != nil {
		r.list.Resize(size)
		r.list.Move(fyne.NewPos(0, 0))
	}
}

func (r *managerRender) Refresh() {
	if r.manager.listChanged {
		r.manager.listChanged = false
		if r.manager.currentList == nil {
			r.list = nil
		} else {
			r.list = NewListGui(r.manager.currentList)
		}
	}
	canvas.Refresh(r.manager)
}

func (r *managerRender) Objects() []fyne.CanvasObject {
	if r.list == nil {
		return []fyne.CanvasObject{}
	}
	return []fyne.CanvasObject{r.list}
}

func (r *managerRender) Destroy() {
	close(r.manager.stopCh)
	r.manager.listUpdate.Close()
}
