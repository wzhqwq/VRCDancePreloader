package cache_gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
)

type LocalFilesGui struct {
	widget.BaseWidget

	Scroll      *container.Scroll
	List        *fyne.Container
	Label       *canvas.Text
	RefreshBtn  *widgets.PaddedIconBtn
	ProgressBar *widgets.SizeProgressBar

	localFileUpdateCh chan string
	allowListUpdateCh chan string
}

func NewLocalFilesGui() *LocalFilesGui {
	list := container.NewVBox()

	label := canvas.NewText(i18n.T("label_cache_local"), theme.Color(theme.ColorNameForeground))
	label.TextSize = 14

	refreshBtn := widgets.NewPaddedIconBtn(theme.ViewRefreshIcon())
	refreshBtn.SetMinSquareSize(30)

	progressBar := widgets.NewSizeProgressBar(cache.GetMaxSize(), 0)

	g := &LocalFilesGui{
		Scroll:      container.NewVScroll(list),
		List:        list,
		Label:       label,
		RefreshBtn:  refreshBtn,
		ProgressBar: progressBar,

		localFileUpdateCh: cache.SubscribeLocalFileEvent(),
		allowListUpdateCh: persistence.GetAllowList().SubscribeEvent(),
	}

	refreshBtn.OnClick = func() {
		g.RefreshFiles()
	}

	g.ExtendBaseWidget(g)

	g.RefreshFiles()

	go func() {
		for {
			select {
			case _, ok := <-g.localFileUpdateCh:
				if !ok {
					return
				}
				g.RefreshFiles()
			case _, ok := <-g.allowListUpdateCh:
				if !ok {
					return
				}
				g.RefreshFiles()
			}
		}
	}()

	return g
}

func (g *LocalFilesGui) CreateRenderer() fyne.WidgetRenderer {
	return &LocalFilesGuiRenderer{
		g: g,
	}
}

func (g *LocalFilesGui) RefreshFiles() {
	g.List.RemoveAll()
	infos := cache.GetLocalCacheInfos()
	totalSize := int64(0)
	for _, info := range infos {
		g.List.Add(NewLocalFileGui(info, false))
		totalSize += info.Size
	}
	g.List.Refresh()
	g.ProgressBar.SetCurrentSize(totalSize)
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

	progressX := r.g.Label.MinSize().Width + p*2
	progressWidth := size.Width - progressX - btnSize - p*2
	r.g.ProgressBar.Resize(fyne.NewSize(progressWidth, btnSize))
	r.g.ProgressBar.Move(fyne.NewPos(progressX, p/2))

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
		r.g.ProgressBar,
	}
}

func (r *LocalFilesGuiRenderer) Destroy() {
	if r.g.localFileUpdateCh != nil {
		close(r.g.localFileUpdateCh)
		r.g.localFileUpdateCh = nil
	}
	if r.g.allowListUpdateCh != nil {
		close(r.g.allowListUpdateCh)
		r.g.allowListUpdateCh = nil
	}
}

type AllowListGui struct {
	widget.BaseWidget

	Scroll     *container.Scroll
	List       *fyne.Container
	Label      *canvas.Text
	RefreshBtn *widgets.PaddedIconBtn

	updateCh chan string
}

func NewAllowListGui() *AllowListGui {
	list := container.NewVBox()

	label := canvas.NewText(i18n.T("label_cache_allow_list"), theme.Color(theme.ColorNameForeground))
	label.TextSize = 14

	refreshBtn := widgets.NewPaddedIconBtn(theme.ViewRefreshIcon())
	refreshBtn.SetMinSquareSize(30)

	g := &AllowListGui{
		Scroll:     container.NewVScroll(list),
		List:       list,
		Label:      label,
		RefreshBtn: refreshBtn,

		updateCh: persistence.GetAllowList().SubscribeEvent(),
	}

	refreshBtn.OnClick = func() {
		g.RefreshFiles()
	}

	g.ExtendBaseWidget(g)

	g.RefreshFiles()

	go func() {
		for {
			select {
			case _, ok := <-g.updateCh:
				if !ok {
					return
				}
				g.RefreshFiles()
			}
		}
	}()

	return g
}

func (g *AllowListGui) CreateRenderer() fyne.WidgetRenderer {
	return &AllowListGuiRenderer{
		g: g,
	}
}

func (g *AllowListGui) RefreshFiles() {
	g.List.RemoveAll()
	entries := persistence.GetAllowListEntries()
	for _, info := range entries {
		g.List.Add(NewLocalFileGui(info, true))
	}
	g.List.Refresh()
}

type AllowListGuiRenderer struct {
	g *AllowListGui
}

func (r *AllowListGuiRenderer) MinSize() fyne.Size {
	return fyne.NewSize(400, 400)
}

func (r *AllowListGuiRenderer) Layout(size fyne.Size) {
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

func (r *AllowListGuiRenderer) Refresh() {
	r.g.Scroll.Refresh()
}

func (r *AllowListGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.g.Scroll,
		r.g.Label,
		r.g.RefreshBtn,
	}
}

func (r *AllowListGuiRenderer) Destroy() {
	if r.g.updateCh != nil {
		close(r.g.updateCh)
		r.g.updateCh = nil
	}
}
