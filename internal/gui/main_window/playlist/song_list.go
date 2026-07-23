package playlist

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets/interactive_widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/catalog"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

var pypyLoadedEm = utils.NewEventManager[bool]()
var wannaLoadedEm = utils.NewEventManager[bool]()
var duduLoadedEm = utils.NewEventManager[bool]()

type SongListButton struct {
	button.PaddedIconBtn
}

func NewSongListButton() *SongListButton {
	wholeContent := container.NewVBox(
		NewCatalogState("PyPyDance", catalog.GetPyPyDanceCatalogManager(), pypyLoadedEm),
		NewCatalogState("WannaDance", catalog.GetWannaDanceCatalogManager(), wannaLoadedEm),
		NewCatalogState("DuDuFitDance", catalog.GetDuDuFitDanceCatalogManager(), duduLoadedEm),
	)

	scroll := container.NewVScroll(container.NewPadded(wholeContent))
	scroll.SetMinSize(fyne.NewSize(250, 300))

	btn := &SongListButton{}
	btn.Extend(nil)
	btn.SetComplete(false)

	btn.OnClick = func() {
		openSongListModal(scroll)
	}

	btn.ExtendBaseWidget(btn)
	btn.AddLifeCycleFn(btn.loop)

	return btn
}

func (b *SongListButton) loop(stopCh <-chan struct{}) {
	pypyCh := pypyLoadedEm.SubscribeEvent()
	defer pypyCh.Close()
	wannaCh := wannaLoadedEm.SubscribeEvent()
	defer wannaCh.Close()
	duduCh := duduLoadedEm.SubscribeEvent()
	defer duduCh.Close()

	pypyComplete := false
	wannaComplete := false
	duduComplete := false

	for {
		select {
		case <-stopCh:
			return
		case pypyComplete = <-pypyCh.Channel:
			b.SetComplete(pypyComplete && wannaComplete && duduComplete)
			fyne.Do(b.Refresh)
		case wannaComplete = <-wannaCh.Channel:
			b.SetComplete(pypyComplete && wannaComplete && duduComplete)
			fyne.Do(b.Refresh)
		case duduComplete = <-duduCh.Channel:
			b.SetComplete(pypyComplete && wannaComplete && duduComplete)
			fyne.Do(b.Refresh)
		}
	}
}

func (b *SongListButton) SetComplete(complete bool) {
	if complete {
		b.SetIcon(theme.NewColoredResource(icons.GetIcon("song-list"), theme.ColorNameSuccess))
	} else {
		b.SetIcon(theme.NewColoredResource(icons.GetIcon("song-list"), theme.ColorNameWarning))
	}
}

func openSongListModal(content fyne.CanvasObject) {
	dialog.NewCustom(
		i18n.T("message_title_song_catalog"),
		i18n.T("btn_close"),
		content,
		custom_fyne.GetParent(),
	).Show()
}

type CatalogState[T any] struct {
	interactive_widgets.LifeCycleWidget

	name string

	manager catalog.Manager[T]
	em      *utils.EventManager[bool]

	refreshCh chan struct{}

	updateTime string
	status     interactive.RemoteStatus
}

func NewCatalogState[T any](name string, manager catalog.Manager[T], em *utils.EventManager[bool]) *CatalogState[T] {
	s := &CatalogState[T]{
		name:    name,
		manager: manager,
		em:      em,

		refreshCh: make(chan struct{}),
	}

	s.ExtendBaseWidget(s)
	s.AddLifeCycleFn(s.loop)

	return s
}

func (s *CatalogState[T]) loop(stopCh <-chan struct{}) {
	handle := s.manager.Handle()
	defer handle.Release()
	ch := handle.Subscribe()
	defer ch.Close()

	s.processSnap(handle.Snapshot())

	for {
		select {
		case <-stopCh:
			return
		case snap := <-ch.Channel:
			s.processSnap(snap)
		case <-s.refreshCh:
			handle.Refresh()
		}
	}
}

func (s *CatalogState[T]) processSnap(snap interactive.RemoteSnapshot[*catalog.Catalog[T]]) {
	s.status = snap.Status
	if snap.HasData {
		s.updateTime = i18n.T("label_song_catalog_version", goeasyi18n.Options{
			Data: map[string]interface{}{
				"Time": snap.Data.UpdatedTime().Format("2006-01-02 15:04:05"),
			},
		})
	} else {
		s.updateTime = i18n.T("label_song_catalog_na")
	}
	s.em.NotifySubscribers(snap.Status.Valid())
	fyne.Do(s.Refresh)
}

func (s *CatalogState[T]) CreateRenderer() fyne.WidgetRenderer {
	name := canvas.NewText(s.name, theme.Color(theme.ColorNameForeground))
	name.TextSize = 18

	status := canvas.NewText(s.status.String(), theme.Color(s.status.Color()))
	status.TextSize = 12

	version := canvas.NewText(s.updateTime, theme.Color(theme.ColorNameForeground))
	version.TextSize = 12

	err := canvas.NewText("", theme.Color(theme.ColorNameError))
	err.TextSize = 12
	if s.status.Err != nil {
		err.Text = s.status.Err.Error()
	} else {
		err.Hide()
	}

	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		s.refreshCh <- struct{}{}
	})

	box := widgets.NewCard(
		container.NewVBox(
			container.NewBorder(
				nil,
				nil,
				container.NewVBox(name, version),
				container.NewVBox(
					refreshBtn,
				),
			),
			status,
			err,
		),
	)

	objects := []fyne.CanvasObject{box}

	r := &catalogStateRenderer[T]{
		s: s,

		box:     box,
		objects: objects,
		name:    name,
		status:  status,
		err:     err,
		version: version,

		refreshBtn: refreshBtn,
	}
	r.Created(s)

	return r
}

type catalogStateRenderer[T any] struct {
	interactive_widgets.BaseLifeCycleRenderer

	s *CatalogState[T]

	box fyne.CanvasObject

	objects []fyne.CanvasObject

	name    *canvas.Text
	status  *canvas.Text
	version *canvas.Text
	err     *canvas.Text

	refreshBtn *widget.Button
}

func (r *catalogStateRenderer[T]) Layout(size fyne.Size) {
	r.box.Move(fyne.NewPos(0, 0))
	r.box.Resize(size)
}

func (r *catalogStateRenderer[T]) MinSize() fyne.Size {
	return r.box.MinSize()
}

func (r *catalogStateRenderer[T]) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *catalogStateRenderer[T]) Refresh() {
	r.name.Text = r.s.name
	r.status.Text = r.s.status.String()
	r.status.Color = theme.Color(r.s.status.Color())
	r.version.Text = r.s.updateTime

	if r.s.status.Err != nil {
		r.err.Text = r.s.status.Err.Error()
		r.err.Show()
	} else {
		r.err.Hide()
	}

	if r.s.status.Fetching() {
		r.refreshBtn.Disable()
	} else {
		r.refreshBtn.Enable()
	}

	r.box.Refresh()
}
