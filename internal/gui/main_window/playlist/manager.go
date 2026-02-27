package playlist

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
)

type Manager struct {
	widget.BaseWidget

	currentList *playlist.PlayList

	listChanged bool
}

func NewPlaylistManager() *Manager {
	m := &Manager{
		currentList: playlist.GetCurrentPlaylist(),
	}

	m.ExtendBaseWidget(m)

	return m
}

func (m *Manager) RenderLoop(stopCh chan struct{}) {
	ch := playlist.SubscribeNewListEvent()
	defer ch.Close()

	for {
		select {
		case <-stopCh:
			return
		case pl := <-ch.Channel:
			m.currentList = pl
			m.listChanged = true
			fyne.Do(func() {
				m.Refresh()
			})
		}
	}
}

func (m *Manager) CreateRenderer() fyne.WidgetRenderer {
	var list *ListGui
	if m.currentList != nil {
		list = NewListGui(m.currentList)
	}

	statusBar := container.NewHBox(
		NewDownloaderStatus(),
		NewSongListButton(),
		NewBroadcastButton(),
	)

	r := &managerRender{
		manager: m,
		stopCh:  make(chan struct{}),
		list:    list,

		statusBar: statusBar,
	}
	go m.RenderLoop(r.stopCh)

	return r
}

type managerRender struct {
	manager *Manager

	stopCh chan struct{}

	list *ListGui

	statusBar *fyne.Container
}

func (r *managerRender) MinSize() fyne.Size {
	if r.list == nil {
		return fyne.NewSize(playItemMinWidth, playItemMinWidth)
	}
	return r.list.MinSize()
}

func (r *managerRender) Layout(size fyne.Size) {
	p := theme.Padding()

	if r.list != nil {
		r.list.Resize(size)
		r.list.Move(fyne.NewPos(0, 0))
	}

	barSize := r.statusBar.MinSize()
	r.statusBar.Resize(barSize)
	r.statusBar.Move(fyne.NewPos(size.Width-barSize.Width-p, p))
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
	r.statusBar.Refresh()
	canvas.Refresh(r.manager)
}

func (r *managerRender) Objects() []fyne.CanvasObject {
	if r.list == nil {
		return []fyne.CanvasObject{}
	}
	return []fyne.CanvasObject{
		r.list,
		r.statusBar,
	}
}

func (r *managerRender) Destroy() {
	close(r.stopCh)
}
