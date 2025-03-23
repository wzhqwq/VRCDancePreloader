package cache_gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/window_app"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

var openedWindow fyne.Window

func OpenCacheWindow() {
	if openedWindow != nil {
		return
	}

	openedWindow = window_app.NewWindow(i18n.T("label_cache_local"))
	localFiles := NewLocalFilesGui()

	splitContainer := container.NewGridWithColumns(2, localFiles, widget.NewSeparator())

	openedWindow.SetContent(splitContainer)
	openedWindow.Show()
	openedWindow.SetOnClosed(func() {
		openedWindow = nil
	})
}

type LocalFilesGui struct {
	widget.BaseWidget

	Scroll     *container.Scroll
	List       *fyne.Container
	Label      *canvas.Text
	RefreshBtn *widgets.PaddedIconBtn
}

func NewLocalFilesGui() *LocalFilesGui {
	list := container.NewVBox()

	label := canvas.NewText(i18n.T("label_cache_local"), theme.Color(theme.ColorNameForeground))
	label.TextSize = 14

	refreshBtn := widgets.NewPaddedIconBtn(theme.ViewRefreshIcon())
	refreshBtn.SetMinSquareSize(30)

	g := &LocalFilesGui{
		Scroll:     container.NewVScroll(list),
		List:       list,
		Label:      label,
		RefreshBtn: refreshBtn,
	}

	refreshBtn.OnClick = func() {
		g.RefreshFiles()
	}

	g.ExtendBaseWidget(g)

	g.RefreshFiles()

	return g
}

func (g *LocalFilesGui) CreateRenderer() fyne.WidgetRenderer {
	return &LocalFilesGuiRenderer{
		g: g,
	}
}

func (g *LocalFilesGui) RefreshFiles() {
	g.List.RemoveAll()
	ids := cache.GetLocalCacheInfos()
	for _, info := range ids {
		g.List.Add(NewLocalFileGui(info.ID, info.Size, info.IsActive))
	}
	g.List.Refresh()
}

type LocalFilesGuiRenderer struct {
	g *LocalFilesGui
}

func (r *LocalFilesGuiRenderer) MinSize() fyne.Size {
	return fyne.NewSize(400, 400)
}

func (r *LocalFilesGuiRenderer) Layout(size fyne.Size) {
	p := theme.Padding()
	btnSize := r.g.RefreshBtn.MinSize().Height
	topHeight := btnSize + p

	labelHeight := r.g.Label.MinSize().Height

	r.g.Label.Resize(r.g.Label.MinSize())
	r.g.Label.Move(fyne.NewPos(p, (topHeight-labelHeight)/2))
	r.g.RefreshBtn.Resize(r.g.RefreshBtn.MinSize())
	r.g.RefreshBtn.Move(fyne.NewPos(size.Width-btnSize-p/2, p/2))

	r.g.Scroll.Resize(fyne.NewSize(size.Width, size.Height-topHeight-theme.Padding()))
	r.g.Scroll.Move(fyne.NewPos(0, topHeight+theme.Padding()))
}

func (r *LocalFilesGuiRenderer) Refresh() {
	r.g.Scroll.Refresh()
}

func (r *LocalFilesGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.g.Scroll,
		r.g.Label,
		r.g.RefreshBtn,
	}
}

func (r *LocalFilesGuiRenderer) Destroy() {

}
