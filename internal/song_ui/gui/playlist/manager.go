package playlist

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
)

type Manager struct {
	widget.BaseWidget

	currentList *playlist.PlayList
	newListCh   chan *playlist.PlayList
	stopCh      chan struct{}

	listChanged bool
}

func NewPlaylistManager() *Manager {
	m := &Manager{
		newListCh: playlist.SubscribeNewListEvent(),
		stopCh:    make(chan struct{}),
	}

	m.ExtendBaseWidget(m)

	return m
}

func (m *Manager) RenderLoop() {
	for {
		select {
		case <-m.stopCh:
			return
		case pl := <-m.newListCh:
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
	return &managerRender{
		manager: m,
	}
}

type managerRender struct {
	manager *Manager

	list *ListGui
}

func (m *managerRender) MinSize() fyne.Size {
	if m.list == nil {
		return fyne.NewSize(playItemMinWidth, playItemMinWidth)
	}
	return m.list.MinSize()
}

func (m *managerRender) Layout(size fyne.Size) {
	if m.list != nil {
		m.list.Resize(size)
		m.list.Move(fyne.NewPos(0, 0))
	}
}

func (m *managerRender) Refresh() {
	if m.manager.listChanged {
		m.list = NewListGui(m.manager.currentList)
		m.manager.listChanged = false
	}
	canvas.Refresh(m.manager)
}

func (m *managerRender) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{m.list}
}

func (m *managerRender) Destroy() {
	close(m.manager.stopCh)
	close(m.manager.newListCh)
}
